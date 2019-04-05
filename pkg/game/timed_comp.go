package game

import (
	"fmt"
	"time"

	"github.com/sauerbraten/waiter/pkg/protocol/nmc"
	"github.com/sauerbraten/waiter/pkg/protocol/playerstate"
)

type competitivelyTimed struct {
	s Server
	casual
	pendingResumeActions []*time.Timer
	mapLoadPending       map[*Player]struct{}
}

func NewCompetitivelyTimed(s Server, t casual) competitivelyTimed {
	return competitivelyTimed{
		s:              s,
		casual:         t,
		mapLoadPending: map[*Player]struct{}{},
	}
}

func (c *competitivelyTimed) Start() {
	c.casual.Start()
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

func (c *competitivelyTimed) ConfirmSpawn(p *Player) {
	if _, ok := c.mapLoadPending[p]; ok {
		delete(c.mapLoadPending, p)
		if len(c.mapLoadPending) == 0 {
			c.s.Broadcast(nmc.ServerMessage, "all players spawned, starting game")
			c.Resume(nil)
		}
	}
}

func (c *competitivelyTimed) Pause(p *Player) {
	if !c.Paused() {
		c.casual.Pause(p)
	} else if len(c.pendingResumeActions) > 0 {
		// a resume is pending, cancel it
		c.Resume(p)
	}
}

func (c *competitivelyTimed) Resume(p *Player) {
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
			c.casual.Resume(p)
			c.pendingResumeActions = nil
		}),
	}
}

func (c *competitivelyTimed) Leave(p *Player) {
	if p.State != playerstate.Spectator && !c.Ended() {
		c.s.Broadcast(nmc.ServerMessage, "a player left the game")
		c.Pause(nil)
	}
}

func (c *competitivelyTimed) CleanUp() {
	if len(c.pendingResumeActions) > 0 {
		for _, action := range c.pendingResumeActions {
			if action != nil {
				action.Stop()
			}
		}
		c.pendingResumeActions = nil
	}
	c.casual.CleanUp()
}

func (c *competitivelyTimed) ToCasual() Timed { return &c.casual }
