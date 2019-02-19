package main

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/sauerbraten/waiter/internal/definitions/disconnectreason"
	"github.com/sauerbraten/waiter/internal/definitions/nmc"
	"github.com/sauerbraten/waiter/internal/definitions/playerstate"
	"github.com/sauerbraten/waiter/internal/definitions/role"
	"github.com/sauerbraten/waiter/internal/net/enet"
	"github.com/sauerbraten/waiter/internal/net/packet"
	"github.com/sauerbraten/waiter/pkg/protocol"
	"github.com/sauerbraten/waiter/pkg/protocol/cubecode"
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

func (cm *ClientManager) FindClientByName(name string) *Client {
	name = strings.ToLower(name)
	for _, c := range cm.cs {
		if strings.Contains(c.Name, name) {
			return c
		}
	}
	return nil
}

// Send a packet to a client's team, but not the client himself, over the specified channel.
func (cm *ClientManager) SendToTeam(c *Client, args ...interface{}) {
	excludeSelfAndOtherTeams := func(_c *Client) bool {
		return _c == c || _c.Team != c.Team
	}
	cm.Broadcast(excludeSelfAndOtherTeams, args...)
}

// Sends a packet to all clients currently in use.
func (cm *ClientManager) Broadcast(exclude func(*Client) bool, args ...interface{}) {
	for _, c := range cm.cs {
		if !c.InUse || (exclude != nil && exclude(c)) {
			continue
		}
		c.Send(args...)
	}
}

func exclude(c *Client) func(*Client) bool {
	return func(_c *Client) bool {
		return _c == c
	}
}

func (cm *ClientManager) Relay(from *Client, args ...interface{}) {
	cm.Broadcast(exclude(from), args...)
}

// Sends 'welcome' information to a newly joined client like map, mode, time left, other players, etc.
func (cm *ClientManager) SendWelcome(c *Client) {
	p := []interface{}{
		nmc.Welcome,
		nmc.MapChange, s.Map, s.GameMode.ID(), s.GameMode.NeedMapInfo(), // currently played mode & map
		nmc.TimeLeft, s.GameMode.TimeLeft(), // time left in this round
	}

	// send list of clients which have privilege higher than PRIV_NONE and their respecitve privilege level
	pup, empty := cm.PrivilegedUsersPacket()
	if !empty {
		p = append(p, pup)
	}

	if s.GameMode.Paused() {
		p = append(p, nmc.PauseGame, 1, -1)
	}

	if teamMode, ok := s.GameMode.(TeamMode); ok {
		p = append(p, nmc.TeamInfo)
		teamMode.ForEach(func(t *Team) {
			if t.Frags > 0 {
				p = append(p, t.Name, t.Frags)
			}
		})
		p = append(p, "")
	}

	// tell the client what team he was put in by the server
	p = append(p, nmc.SetTeam, c.CN, c.Team.Name, -1)

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
			p = append(p, nmc.InitializeClient, client.CN, client.Name, client.Team.Name, client.PlayerModel)
		}
	}

	c.Send(p...)
}

// Tells other clients that the client disconnected, giving a disconnect reason in case it's not a normal leave.
func (cm *ClientManager) Disconnect(c *Client, reason disconnectreason.ID) {
	if !c.InUse {
		return
	}

	cm.Relay(c, nmc.Leave, c.CN)

	msg := ""
	if reason != disconnectreason.None {
		msg = fmt.Sprintf("%s (%s) disconnected because: %s", cm.UniqueName(c), c.Peer.Address.IP, reason)
		cm.Relay(c, nmc.ServerMessage, msg)
	} else {
		msg = fmt.Sprintf("%s (%s) disconnected", cm.UniqueName(c), c.Peer.Address.IP)
	}
	log.Println(cubecode.SanitizeString(msg))

	c.Peer.Disconnect(uint32(reason))

	c.Reset()
}

// Informs all other clients that a client joined the game.
func (cm *ClientManager) InformOthersOfJoin(c *Client) {
	cm.Relay(c, nmc.InitializeClient, c.CN, c.Name, c.Team.Name, c.PlayerModel)
	if c.GameState.State == playerstate.Spectator {
		cm.Relay(c, nmc.Spectator, c.CN, 1)
	}
}

func (cm *ClientManager) MapChange() {
	cm.ForEach(func(c *Client) {
		c.GameState.Reset()
		if c.GameState.State == playerstate.Spectator {
			return
		}
		s.Spawn(c)
		c.Send(nmc.SpawnState, c.CN, c.GameState.ToWire())
	})
}

func (cm *ClientManager) PrivilegedUsers() (privileged []*Client) {
	cm.ForEach(func(c *Client) {
		if c.Role > role.None {
			privileged = append(privileged, c)
		}
	})
	return
}

func (cm *ClientManager) PrivilegedUsersPacket() (p protocol.Packet, noPrivilegedUsers bool) {
	q := []interface{}{nmc.CurrentMaster, s.MasterMode}

	cm.ForEach(func(c *Client) {
		if c.Role > role.None {
			q = append(q, c.CN, c.Role)
		}
	})

	q = append(q, -1)

	return packet.Encode(q...), len(q) <= 3
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

func (cm *ClientManager) UniqueName(c *Client) string {
	unique := true
	cm.ForEach(func(_c *Client) {
		if _c != c && _c.Name == c.Name {
			unique = false
		}
	})

	if !unique {
		return c.Name + cubecode.Magenta(" ("+strconv.FormatUint(uint64(c.CN), 10)+")")
	}
	return c.Name
}
