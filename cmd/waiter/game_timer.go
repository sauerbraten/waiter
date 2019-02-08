package main

import (
	"time"

	"github.com/sauerbraten/waiter/pkg/pausableticker"
)

type GameTimer struct {
	*pausableticker.Ticker

	TimeLeft int32 // in milliseconds

	duration     time.Duration
	intermission func()
}

func NewGameTimer(duration time.Duration, intermission func()) *GameTimer {
	return &GameTimer{
		Ticker: pausableticker.NewTicker(100 * time.Millisecond),

		TimeLeft:     int32(duration / time.Millisecond),
		duration:     duration,
		intermission: intermission,
	}
}

func (t *GameTimer) Reset() {
	t.Stop()
	*t = *NewGameTimer(t.duration, t.intermission) // swap out the GameTimer t points to
}

func (t *GameTimer) run() {
	defer t.intermission()
	for {
		<-t.C
		s.TimeLeft -= 100

		if s.TimeLeft <= 0 {
			return
		}
	}
}
