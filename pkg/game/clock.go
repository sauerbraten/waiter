package game

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/sauerbraten/timer"
	"github.com/sauerbraten/waiter/pkg/protocol/nmc"
	"github.com/sauerbraten/waiter/pkg/protocol/playerstate"
)

// implemented by modes with modes with either flags or pickups
type hasTimers interface {
	// Start() // flags/pickups "auto-start" once they are received from a client
	Pause()
	Resume()
	Stop()
	CleanUp()
}

var (
	_ hasTimers = &flagMode{}
	_ hasTimers = &handlesPickups{}
)

type casualClock struct {
	s Server
	t *timer.Timer
	m hasTimers
}

func NewCasualClock(s Server, m Mode) *casualClock {
	timedMode, _ := m.(hasTimers) // nil if mode has no timers
	return &casualClock{
		s: s,
		t: timer.AfterFunc(s.GameDuration(), s.Intermission),
		m: timedMode,
	}
}

func (c *casualClock) Start() {
	c.t.Start()
	log.Println("started game timer, time left:", c.t.TimeLeft())
	c.s.Broadcast(nmc.TimeLeft, int32(c.t.TimeLeft()/time.Second))
}

func (c *casualClock) Pause(p *Player) {
	if c.t.Paused() {
		return
	}
	cn := -1
	if p != nil {
		cn = int(p.CN)
	}
	c.s.Broadcast(nmc.PauseGame, 1, cn)
	c.t.Pause()
	if c.m != nil {
		c.m.Pause()
	}
}

func (c *casualClock) Paused() bool {
	return c.t.Paused()
}

func (c *casualClock) Resume(p *Player) {
	if !c.t.Paused() {
		return
	}
	cn := -1
	if p != nil {
		cn = int(p.CN)
	}
	c.s.Broadcast(nmc.PauseGame, 0, cn)
	c.t.Start()
	if c.m != nil {
		c.m.Resume()
	}
}

func (*casualClock) Join(*Player)      {}
func (*casualClock) Spawned(p *Player) {}
func (*casualClock) Leave(*Player)     {}

func (c *casualClock) Stop() {
	c.s.Broadcast(nmc.TimeLeft, 0)
	log.Println("stopping game timer, time left:", c.t.TimeLeft())
	c.t.Stop()
	if c.m != nil {
		c.m.Stop()
	}
}

func (c *casualClock) Ended() bool {
	return c.t.TimeLeft() <= 0
}

func (c *casualClock) TimeLeft() time.Duration {
	return c.t.TimeLeft()
}

func (c *casualClock) SetTimeLeft(d time.Duration) {
	log.Println("setting game timer to", d)
	if !c.t.SetTimeLeft(d) {
		log.Println("game timer had already expired")
	}
	c.s.Broadcast(nmc.TimeLeft, int32(d/time.Second))
}

type competitiveClock struct {
	*casualClock
	pendingResumeActions []*time.Timer
	waitingForSpawns     bool
	pendingSpawns        *sync.WaitGroup
}

func NewCompetitiveClock(s Server, m Mode) *competitiveClock {
	return &competitiveClock{
		casualClock:   NewCasualClock(s, m),
		pendingSpawns: new(sync.WaitGroup),
	}
}

func (c *competitiveClock) Start() {
	c.casualClock.Start()
	c.waitingForSpawns = false
	c.s.ForEachPlayer(func(p *Player) {
		if p.State != playerstate.Spectator {
			c.pendingSpawns.Add(1)
			c.waitingForSpawns = true
		}
	})
	if c.waitingForSpawns {
		c.s.Broadcast(nmc.ServerMessage, "waiting for all players to load the map")
		c.Pause(nil)
		go func() {
			c.pendingSpawns.Wait()
			c.waitingForSpawns = false
			c.s.Broadcast(nmc.ServerMessage, "all players spawned, starting game")
			c.Resume(nil)
		}()
	}
}

func (c *competitiveClock) Join(p *Player) {
	if p.State != playerstate.Spectator && c.waitingForSpawns {
		c.pendingSpawns.Add(1)
	}
}

func (c *competitiveClock) Spawned(p *Player) {
	c.pendingSpawns.Done()
}

func (c *competitiveClock) Pause(p *Player) {
	if !c.Paused() {
		c.casualClock.Pause(p)
	} else if len(c.pendingResumeActions) > 0 {
		// a resume is pending, cancel it
		c.Resume(p)
	}
}

func (c *competitiveClock) Resume(p *Player) {
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
	c.s.Broadcast(nmc.ServerMessage, "resuming game in 3")
	c.pendingResumeActions = []*time.Timer{
		time.AfterFunc(1*650*time.Millisecond, func() { c.s.Broadcast(nmc.ServerMessage, "... 2") }),
		time.AfterFunc(2*650*time.Millisecond, func() { c.s.Broadcast(nmc.ServerMessage, "... 1") }),
		time.AfterFunc(3*650*time.Millisecond, func() {
			c.casualClock.Resume(p)
			c.pendingResumeActions = nil
		}),
	}
}

func (c *competitiveClock) Leave(p *Player) {
	if p.State != playerstate.Spectator && !c.Ended() {
		c.s.Broadcast(nmc.ServerMessage, "a player left the game")
		c.Pause(nil)
	}
}

func (c *competitiveClock) Stop() {
	if len(c.pendingResumeActions) > 0 {
		for _, action := range c.pendingResumeActions {
			if action != nil {
				action.Stop()
			}
		}
		c.pendingResumeActions = nil
	}
	c.casualClock.Stop()
}

func (c *competitiveClock) ToCasual() *casualClock { return c.casualClock }
