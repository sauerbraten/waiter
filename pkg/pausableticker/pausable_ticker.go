package pausableticker

import (
	"time"
)

type Ticker struct {
	C <-chan time.Time // The channel on which the ticks are delivered.

	pause  chan bool
	paused bool
	stop   chan struct{}
	ticker *time.Ticker
}

func New(d time.Duration) *Ticker {
	c := make(chan time.Time)
	pause := make(chan bool)
	stop := make(chan struct{})
	ticker := time.NewTicker(d)

	t := &Ticker{
		C:      c,
		pause:  pause,
		stop:   stop,
		ticker: ticker,
	}

	go t.run(c)

	return t
}

func (t *Ticker) run(c chan<- time.Time) {
	for {
		select {
		case c <- <-t.ticker.C:
		case shouldPause := <-t.pause:
			if shouldPause {
				t.paused = true
				for shouldPause {
					shouldPause = <-t.pause
				}
				t.paused = false
			}
		case <-t.stop:
			close(t.pause)
			close(t.stop)
			return
		}
	}
}

func (t *Ticker) Pause() {
	t.pause <- true
}

func (t *Ticker) Paused() bool {
	return t.paused
}

func (t *Ticker) Resume() {
	t.pause <- false
}

func (t *Ticker) Stop() {
	t.stop <- struct{}{}
	<-t.stop
	t.ticker.Stop()
}
