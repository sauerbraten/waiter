package client

import (
	"log"

	"github.com/sauerbraten/waiter/internal/client/playerstate"
	"github.com/sauerbraten/waiter/internal/client/privilege"
	"github.com/sauerbraten/waiter/internal/enet"
	"github.com/sauerbraten/waiter/internal/protocol"
	"github.com/sauerbraten/waiter/internal/protocol/definitions/disconnectreason"
	"github.com/sauerbraten/waiter/internal/protocol/definitions/mastermode"
	"github.com/sauerbraten/waiter/internal/protocol/definitions/nmc"
	"github.com/sauerbraten/waiter/internal/protocol/definitions/weapon"
	"github.com/sauerbraten/waiter/internal/protocol/packet"
	"github.com/sauerbraten/waiter/internal/server"
	"github.com/sauerbraten/waiter/internal/server/config"
	"github.com/sauerbraten/waiter/internal/utils"
)

// Describes a client.
type Client struct {
	CN                      int32
	Name                    string
	Team                    string
	PlayerModel             int32
	Privilege               privilege.Privilege
	GameState               *GameState
	Joined                  bool                              // true if the player is actually in the game
	HasToAuthForConnect     bool                              // true if the server is private or demands auth-on-connect and the client has not yet joined the actual game
	ReasonWhyAuthNeeded     disconnectreason.DisconnectReason // e.g. server is in private mode
	AI                      bool                              // wether this is a bot or not
	AISkill                 int32
	InUse                   bool // true if this client's *enet.Peer is in use (i.e. the client object belongs to a connection)
	Peer                    *enet.Peer
	SessionID               int32
	Ping                    int32
	QueuedBroadcastMessages map[uint8]*packet.Packet // channel → packet
	manager                 *ClientManager
}

func newClient(cn int32, peer *enet.Peer, manager *ClientManager) *Client {
	return &Client{
		CN:        cn,
		InUse:     true,
		Peer:      peer,
		SessionID: utils.RNG.Int31(),
		Team:      "good", // TODO: select weaker team
		GameState: newGameState(),
		QueuedBroadcastMessages: map[uint8]*packet.Packet{
			0: packet.New(),
			1: packet.New(),
		},
		manager: manager,
	}
}

// Sends a packet to a client over the specified channel.
func (c *Client) Send(flags enet.PacketFlag, channel uint8, p *packet.Packet) {
	if channel == 1 {
		//log.Println(p.buf, "→", c.CN, "on channel", channel)
	}

	c.Peer.Send(p.Bytes(), flags, channel)
}

// Sends a packet to all clients but the client himself.
func (c *Client) SendToAllOthers(flags enet.PacketFlag, channel uint8, p *packet.Packet) {
	for _, client := range c.manager.clients {
		if client == c || !client.InUse {
			continue
		}
		client.Send(flags, channel, p)
	}
}

// Send a packet to a client's team, but not the client himself, over the specified channel.
func (c *Client) SendToTeam(flags enet.PacketFlag, channel uint8, p *packet.Packet) {
	for _, client := range c.manager.clients {
		if client == c || !client.InUse || client.Team != c.Team {
			continue
		}
		client.Send(flags, channel, p)
	}
}

// Sends basic server info to the client.
func (c *Client) SendServerConfig(config *config.Config) {
	p := packet.New(nmc.ServerInfo, c.CN, protocol.Version, c.SessionID)

	if config.ServerPassword != "" {
		p.Put(1)
	} else {
		p.Put(0)
	}

	p.Put(config.ServerDescription, config.ServerAuthDomains[0])

	c.Send(enet.PACKET_FLAG_RELIABLE, 1, p)
}

// Tries to let a client join the current game, using the data the client provided with his N_JOIN packet.
func (c *Client) TryToJoin(name string, playerModel int32, hash string, authDomain string, authName string, serv *server.Server) bool {
	// TODO: check server password hash

	// check for mandatory connect auth
	if c.HasToAuthForConnect {
		if authDomain != serv.Config.ServerAuthDomains[0] {
			// client has no authkey for the server domain
			// TODO: disconnect client with disconnect reason

			return false
		}
	}

	// player may join
	c.Joined = true
	c.Name = name
	c.PlayerModel = playerModel

	c.GameState.Spawn(serv.State.GameMode)

	if serv.State.MasterMode == mastermode.Locked {
		c.GameState.State = playerstate.Spectator
	}

	log.Printf("join: %s (%d)\n", name, c.CN)

	return true
}

// Sends 'welcome' information to a newly joined client like map, mode, time left, other players, etc.
func (c *Client) SendWelcome(state *server.State) {
	p := packet.New(nmc.Welcome)

	// send currently played mode & map
	p.Put(nmc.MapChange, state.Map, state.GameMode, state.NotGotItems)

	// send time left in this round
	p.Put(nmc.TimeLeft, state.TimeLeft/1000)

	// send list of clients which have privilege higher than PRIV_NONE and their respecitve privilege level
	if state.HasMaster {
		p.Put(nmc.CurrentMaster, state.MasterMode)
		c.manager.ForEach(func(client *Client) {
			if client.Privilege > privilege.None {
				p.Put(client.CN, client.Privilege)
			}
		})
		p.Put(-1)
	}

	// tell the client what team he was put in by the server
	p.Put(nmc.SetTeam, c.CN, c.Team, -1)

	// tell the client how to spawn (what health, what armour, what weapons, what ammo, etc.)
	if c.GameState.State == playerstate.Spectator {
		p.Put(nmc.Spectator, c.CN, 1)
	} else {
		// TODO: handle spawn delay (e.g. in ctf modes)
		p.Put(nmc.SpawnState, c.CN, c.GameState.ToWire())
	}

	// send other players' state (frags, flags, etc.)
	p.Put(nmc.Resume)
	c.manager.ForEach(func(client *Client) {
		if client != c && client.InUse {
			p.Put(client.CN, client.GameState.State, client.GameState.Frags, client.GameState.Flags, client.GameState.QuadTimeLeft, client.GameState.ToWire())
		}
	})
	p.Put(-1)

	// send other client's state (name, team, playermodel)
	c.manager.ForEach(func(client *Client) {
		if client != c && client.InUse {
			p.Put(nmc.InitializeClient, client.CN, client.Name, client.Team, client.PlayerModel)
		}
	})

	c.Send(enet.PACKET_FLAG_RELIABLE, 1, p)
}

// For when a client disconnects deliberately.
func (c *Client) Leave() {
	log.Printf("left: %s (%d)\n", c.Name, c.CN)
	c.Disconnect(disconnectreason.None)
}

// Tells other clients that the client disconnected, giving a disconnect reason in case it's not a normal leave.
func (c *Client) Disconnect(reason disconnectreason.DisconnectReason) {
	if !c.InUse {
		return
	}

	// inform others
	c.InformOthersOfDisconnect(reason)

	if reason != disconnectreason.None {
		log.Printf("disconnected: %s (%d) - %s", c.Name, c.CN, disconnectreason.String[reason])
	}

	c.Peer.Disconnect(uint32(reason))

	c.Reset()
}

// Informs all other clients that a client joined the game.
func (c *Client) InformOthersOfJoin() {
	c.SendToAllOthers(enet.PACKET_FLAG_RELIABLE, 1, packet.New(nmc.InitializeClient, c.CN, c.Name, c.Team, c.PlayerModel))
	if c.GameState.State == playerstate.Spectator {
		c.SendToAllOthers(enet.PACKET_FLAG_RELIABLE, 1, packet.New(nmc.Spectator, c.CN, 1))
	}
}

// Informs all other clients that a client left the game.
func (c *Client) InformOthersOfDisconnect(reason disconnectreason.DisconnectReason) {
	c.SendToAllOthers(enet.PACKET_FLAG_RELIABLE, 1, packet.New(nmc.Leave, c.CN))
	// TOOD: send a server message with the disconnect reason in case it's not a normal leave
}

// Tells the player how to spawn (with what amount of health, armmo, armour, etc.).
func (c *Client) SendSpawnState(state *server.State) {
	//client.GameState.reset()
	c.GameState.Spawn(state.GameMode)
	c.GameState.LifeSequence = (c.GameState.LifeSequence + 1) % 128

	c.Send(enet.PACKET_FLAG_RELIABLE, 1, packet.New(nmc.SpawnState, c.CN, c.GameState.ToWire()))

	c.GameState.LastSpawn = state.TimeLeft
}

// Tries to let the player spawn, returns wether that worked or not.
func (c *Client) TryToSpawn(lifeSequence int32, selectedWeapon weapon.Weapon) bool {
	if (c.GameState.State != playerstate.Alive && c.GameState.State != playerstate.Dead) || lifeSequence != c.GameState.LifeSequence || c.GameState.LastSpawn < 0 {
		// client may not spawn
		return false
	}

	c.GameState.State = playerstate.Alive
	c.GameState.SelectedWeapon = selectedWeapon
	c.GameState.LastSpawn = -1

	return true
}

// IsValidMessage hecks if this client is allowed to send a certain type of message to us.
func (c *Client) IsValidMessage(networkMessageCode nmc.NetMessCode) bool {
	if !c.Joined {
		if c.HasToAuthForConnect {
			return networkMessageCode == nmc.AUTHANS || networkMessageCode == nmc.Ping
		}
		return networkMessageCode == nmc.Join || networkMessageCode == nmc.Ping
	} else if networkMessageCode == nmc.Join || networkMessageCode == nmc.AUTHANS {
		return false
	}

	for _, soNMC := range nmc.ServerOnlyNMCs {
		if soNMC == networkMessageCode {
			return false
		}
	}

	return true
}

func (c *Client) ClearBroadcastMessageQueue(channelID uint8) {
	c.QueuedBroadcastMessages[channelID].Clear()
}

// Resets the client object. Keeps the client's CN, so low CNs can be reused.
func (c *Client) Reset() {
	log.Println("reset:", c.CN)

	c.Name = ""
	c.PlayerModel = -1
	c.Joined = false
	c.HasToAuthForConnect = false
	c.ReasonWhyAuthNeeded = disconnectreason.None
	c.AI = false
	c.AISkill = -1
	c.InUse = false
	c.SessionID = utils.RNG.Int31()
	c.Ping = 0
	c.ClearBroadcastMessageQueue(0)
	c.ClearBroadcastMessageQueue(1)

	c.GameState.Reset()
}
