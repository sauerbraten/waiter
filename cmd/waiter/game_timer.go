package main

import (
	"time"

	"github.com/sauerbraten/waiter/pkg/pausableticker"
)

type GameTimer struct {
	*pausableticker.Ticker
	TimeLeft     int32 // in milliseconds
	duration     time.Duration
	intermission func()
}

func StartTimer(duration time.Duration, intermission func()) *GameTimer {
	gt := &GameTimer{
		Ticker:       pausableticker.New(100 * time.Millisecond),
		TimeLeft:     int32(duration / time.Millisecond),
		duration:     duration,
		intermission: intermission,
	}
	go gt.run()
	return gt
}

func (gt *GameTimer) Restart() {
	gt.Stop()
	gt.Ticker = pausableticker.New(100 * time.Millisecond)
	gt.TimeLeft = int32(gt.duration / time.Millisecond)
	go gt.run()
}

func (gt *GameTimer) run() {
	for range gt.C {
		s.timer.TimeLeft -= 100
		if s.timer.TimeLeft <= 0 {
			gt.intermission()
			return
		}
	}
}
