package main

import (
	"log"
	"time"

	"github.com/sauerbraten/waiter/internal/definitions/nmc"
	"github.com/sauerbraten/waiter/pkg/pausableticker"
)

type GameTimer struct {
	*pausableticker.Ticker
	TimeLeft             int32 // in milliseconds
	duration             time.Duration
	intermission         func()
	pendingResumeActions []*time.Timer
}

func StartTimer(duration time.Duration, intermission func()) *GameTimer {
	gt := &GameTimer{
		Ticker:               pausableticker.New(100 * time.Millisecond),
		TimeLeft:             int32(duration / time.Millisecond),
		duration:             duration,
		intermission:         intermission,
		pendingResumeActions: []*time.Timer{},
	}
	go gt.run()
	return gt
}

func (gt *GameTimer) ResumeWithCountdown(cn int) {
	if len(gt.pendingResumeActions) > 0 {
		for _, action := range gt.pendingResumeActions {
			if action != nil {
				action.Stop()
			}
		}
		gt.pendingResumeActions = nil
		s.Clients.Broadcast(nil, nmc.ServerMessage, "resuming aborted")
		return
	}

	s.Clients.Broadcast(nil, nmc.ServerMessage, "resuming in 3 seconds")
	s.timer.pendingResumeActions = []*time.Timer{
		time.AfterFunc(1*time.Second, func() {
			s.Clients.Broadcast(nil, nmc.ServerMessage, "resuming in 2 seconds")
		}),
		time.AfterFunc(2*time.Second, func() {
			s.Clients.Broadcast(nil, nmc.ServerMessage, "resuming in 1 second")
		}),
		time.AfterFunc(3*time.Second, func() {
			log.Println("resuming game at", s.timer.TimeLeft/1000, "seconds left")
			gt.Ticker.Resume()
			gt.pendingResumeActions = nil
			s.Clients.Broadcast(nil, nmc.PauseGame, 0, cn)
		}),
	}
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
			s.timer.TimeLeft = 0
			gt.intermission()
			return
		}
	}
}
