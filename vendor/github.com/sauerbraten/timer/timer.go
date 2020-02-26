package timer

import (
	"sync"
	"time"
)

const (
	stateIdle = iota
	stateActive
	stateExpired
)

// The Timer type represents a single event. When the Timer expires,
// the current time will be sent on C, unless the Timer was created by AfterFunc.
// A Timer must be created with NewTimer or AfterFunc.
type Timer struct {
	t  *time.Timer
	C  <-chan time.Time
	c  chan time.Time
	fn func()

	l         *sync.Mutex // to synchronize access to the fields below
	state     int
	duration  time.Duration
	startedAt time.Time
}

// AfterFunc waits after calling its Start method for the duration
// to elapse and then calls f in its own goroutine.
// It returns a Timer that can be used to cancel the call using its Stop method,
// or pause using its Pause method
func AfterFunc(d time.Duration, f func()) *Timer {
	t := new(Timer)
	t.duration = d
	t.fn = func() {
		t.state = stateExpired
		f()
	}
	return t
}

// NewTimer creates a new Timer.
// It returns a Timer that can be used to cancel the call using its Stop method,
// or pause using its Pause method
func NewTimer(d time.Duration) *Timer {
	c := make(chan time.Time, 1)
	t := new(Timer)
	t.C = c
	t.c = c
	t.duration = d
	t.fn = func() {
		t.state = stateExpired
		t.c <- time.Now()
	}
	return t
}

// Pause pauses current timer until Start method will be called.
// Next Start call will wait rest of duration.
func (t *Timer) Pause() bool {
	t.l.Lock()
	defer t.l.Unlock()
	if t.state != stateActive {
		return false
	}
	if !t.t.Stop() {
		t.state = stateExpired
		return false
	}
	t.state = stateIdle
	t.duration -= time.Now().Sub(t.startedAt)
	return true
}

// Start starts Timer that will send the current time on its channel after at least duration d.
func (t *Timer) Start() bool {
	t.l.Lock()
	defer t.l.Unlock()
	if t.state != stateIdle {
		return false
	}
	t.startedAt = time.Now()
	t.state = stateActive
	t.t = time.AfterFunc(t.duration, t.fn)
	return true
}

// Stop prevents the Timer from firing. It returns true if the call stops the timer,
// false if the timer has already expired or been stopped.
// Stop does not close the channel, to prevent a read from the channel succeeding incorrectly.
func (t *Timer) Stop() bool {
	t.l.Lock()
	defer t.l.Unlock()
	if t.state != stateActive {
		return false
	}
	t.startedAt = time.Now()
	t.state = stateExpired
	t.t.Stop()
	return true
}

// TimeLeft returns the duration left to run before the timer expires.
func (t *Timer) TimeLeft() time.Duration {
	t.l.Lock()
	defer t.l.Unlock()
	switch t.state {
	case stateIdle:
		return t.duration
	case stateActive:
		return time.Now().Sub(t.startedAt)
	case stateExpired:
		return 0
	default:
		panic("unhandled timer state")
	}
}
