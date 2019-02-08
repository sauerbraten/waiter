package main

import (
	"time"

	"github.com/sauerbraten/waiter/pkg/pausableticker"
)

type GameTimer struct {
	TimeLeft int32 // in milliseconds

	ticker       *pausableticker.Ticker
	duration     time.Duration
	intermission func()
}

func NewGameTimer(duration time.Duration, intermission func()) *GameTimer {
	return &GameTimer{
		TimeLeft:     int32(duration / time.Millisecond),
		ticker:       pausableticker.NewTicker(100 * time.Millisecond),
		duration:     duration,
		intermission: intermission,
	}
}

func (t *GameTimer) Pause() {
	t.ticker.Pause()
}

func (t *GameTimer) Resume() {
	t.ticker.Resume()
}

func (t *GameTimer) IsPaused() bool {
	return t.ticker.Paused
}

func (t *GameTimer) Stop() {
	t.ticker.Stop()
}

func (t *GameTimer) Reset() {
	t.Stop()
	*t = *NewGameTimer(t.duration, t.intermission) // swap out the GameTimer t points to
}

func (t *GameTimer) run() {
	defer t.intermission()
	for {
		<-t.ticker.C
		s.TimeLeft -= 100

		if s.TimeLeft <= 0 {
			return
		}
	}
}
