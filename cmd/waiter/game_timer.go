package main

import (
	"time"

	"github.com/sauerbraten/waiter/internal/pausableticker"
)

const DefaultMapDuration = 1 * time.Minute // for testing and debugging purposes

type GameTimer struct {
	TimeLeft int32 // in milliseconds

	ticker       *pausableticker.Ticker
	intermission func()
}

func NewGameTimer(intermission func()) *GameTimer {
	return &GameTimer{
		TimeLeft:     int32(DefaultMapDuration / time.Millisecond),
		ticker:       pausableticker.NewTicker(100 * time.Millisecond),
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
	*t = *NewGameTimer(t.intermission) // swap out the GameTimer t points to
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
