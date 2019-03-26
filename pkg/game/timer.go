package game

import (
	"time"

	"github.com/sauerbraten/waiter/pkg/pausableticker"
)

type Timer struct {
	*pausableticker.Ticker
	TimeLeft             time.Duration
	duration             time.Duration
	intermission         func()
	pendingResumeActions []*time.Timer
}

func StartTimer(duration time.Duration, intermission func()) *Timer {
	t := &Timer{
		Ticker:               pausableticker.New(100 * time.Millisecond),
		TimeLeft:             duration,
		duration:             duration,
		intermission:         intermission,
		pendingResumeActions: []*time.Timer{},
	}
	go t.run()
	return t
}

func (t *Timer) run() {
	for range t.C {
		t.TimeLeft -= 100 * time.Millisecond
		if t.TimeLeft <= 0 {
			t.TimeLeft = 0
			t.intermission()
			return
		}
	}
}
