package game

import (
	"time"

	"github.com/sauerbraten/waiter/pkg/protocol/nmc"
)

type TimedMode interface {
	Mode
	Pause(*Player)
	Paused() bool
	Resume(*Player)
	Ended() bool
	TimeLeft() time.Duration
	SetTimeLeft(time.Duration)
}

type timedMode struct {
	t *Timer
	s Server
}

func newTimedMode(s Server) timedMode {
	return timedMode{
		s: s,
	}
}

func (tm *timedMode) Start() {
	tm.t = StartTimer(tm.s.GameDuration(), tm.s.Intermission)
	tm.s.Broadcast(nmc.TimeLeft, tm.s.GameDuration())
}

func (tm *timedMode) Pause(p *Player) {
	cn := -1
	if p != nil {
		cn = int(p.CN)
	}
	tm.s.Broadcast(nmc.PauseGame, 1, cn)
	tm.t.Pause()
}

func (tm *timedMode) Paused() bool {
	return tm.t.Paused()
}

func (tm *timedMode) Resume(p *Player) {
	cn := -1
	if p != nil {
		cn = int(p.CN)
	}
	tm.s.Broadcast(nmc.PauseGame, 0, cn)
	tm.t.Resume()
}

func (tm *timedMode) End() {
	tm.s.Broadcast(nmc.TimeLeft, 0)
	tm.t.Stop()
}

func (tm *timedMode) Ended() bool {
	return tm.t.Stopped()
}

func (tm *timedMode) CleanUp() {
	if tm.Paused() {
		tm.Resume(nil)
	}
	tm.t.Stop()
}

func (tm *timedMode) TimeLeft() time.Duration {
	return tm.t.TimeLeft
}

func (tm *timedMode) SetTimeLeft(d time.Duration) {
	tm.t.TimeLeft = d
	tm.s.Broadcast(nmc.TimeLeft, d)
}
