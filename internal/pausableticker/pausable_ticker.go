package pausableticker

import "time"

type Ticker struct {
	C      <-chan time.Time // The channel on which the ticks are delivered.
	Paused bool

	pause  chan bool
	stop   chan struct{}
	ticker *time.Ticker
}

func NewTicker(d time.Duration) *Ticker {
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
				t.Paused = true
				for shouldPause {
					shouldPause = <-t.pause
				}
				t.Paused = false
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

func (t *Ticker) Resume() {
	t.pause <- false
}

func (t *Ticker) Stop() {
	t.ticker.Stop()
	t.stop <- struct{}{}
	<-t.stop
}
