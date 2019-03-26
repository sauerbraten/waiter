package game

import (
	"fmt"
	"time"

	"github.com/sauerbraten/waiter/pkg/protocol/nmc"
	"github.com/sauerbraten/waiter/pkg/protocol/playerstate"
)

type Competitive struct {
	s Server
	TimedMode
	started              bool
	mapLoadPending       map[*Player]struct{}
	pendingResumeActions []*time.Timer
}

func NewCompetitive(s Server, mode TimedMode) *Competitive {
	return &Competitive{
		s:              s,
		TimedMode:      mode,
		mapLoadPending: map[*Player]struct{}{},
	}
}

func (c *Competitive) ToCasual() TimedMode {
	return c.TimedMode
}

func (c *Competitive) Start() {
	c.TimedMode.Start()
	c.s.ForEach(func(p *Player) {
		if p.State != playerstate.Spectator {
			c.mapLoadPending[p] = struct{}{}
		}
	})
	if len(c.mapLoadPending) > 0 {
		c.s.Broadcast(nmc.ServerMessage, "waiting for all players to load the map")
		c.Pause(nil)
	}
}

func (c *Competitive) Resume(p *Player) {
	if len(c.pendingResumeActions) > 0 {
		for _, action := range c.pendingResumeActions {
			if action != nil {
				action.Stop()
			}
		}
		c.pendingResumeActions = nil
		c.s.Broadcast(nmc.ServerMessage, "resuming aborted")
		return
	}

	if p != nil {
		c.s.Broadcast(nmc.ServerMessage, fmt.Sprintf("%s wants to resume the game", c.s.UniqueName(p)))
	}
	c.s.Broadcast(nmc.ServerMessage, "resuming game in 3 seconds")
	c.pendingResumeActions = []*time.Timer{
		time.AfterFunc(1*time.Second, func() { c.s.Broadcast(nmc.ServerMessage, "resuming game in 2 seconds") }),
		time.AfterFunc(2*time.Second, func() { c.s.Broadcast(nmc.ServerMessage, "resuming game in 1 second") }),
		time.AfterFunc(3*time.Second, func() {
			c.TimedMode.Resume(p)
			c.pendingResumeActions = nil
		}),
	}
}

func (g *Competitive) ConfirmSpawn(p *Player) {
	g.TimedMode.ConfirmSpawn(p)
	if _, ok := g.mapLoadPending[p]; ok {
		delete(g.mapLoadPending, p)
		if len(g.mapLoadPending) == 0 {
			g.s.Broadcast(nmc.ServerMessage, "all players spawned, starting game")
			g.Resume(nil)
		}
	}
}

func (g *Competitive) Leave(p *Player) {
	g.TimedMode.Leave(p)
	if p.State != playerstate.Spectator && !g.TimedMode.Ended() {
		g.s.Broadcast(nmc.ServerMessage, "a player left the game")
		if !g.Paused() {
			g.Pause(nil)
		} else if len(g.pendingResumeActions) > 0 {
			// a resume is pending, cancel it
			g.Resume(nil)
		}
	}
}

func (g *Competitive) CleanUp() {
	if len(g.pendingResumeActions) > 0 {
		for _, action := range g.pendingResumeActions {
			if action != nil {
				action.Stop()
			}
		}
		g.pendingResumeActions = nil
	}
	g.TimedMode.CleanUp()
}
