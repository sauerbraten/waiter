package main

import (
	"github.com/sauerbraten/waiter/internal/definitions/gamemode"
	"github.com/sauerbraten/waiter/internal/definitions/nmc"
	"github.com/sauerbraten/waiter/internal/definitions/playerstate"
)

type Game interface {
	Mode() GameMode
	Start()
	ConfirmSpawn(*Client)
	Leave(*Client)
}

func NewGame(id gamemode.ID) Game {
	mode := func() GameMode {
		switch id {
		case gamemode.Insta:
			return NewInsta()
		case gamemode.InstaTeam:
			return NewInstaTeam(s.KeepTeams)
		case gamemode.Effic:
			return NewEffic()
		case gamemode.EfficTeam:
			return NewEfficTeam(s.KeepTeams)
		case gamemode.Tactics:
			return NewTactics()
		case gamemode.TacticsTeam:
			return NewTacticsTeam(s.KeepTeams)
		case gamemode.InstaCTF:
			return NewInstaCTF(s.KeepTeams)
		case gamemode.EfficCTF:
			return NewEfficCTF(s.KeepTeams)
		default:
			return nil
		}
	}()

	if s.CompetitiveMode {
		return NewCompetitiveGame(mode)
	} else {
		return &CasualGame{
			GameMode: mode,
		}
	}
}

type CasualGame struct {
	GameMode
}

func (g *CasualGame) Mode() GameMode { return g.GameMode }

func (*CasualGame) Start() {}

func (*CasualGame) ConfirmSpawn(*Client) {}

type CompetitiveGame struct {
	GameMode
	started        bool
	mapLoadPending map[*Client]struct{}
}

func NewCompetitiveGame(mode GameMode) *CompetitiveGame {
	return &CompetitiveGame{
		GameMode:       mode,
		mapLoadPending: map[*Client]struct{}{},
	}
}

func (g *CompetitiveGame) Mode() GameMode { return g.GameMode }

func (g *CompetitiveGame) Start() {
	s.Clients.Broadcast(nil, nmc.ServerMessage, "waiting for all players to load the map")
	s.PauseGame(nil)
	s.Clients.ForEach(func(c *Client) {
		if c.GameState.State != playerstate.Spectator {
			g.mapLoadPending[c] = struct{}{}
		}
	})
}

func (g *CompetitiveGame) ConfirmSpawn(c *Client) {
	if _, ok := g.mapLoadPending[c]; ok {
		delete(g.mapLoadPending, c)
		if len(g.mapLoadPending) == 0 {
			s.Clients.Broadcast(nil, nmc.ServerMessage, "all players spawned")
			s.ResumeGame(nil)
		}
	}
}

func (g *CompetitiveGame) Leave(c *Client) {
	if c.GameState.State == playerstate.Dead || c.GameState.State == playerstate.Alive {
		s.PauseGame(nil)
		s.Clients.Broadcast(nil, nmc.ServerMessage, "a player left the game")
	}
	g.GameMode.Leave(c)
}
