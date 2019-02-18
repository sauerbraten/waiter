package main

import (
	"github.com/sauerbraten/waiter/internal/definitions/nmc"
	"github.com/sauerbraten/waiter/internal/definitions/playerstate"
)

type CompetitiveGame struct {
	GameMode
	mapLoadPending map[*Client]struct{}
}

func NewCompetitiveGame(mode GameMode) CompetitiveGame {
	return CompetitiveGame{
		GameMode:       mode,
		mapLoadPending: map[*Client]struct{}{},
	}
}

func (comp CompetitiveGame) Start() {
	comp.GameMode.Start()
	s.Clients.Broadcast(nil, nmc.ServerMessage, "waiting for all players to load the map")
	s.PauseGame(nil)
	s.Clients.ForEach(func(c *Client) {
		if c.GameState.State != playerstate.Spectator {
			comp.mapLoadPending[c] = struct{}{}
		}
	})
}

func (comp CompetitiveGame) ConfirmSpawn(c *Client) {
	delete(comp.mapLoadPending, c)
	if len(comp.mapLoadPending) == 0 {
		s.Clients.Broadcast(nil, nmc.ServerMessage, "all players spawned")
		s.ResumeGame(nil)
	}
}

func (comp CompetitiveGame) Leave(c *Client) {
	if c.GameState.State == playerstate.Dead || c.GameState.State == playerstate.Alive {
		s.PauseGame(nil)
		s.Clients.Broadcast(nil, nmc.ServerMessage, "a player left the game")
	}
	comp.GameMode.Leave(c)
}
