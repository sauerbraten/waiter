package game

import (
	"time"

	"github.com/sauerbraten/waiter/pkg/protocol/nmc"
)

type casual struct {
	t *Timer
	s Server
}

var _ Timed = &casual{}

func NewCasual(s Server) casual {
	return casual{
		s: s,
	}
}

func (c *casual) Start() {
	c.t = StartTimer(c.s.GameDuration(), c.s.Intermission)
	c.s.Broadcast(nmc.TimeLeft, c.s.GameDuration())
}

func (c *casual) ConfirmSpawn(*Player) {}

func (c *casual) Pause(p *Player) {
	cn := -1
	if p != nil {
		cn = int(p.CN)
	}
	c.s.Broadcast(nmc.PauseGame, 1, cn)
	c.t.Pause()
}

func (c *casual) Paused() bool { return c.t.Paused() }

func (c *casual) Resume(p *Player) {
	cn := -1
	if p != nil {
		cn = int(p.CN)
	}
	c.s.Broadcast(nmc.PauseGame, 0, cn)
	c.t.Resume()
}

func (c *casual) Leave(*Player) {}

func (c *casual) End() {
	c.s.Broadcast(nmc.TimeLeft, 0)
	c.t.Stop()
}

func (c *casual) Ended() bool { return c.t.Stopped() }

func (c *casual) CleanUp() {
	if c.Paused() {
		c.Resume(nil)
	}
	c.t.Stop()
}

func (c *casual) TimeLeft() time.Duration { return c.t.TimeLeft }

func (c *casual) SetTimeLeft(d time.Duration) {
	c.t.TimeLeft = d
	c.s.Broadcast(nmc.TimeLeft, d)
}
