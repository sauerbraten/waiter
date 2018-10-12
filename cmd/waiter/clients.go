package main

import (
	"log"

	"github.com/sauerbraten/waiter/cubecode"
	"github.com/sauerbraten/waiter/internal/client/playerstate"
	"github.com/sauerbraten/waiter/internal/client/privilege"
	"github.com/sauerbraten/waiter/internal/definitions/disconnectreason"
	"github.com/sauerbraten/waiter/internal/definitions/mastermode"
	"github.com/sauerbraten/waiter/internal/definitions/nmc"
	"github.com/sauerbraten/waiter/internal/protocol"
	"github.com/sauerbraten/waiter/internal/protocol/enet"
	"github.com/sauerbraten/waiter/internal/protocol/packet"
)

type ClientManager struct {
	cs []*Client
}

// Links an ENet peer to a client object. If no unused client object can be found, a new one is created and added to the global set of clients.
func (cm *ClientManager) Add(peer *enet.Peer) *Client {
	// re-use unused client object with low cn
	for _, c := range cm.cs {
		if !c.InUse {
			c.Peer = peer
			c.InUse = true
			return c
		}
	}

	cn := uint32(len(cm.cs))
	c := NewClient(cn, peer)
	cm.cs = append(cm.cs, c)
	return c
}

func (cm *ClientManager) GetClientByCN(cn uint32) *Client {
	if int(cn) < 0 || int(cn) >= len(cm.cs) {
		return nil
	}
	return cm.cs[cn]
}

func (cm *ClientManager) GetClientByPeer(peer *enet.Peer) *Client {
	if peer == nil {
		return nil
	}

	for _, c := range cm.cs {
		if c.Peer == peer {
			return c
		}
	}

	return nil
}

// Sends a packet to a client over the specified channel.
func (cm *ClientManager) Send(to *Client, channel uint8, flags enet.PacketFlag, p []byte) {
	to.Peer.Send(channel, flags, p)
}

// Send a packet to a client's team, but not the client himself, over the specified channel.
func (cm *ClientManager) SendToTeam(c *Client, channel uint8, flags enet.PacketFlag, args ...interface{}) {
	excludeSelfAndOtherTeams := func(_c *Client) bool {
		return _c == c || _c.Team != c.Team
	}
	cm.Broadcast(excludeSelfAndOtherTeams, channel, flags, args...)
}

// Sends a packet to all clients currently in use.
func (cm *ClientManager) Broadcast(exclude func(*Client) bool, channel uint8, flags enet.PacketFlag, args ...interface{}) {
	for _, c := range cm.cs {
		if !c.InUse || (exclude != nil && exclude(c)) {
			continue
		}
		cm.Send(c, channel, flags, packet.Encode(args...))
	}
}

func exclude(c *Client) func(*Client) bool {
	return func(_c *Client) bool {
		return _c == c
	}
}

func (cm *ClientManager) Relay(from *Client, channel uint8, flags enet.PacketFlag, args ...interface{}) {
	cm.Broadcast(exclude(from), channel, flags, args...)
}

// Sends basic server info to the client.
func (cm *ClientManager) SendServerConfig(c *Client, config *Config) {
	p := packet.Encode(
		nmc.ServerInfo,
		c.CN,
		protocol.Version,
		c.SessionID,
		config.ServerPassword != "",
		config.ServerDescription,
		config.AuthDomains[0],
	)

	cm.Send(c, 1, enet.PACKET_FLAG_RELIABLE, p)
}

// Sends 'welcome' information to a newly joined client like map, mode, time left, other players, etc.
func (cm *ClientManager) SendWelcome(c *Client) {
	p := []interface{}{
		nmc.Welcome,
		nmc.MapChange, s.Map, s.GameMode, s.NotGotItems, // currently played mode & map
		nmc.TimeLeft, s.TimeLeft / 1000, // time left in this round
	}

	// send list of clients which have privilege higher than PRIV_NONE and their respecitve privilege level
	p = append(p, cm.PrivilegedUsersPacket())

	// tell the client what team he was put in by the server
	p = append(p, nmc.SetTeam, c.CN, c.Team, -1)

	// tell the client how to spawn (what health, what armour, what weapons, what ammo, etc.)
	if c.GameState.State == playerstate.Spectator {
		p = append(p, nmc.Spectator, c.CN, 1)
	} else {
		// TODO: handle spawn delay (e.g. in ctf modes)
		p = append(p, nmc.SpawnState, c.CN, c.GameState.ToWire())
	}

	// send other players' state (frags, flags, etc.)
	p = append(p, nmc.Resume)
	for _, client := range cm.cs {
		if client != c && client.InUse {
			p = append(p, client.CN, client.GameState.State, client.GameState.Frags, client.GameState.Flags, client.GameState.QuadTimeLeft, client.GameState.ToWire())
		}
	}
	p = append(p, -1)

	// send other client's state (name, team, playermodel)
	for _, client := range cm.cs {
		if client != c && client.InUse {
			p = append(p, nmc.InitializeClient, client.CN, client.Name, client.Team, client.PlayerModel)
		}
	}

	cm.Send(c, 1, enet.PACKET_FLAG_RELIABLE, packet.Encode(p...))
}

// Puts a client into the current game, using the data the client provided with his N_JOIN packet.
func (cm *ClientManager) Join(c *Client, name string, playerModel int32) {
	c.Joined = true
	c.Name = name
	c.PlayerModel = playerModel

	c.GameState.Spawn(s.GameMode)

	if s.MasterMode == mastermode.Locked {
		c.GameState.State = playerstate.Spectator
	} else {
		c.GameState.State = playerstate.Alive
	}

	log.Printf("join: %s (%d)\n", name, c.CN)
}

// For when a client disconnects deliberately.
func (cm *ClientManager) Leave(c *Client) {
	log.Printf("left: %s (%d)\n", c.Name, c.CN)
	cm.Disconnect(c, disconnectreason.None)
}

// Tells other clients that the client disconnected, giving a disconnect reason in case it's not a normal leave.
func (cm *ClientManager) Disconnect(c *Client, reason disconnectreason.DisconnectReason) {
	if !c.InUse {
		return
	}

	// inform others
	cm.InformOthersOfDisconnect(c, reason)

	if reason != disconnectreason.None {
		log.Printf("disconnected: %s (%d) - %s", c.Name, c.CN, disconnectreason.String[reason])
	}

	c.Peer.Disconnect(uint32(reason))

	c.Reset()
}

// Informs all other clients that a client joined the game.
func (cm *ClientManager) InformOthersOfJoin(c *Client) {
	cm.Broadcast(exclude(c), 1, enet.PACKET_FLAG_RELIABLE, nmc.InitializeClient, c.CN, c.Name, c.Team, c.PlayerModel)
	if c.GameState.State == playerstate.Spectator {
		cm.Broadcast(exclude(c), 1, enet.PACKET_FLAG_RELIABLE, nmc.Spectator, c.CN, 1)
	}
}

// Informs all other clients that a client left the game.
func (cm *ClientManager) InformOthersOfDisconnect(c *Client, reason disconnectreason.DisconnectReason) {
	cm.Broadcast(exclude(c), 1, enet.PACKET_FLAG_RELIABLE, nmc.Leave, c.CN)
	// TOOD: send a server message with the disconnect reason in case it's not a normal leave
}

func (cm *ClientManager) MapChange() {
	for _, c := range cm.cs {
		if !c.InUse {
			continue
		}
		c.GameState.Reset()
		c.GameState.Spawn(s.GameMode)
		c.Peer.Send(1, enet.PACKET_FLAG_RELIABLE, packet.Encode(nmc.SpawnState, c.CN, c.GameState.ToWire()))
	}
}

func (cm *ClientManager) PrivilegedUsers() (privileged []*Client) {
	cm.ForEach(func(c *Client) {
		if c.Privilege > privilege.None {
			privileged = append(privileged, c)
		}
	})
	return
}

func (cm *ClientManager) PrivilegedUsersPacket() cubecode.Packet {
	p := []interface{}{nmc.CurrentMaster, s.MasterMode}

	cm.ForEach(func(c *Client) {
		if c.Privilege > privilege.None {
			p = append(p, c.CN, c.Privilege)
		}
	})

	p = append(p, -1)

	if len(p) <= 3 {
		return nil
	}

	return packet.Encode(p...)
}

// Returns the number of connected clients.
func (cm *ClientManager) NumberOfClientsConnected() (n int) {
	for _, c := range cm.cs {
		if !c.InUse {
			continue
		}
		n++
	}
	return
}

func (cm *ClientManager) ForEach(do func(c *Client)) {
	for _, c := range cm.cs {
		if !c.InUse {
			continue
		}
		do(c)
	}
}
