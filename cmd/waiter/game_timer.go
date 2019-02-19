package main

import (
	"time"

	"github.com/sauerbraten/waiter/pkg/pausableticker"
)

type GameTimer struct {
	*pausableticker.Ticker
	TimeLeft             time.Duration
	duration             time.Duration
	intermission         func()
	pendingResumeActions []*time.Timer
}

func StartTimer(duration time.Duration, intermission func()) *GameTimer {
	gt := &GameTimer{
		Ticker:               pausableticker.New(100 * time.Millisecond),
		TimeLeft:             duration,
		duration:             duration,
		intermission:         intermission,
		pendingResumeActions: []*time.Timer{},
	}
	go gt.run()
	return gt
}

func (gt *GameTimer) run() {
	for range gt.C {
		gt.TimeLeft -= 100 * time.Millisecond
		if gt.TimeLeft <= 0 {
			gt.TimeLeft = 0
			gt.intermission()
			return
		}
	}
}
