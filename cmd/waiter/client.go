package main

import (
	"fmt"

	"github.com/sauerbraten/waiter/internal/relay"

	"github.com/sauerbraten/waiter/internal/game"
	"github.com/sauerbraten/waiter/internal/net/enet"
	"github.com/sauerbraten/waiter/internal/net/packet"
	"github.com/sauerbraten/waiter/internal/utils"
	"github.com/sauerbraten/waiter/pkg/definitions/disconnectreason"
	"github.com/sauerbraten/waiter/pkg/definitions/nmc"
	"github.com/sauerbraten/waiter/pkg/definitions/role"
)

type Authentication struct {
	reqID uint32
	name  string
}

// Describes a client.
type Client struct {
	game.Player
	Role                role.ID
	Joined              bool                // true if the player is actually in the game
	AuthRequiredBecause disconnectreason.ID // e.g. server is in private mode
	InUse               bool                // true if this client's *enet.Peer is in use (i.e. the client object belongs to a connection)
	Peer                *enet.Peer
	SessionID           int32
	Ping                int32
	Positions           *relay.Publisher
	Packets             *relay.Publisher
	Authentications     map[string]*Authentication
}

func NewClient(cn uint32, peer *enet.Peer) *Client {
	return &Client{
		Player:          game.NewPlayer(cn),
		InUse:           true,
		Peer:            peer,
		SessionID:       utils.RNG.Int31(),
		Authentications: map[string]*Authentication{},
	}
}

// Resets the client object.
func (c *Client) Reset() {
	c.Player.Reset()
	c.Role = role.None
	c.Joined = false
	c.AuthRequiredBecause = disconnectreason.None
	c.InUse = false
	c.Peer = nil
	c.SessionID = utils.RNG.Int31()
	c.Ping = 0
	if c.Positions != nil {
		c.Positions.Close()
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
	c.Peer.Send(1, packet.Encode(typ, packet.Encode(args...)))
}
