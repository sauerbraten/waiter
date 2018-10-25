package main

import (
	"github.com/sauerbraten/waiter/internal/client/playerstate"
	"github.com/sauerbraten/waiter/internal/client/privilege"
	"github.com/sauerbraten/waiter/internal/definitions/disconnectreason"
	"github.com/sauerbraten/waiter/internal/definitions/nmc"
	"github.com/sauerbraten/waiter/internal/definitions/weapon"
	"github.com/sauerbraten/waiter/internal/geom"
	"github.com/sauerbraten/waiter/internal/protocol/enet"
	"github.com/sauerbraten/waiter/internal/utils"
)

// Describes a client.
type Client struct {
	CN                  uint32
	Name                string
	Team                string
	PlayerModel         int32
	Privilege           privilege.Privilege
	GameState           *GameState
	Joined              bool                              // true if the player is actually in the game
	AuthRequiredBecause disconnectreason.DisconnectReason // e.g. server is in private mode
	IsBot               bool                              // wether this is a bot or not
	BotSkill            int32
	InUse               bool // true if this client's *enet.Peer is in use (i.e. the client object belongs to a connection)
	Peer                *enet.Peer
	SessionID           int32
	Ping                int32
	Position            *Publisher
	Packets             *Publisher
}

func NewClient(cn uint32, peer *enet.Peer) *Client {
	return &Client{
		CN:        cn,
		InUse:     true,
		Peer:      peer,
		SessionID: utils.RNG.Int31(),
		Team:      "good", // TODO: select weaker team
		GameState: NewGameState(),
	}
}

// IsValidMessage hecks if this client is allowed to send a certain type of message to us.
func (c *Client) IsValidMessage(networkMessageCode nmc.NetMessCode) bool {
	if !c.Joined {
		if c.AuthRequiredBecause > disconnectreason.None {
			return networkMessageCode == nmc.AuthAnswer || networkMessageCode == nmc.Ping
		}
		return networkMessageCode == nmc.Join || networkMessageCode == nmc.Ping
	} else if networkMessageCode == nmc.Join {
		return false
	}

	if c.GameState.State != playerstate.Alive {
		for _, aoNMC := range nmc.ServerOnlyNMCs {
			if aoNMC == networkMessageCode {
				return false
			}
		}
	}

	for _, soNMC := range nmc.ServerOnlyNMCs {
		if soNMC == networkMessageCode {
			return false
		}
	}

	return true
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
	c.Privilege = privilege.None
	if c.GameState != nil {
		c.GameState.Reset()
	}
	c.Joined = false
	c.AuthRequiredBecause = disconnectreason.None
	c.IsBot = false
	c.BotSkill = -1
	c.InUse = false
	c.Peer = nil
	c.SessionID = utils.RNG.Int31()
	c.Ping = 0
	if c.Packets != nil {
		c.Packets.Close()
	}
	if c.Position != nil {
		c.Position.Close()
	}
}
