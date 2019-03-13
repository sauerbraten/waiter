package main

import (
	"github.com/sauerbraten/waiter/internal/game"
	"github.com/sauerbraten/waiter/pkg/definitions/gamemode"
)

func NewGame(id gamemode.ID) game.Mode {
	mode := func() game.Mode {
		switch id {
		case gamemode.Insta:
			return game.NewInsta(s)
		case gamemode.InstaTeam:
			return game.NewInstaTeam(s, s.KeepTeams)
		case gamemode.Effic:
			return game.NewEffic(s)
		case gamemode.EfficTeam:
			return game.NewEfficTeam(s, s.KeepTeams)
		case gamemode.Tactics:
			return game.NewTactics(s)
		case gamemode.TacticsTeam:
			return game.NewTacticsTeam(s, s.KeepTeams)
		case gamemode.InstaCTF:
			return game.NewInstaCTF(s, s.KeepTeams)
		case gamemode.EfficCTF:
			return game.NewEfficCTF(s, s.KeepTeams)
		default:
			return nil
		}
	}()

	if timed, ok := mode.(game.TimedMode); ok && s.CompetitiveMode {
		return game.NewCompetitive(s, timed)
	} else {
		return mode
	}
}
