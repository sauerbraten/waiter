package server

import (
	"github.com/sauerbraten/waiter/pkg/game"
	"github.com/sauerbraten/waiter/pkg/protocol/gamemode"
)

func (s *Server) StartMode(id gamemode.ID) game.TimedMode {
	casual := game.NewCasual(s)
	var t game.Timed = &casual
	if s.CompetitiveMode {
		comp := game.NewCompetitivelyTimed(s, casual)
		t = &comp
	}

	switch id {
	case gamemode.Insta:
		return game.NewInsta(s, t)
	case gamemode.InstaTeam:
		return game.NewInstaTeam(s, s.KeepTeams, t)
	case gamemode.Effic:
		return game.NewEffic(s, t)
	case gamemode.EfficTeam:
		return game.NewEfficTeam(s, s.KeepTeams, t)
	case gamemode.Tactics:
		return game.NewTactics(s, t)
	case gamemode.TacticsTeam:
		return game.NewTacticsTeam(s, s.KeepTeams, t)
	case gamemode.InstaCTF:
		return game.NewInstaCTF(s, s.KeepTeams, t)
	case gamemode.EfficCTF:
		return game.NewEfficCTF(s, s.KeepTeams, t)
	default:
		return nil
	}
}
