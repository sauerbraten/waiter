package main

import (
	"time"

	"github.com/sauerbraten/waiter/internal/definitions/gamemode"
	"github.com/sauerbraten/waiter/internal/definitions/nmc"
	"github.com/sauerbraten/waiter/internal/definitions/playerstate"
)

func NewGame(id gamemode.ID) GameMode {
	d := s.GameDuration
	mode := func() GameMode {
		switch id {
		case gamemode.Insta:
			return NewInsta(d)
		case gamemode.InstaTeam:
			return NewInstaTeam(d, s.KeepTeams)
		case gamemode.Effic:
			return NewEffic(d)
		case gamemode.EfficTeam:
			return NewEfficTeam(d, s.KeepTeams)
		case gamemode.Tactics:
			return NewTactics(d)
		case gamemode.TacticsTeam:
			return NewTacticsTeam(d, s.KeepTeams)
		case gamemode.InstaCTF:
			return NewInstaCTF(d, s.KeepTeams)
		case gamemode.EfficCTF:
			return NewEfficCTF(d, s.KeepTeams)
		default:
			return nil
		}
	}()

	if s.CompetitiveMode {
		return newCompetitiveMode(mode)
	} else {
		return mode
	}
}

type CompetitiveMode struct {
	GameMode
	started              bool
	mapLoadPending       map[*Client]struct{}
	pendingResumeActions []*time.Timer
}

func newCompetitiveMode(mode GameMode) *CompetitiveMode {
	return &CompetitiveMode{
		GameMode:       mode,
		mapLoadPending: map[*Client]struct{}{},
	}
}

func (g *CompetitiveMode) ToCasual() GameMode {
	return g.GameMode
}

func (g *CompetitiveMode) Start() {
	g.GameMode.Start()
	s.Clients.Broadcast(nil, nmc.ServerMessage, "waiting for all players to load the map")
	g.Pause(nil)
	s.Clients.ForEach(func(c *Client) {
		if c.GameState.State != playerstate.Spectator {
			g.mapLoadPending[c] = struct{}{}
		}
	})
}

func (g *CompetitiveMode) Resume(c *Client) {
	if len(g.pendingResumeActions) > 0 {
		for _, action := range g.pendingResumeActions {
			if action != nil {
				action.Stop()
			}
		}
		g.pendingResumeActions = nil
		s.Clients.Broadcast(nil, nmc.ServerMessage, "resuming aborted")
		return
	}

	s.Clients.Broadcast(nil, nmc.ServerMessage, "resuming in 3 seconds")
	g.pendingResumeActions = []*time.Timer{
		time.AfterFunc(1*time.Second, func() { s.Clients.Broadcast(nil, nmc.ServerMessage, "resuming in 2 seconds") }),
		time.AfterFunc(2*time.Second, func() { s.Clients.Broadcast(nil, nmc.ServerMessage, "resuming in 1 seconds") }),
		time.AfterFunc(3*time.Second, func() {
			g.GameMode.Resume(c)
			g.pendingResumeActions = nil
		}),
	}
}

func (g *CompetitiveMode) ConfirmSpawn(c *Client) {
	g.GameMode.ConfirmSpawn(c)
	if _, ok := g.mapLoadPending[c]; ok {
		delete(g.mapLoadPending, c)
		if len(g.mapLoadPending) == 0 {
			s.Clients.Broadcast(nil, nmc.ServerMessage, "all players spawned")
			g.Resume(nil)
		}
	}
}

func (g *CompetitiveMode) Leave(c *Client) {
	g.GameMode.Leave(c)
	if c.GameState.State != playerstate.Spectator {
		g.Pause(nil)
		s.Clients.Broadcast(nil, nmc.ServerMessage, "a player left the game")
	}
}
