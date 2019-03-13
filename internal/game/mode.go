package game

import (
	"github.com/sauerbraten/waiter/pkg/definitions/gamemode"
	"github.com/sauerbraten/waiter/pkg/definitions/nmc"
	"github.com/sauerbraten/waiter/pkg/protocol"
)

type Mode interface {
	ID() gamemode.ID

	NeedMapInfo() bool
	Start()
	End()
	CleanUp()

	Join(*Player)
	Init(*Player) (nmc.ID, []interface{}) // may return an init packet to send to the player
	Leave(*Player)
	CanSpawn(*Player) bool
	Spawn(*Player) // sets armour, ammo, and health
	ConfirmSpawn(*Player)

	HandleFrag(fragger, victim *Player)
	HandlePacket(*Player, nmc.ID, *protocol.Packet) bool
}

// methods that are shadowed by CompetitiveMode so all modes implement GameMode
type casualMode struct {
	s Server
}

func newCasualMode(s Server) casualMode {
	return casualMode{
		s: s,
	}
}

func (*casualMode) ConfirmSpawn(*Player) {}

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
	}
	actor.Frags++
	tlm.s.Broadcast(nmc.Died, victim.CN, actor.CN, actor.Frags, actor.Team.Frags)
}
