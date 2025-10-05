package game

import (
	"github.com/sauerbraten/waiter/pkg/protocol"
	"github.com/sauerbraten/waiter/pkg/protocol/gamemode"
	"github.com/sauerbraten/waiter/pkg/protocol/nmc"
)

type Mode interface {
	ID() gamemode.ID
	NeedsMapInfo() bool
	Join(*Player)
	InitPlayer() []any
	Leave(*Player)
	CanSpawn(*Player) bool
	Spawn(*PlayerState) // sets armour, ammo, and health
	HandleFrag(fragger, victim *Player)
	HandlePacket(*Player, nmc.ID, *protocol.Packet) bool
}

type noMapInfo struct{}

func (noMapInfo) NeedsMapInfo() bool                                  { return false }
func (noMapInfo) InitPlayer() []any                                   { return nil }
func (noMapInfo) HandlePacket(*Player, nmc.ID, *protocol.Packet) bool { return false }
func (noMapInfo) CleanUp()                                            {}
