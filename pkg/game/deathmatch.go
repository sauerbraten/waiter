package game

import (
	"github.com/sauerbraten/waiter/pkg/protocol/gamemode"
	"github.com/sauerbraten/waiter/pkg/protocol/nmc"
)

type deathmatch struct {
	s Server
}

func newDeathmatch(s Server) *deathmatch {
	return &deathmatch{
		s: s,
	}
}

func (m *deathmatch) HandleFrag(actor, victim *Player) {
	victim.Die()
	if actor == victim {
		actor.Frags--
	} else {
		actor.Frags++
	}
	m.s.Broadcast(nmc.Died, victim.CN, actor.CN, actor.Frags, actor.Team.Frags)
}

func (*deathmatch) Join(*Player)          {}
func (*deathmatch) CanSpawn(*Player) bool { return true }
func (*deathmatch) Leave(*Player)         {}

type FFA struct {
	*deathmatch
	ffaSpawnState
	*handlesPickups
}

func NewFFA(s Server) *FFA {
	return &FFA{
		deathmatch:     newDeathmatch(s),
		handlesPickups: handlingPickups(s),
	}
}

func (*FFA) ID() gamemode.ID { return gamemode.FFA }

type Effic struct {
	*deathmatch
	efficSpawnState
	noMapInfo
}

func NewEffic(s Server) *Effic {
	return &Effic{
		deathmatch: newDeathmatch(s),
	}
}

func (*Effic) ID() gamemode.ID { return gamemode.Effic }

type Insta struct {
	*deathmatch
	instaSpawnState
	noMapInfo
}

func NewInsta(s Server) *Insta {
	return &Insta{
		deathmatch: newDeathmatch(s),
	}
}

func (*Insta) ID() gamemode.ID { return gamemode.Insta }

type Tactics struct {
	*deathmatch
	tacticsSpawnState
	noMapInfo
}

func NewTactics(s Server) *Tactics {
	return &Tactics{
		deathmatch: newDeathmatch(s),
	}
}

func (*Tactics) ID() gamemode.ID { return gamemode.Tactics }

type teamDeathmatchMode struct {
	*teamMode
}

func newTeamDeathmatchMode(s Server, keepTeams bool) *teamDeathmatchMode {
	return &teamDeathmatchMode{
		teamMode: withTeams(s, true, keepTeams, NewTeam("good"), NewTeam("evil")),
	}
}

func (*teamDeathmatchMode) CanSpawn(*Player) bool { return true }

func (*teamDeathmatchMode) InitPacket() []any { return nil }

type Teamplay struct {
	*teamDeathmatchMode
	ffaSpawnState
	*handlesPickups
}

func NewTeamplay(s Server, keepTeams bool) *Teamplay {
	return &Teamplay{
		teamDeathmatchMode: newTeamDeathmatchMode(s, keepTeams),
		handlesPickups:     handlingPickups(s),
	}
}

func (*Teamplay) ID() gamemode.ID { return gamemode.Teamplay }

type EfficTeam struct {
	*teamDeathmatchMode
	efficSpawnState
	noMapInfo
}

func NewEfficTeam(s Server, keepTeams bool) *EfficTeam {
	return &EfficTeam{
		teamDeathmatchMode: newTeamDeathmatchMode(s, keepTeams),
	}
}

func (*EfficTeam) ID() gamemode.ID { return gamemode.EfficTeam }

type InstaTeam struct {
	*teamDeathmatchMode
	instaSpawnState
	noMapInfo
}

func NewInstaTeam(s Server, keepTeams bool) *InstaTeam {
	return &InstaTeam{
		teamDeathmatchMode: newTeamDeathmatchMode(s, keepTeams),
	}
}

func (*InstaTeam) ID() gamemode.ID { return gamemode.InstaTeam }

type TacticsTeam struct {
	*teamDeathmatchMode
	tacticsSpawnState
	noMapInfo
}

func NewTacticsTeam(s Server, keepTeams bool) *TacticsTeam {
	return &TacticsTeam{
		teamDeathmatchMode: newTeamDeathmatchMode(s, keepTeams),
	}
}

func (*TacticsTeam) ID() gamemode.ID { return gamemode.TacticsTeam }
