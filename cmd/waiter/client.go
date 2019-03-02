package main

import (
	"fmt"

	"github.com/sauerbraten/waiter/internal/geom"
	"github.com/sauerbraten/waiter/internal/net/enet"
	"github.com/sauerbraten/waiter/internal/net/packet"
	"github.com/sauerbraten/waiter/internal/utils"
	"github.com/sauerbraten/waiter/pkg/definitions/disconnectreason"
	"github.com/sauerbraten/waiter/pkg/definitions/nmc"
	"github.com/sauerbraten/waiter/pkg/definitions/role"
	"github.com/sauerbraten/waiter/pkg/definitions/weapon"
)

type Authentication struct {
	reqID uint32
	name  string
}

// Describes a client.
type Client struct {
	CN                  uint32
	Name                string
	Team                *Team
	PlayerModel         int32
	Role                role.ID
	GameState           *GameState
	Joined              bool                // true if the player is actually in the game
	AuthRequiredBecause disconnectreason.ID // e.g. server is in private mode
	InUse               bool                // true if this client's *enet.Peer is in use (i.e. the client object belongs to a connection)
	Peer                *enet.Peer
	SessionID           int32
	Ping                int32
	CurrentPos          *geom.Vector
	Position            *Publisher
	Packets             *Publisher
	Authentications     map[string]*Authentication
}

func NewClient(cn uint32, peer *enet.Peer) *Client {
	return &Client{
		CN:              cn,
		InUse:           true,
		Peer:            peer,
		SessionID:       utils.RNG.Int31(),
		Team:            NoTeam,
		GameState:       NewGameState(),
		Authentications: map[string]*Authentication{},
	}
}

func (c *Client) applyDamage(attacker *Client, damage int32, weapon weapon.ID, direction *geom.Vector) {
	c.GameState.applyDamage(damage)
	if attacker != c && attacker.Team != c.Team {
		attacker.GameState.Damage += damage
	}

	// TODO
}

func (c *Client) Die() {
	c.Position.Publish()
	c.GameState.Die()
}

// Resets the client object. Keeps the client's CN, so low CNs can be reused.
func (c *Client) Reset() {
	c.Name = ""
	c.PlayerModel = -1
	c.Role = role.None
	if c.GameState != nil {
		c.GameState.Reset()
	}
	c.Joined = false
	c.AuthRequiredBecause = disconnectreason.None
	c.InUse = false
	c.Peer = nil
	c.SessionID = utils.RNG.Int31()
	c.Ping = 0
	if c.Position != nil {
		c.Position.Close()
	}
	if c.Packets != nil {
		c.Packets.Close()
	}
	for domain := range c.Authentications {
		delete(c.Authentications, domain)
	}
}

func (c *Client) String() string {
	return fmt.Sprintf("%s (%d)", c.Name, c.CN)
}

func (c *Client) Send(typ nmc.ID, args ...interface{}) {
	c.Peer.Send(1, enet.PACKET_FLAG_RELIABLE, packet.Encode(typ, packet.Encode(args...)))
}
