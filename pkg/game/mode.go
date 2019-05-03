package game

import (
	"github.com/sauerbraten/waiter/pkg/protocol"
	"github.com/sauerbraten/waiter/pkg/protocol/gamemode"
	"github.com/sauerbraten/waiter/pkg/protocol/nmc"
)

type Mode interface {
	ID() gamemode.ID
	NeedMapInfo() bool
	Join(*Player)
	Init(*Player) (nmc.ID, []interface{}) // may return an init packet to send to the player
	CanSpawn(*Player) bool
	Spawn(*Player) // sets armour, ammo, and health
	HandleFrag(fragger, victim *Player)
	HandlePacket(*Player, nmc.ID, *protocol.Packet) bool
}

// no spawn timeout
type noSpawnWaitMode struct{}

func (*noSpawnWaitMode) CanSpawn(*Player) bool { return true }

// no pick-ups, no flags, no bases
type noItemsMode struct{}

func (*noItemsMode) NeedMapInfo() bool { return false }

func (*noItemsMode) Init(*Player) (nmc.ID, []interface{}) { return nmc.None, nil }

func (*noItemsMode) HandlePacket(*Player, nmc.ID, *protocol.Packet) bool { return false }

type teamlessMode struct {
	s Server
}

func newTeamlessMode(s Server) teamlessMode {
	return teamlessMode{
		s: s,
	}
}

func (*teamlessMode) Join(*Player) {}

func (*teamlessMode) Leave(*Player) {}

func (tlm *teamlessMode) HandleFrag(actor, victim *Player) {
	victim.Die()
	if actor == victim {
		actor.Frags--
	} else {
		actor.Frags++
	}
	tlm.s.Broadcast(nmc.Died, victim.CN, actor.CN, actor.Frags, actor.Team.Frags)
}
