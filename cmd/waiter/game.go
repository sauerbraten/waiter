package main

import (
	"fmt"
	"time"

	"github.com/sauerbraten/waiter/pkg/definitions/gamemode"
	"github.com/sauerbraten/waiter/pkg/definitions/nmc"
	"github.com/sauerbraten/waiter/pkg/definitions/playerstate"
)

func NewGame(id gamemode.ID) GameMode {
	mode := func() GameMode {
		switch id {
		case gamemode.Insta:
			return NewInsta()
		case gamemode.InstaTeam:
			return NewInstaTeam()
		case gamemode.Effic:
			return NewEffic()
		case gamemode.EfficTeam:
			return NewEfficTeam()
		case gamemode.Tactics:
			return NewTactics()
		case gamemode.TacticsTeam:
			return NewTacticsTeam()
		case gamemode.InstaCTF:
			return NewInstaCTF()
		case gamemode.EfficCTF:
			return NewEfficCTF()
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
	s.Clients.ForEach(func(c *Client) {
		if c.GameState.State != playerstate.Spectator {
			g.mapLoadPending[c] = struct{}{}
		}
	})
	if len(g.mapLoadPending) > 0 {
		s.Clients.Broadcast(nmc.ServerMessage, "waiting for all players to load the map")
		g.Pause(nil)
	}
}

func (g *CompetitiveMode) Resume(c *Client) {
	if len(g.pendingResumeActions) > 0 {
		for _, action := range g.pendingResumeActions {
			if action != nil {
				action.Stop()
			}
		}
		g.pendingResumeActions = nil
		s.Clients.Broadcast(nmc.ServerMessage, "resuming aborted")
		return
	}

	if c != nil {
		s.Clients.Broadcast(nmc.ServerMessage, fmt.Sprintf("%s wants to resume the game", s.Clients.UniqueName(c)))
	}
	s.Clients.Broadcast(nmc.ServerMessage, "resuming game in 3 seconds")
	g.pendingResumeActions = []*time.Timer{
		time.AfterFunc(1*time.Second, func() { s.Clients.Broadcast(nmc.ServerMessage, "resuming game in 2 seconds") }),
		time.AfterFunc(2*time.Second, func() { s.Clients.Broadcast(nmc.ServerMessage, "resuming game in 1 second") }),
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
			s.Clients.Broadcast(nmc.ServerMessage, "all players spawned, starting game")
			g.Resume(nil)
		}
	}
}

func (g *CompetitiveMode) Leave(c *Client) {
	g.GameMode.Leave(c)
	if c.GameState.State != playerstate.Spectator && !g.GameMode.Ended() {
		s.Clients.Broadcast(nmc.ServerMessage, "a player left the game")
		if !g.Paused() {
			g.Pause(nil)
		} else if len(g.pendingResumeActions) > 0 {
			// a resume is pending, cancel it
			g.Resume(nil)
		}
	}
}

func (g *CompetitiveMode) CleanUp() {
	if len(g.pendingResumeActions) > 0 {
		for _, action := range g.pendingResumeActions {
			if action != nil {
				action.Stop()
			}
		}
		g.pendingResumeActions = nil
	}
	g.GameMode.CleanUp()
}
