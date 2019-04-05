package game

import (
	"time"
)

type TimedMode interface {
	Timed
	Mode
}

type Timed interface {
	Start()
	ConfirmSpawn(*Player)
	Pause(*Player)
	Paused() bool
	Resume(*Player)
	Leave(*Player)
	End()
	Ended() bool
	CleanUp()
	TimeLeft() time.Duration
	SetTimeLeft(time.Duration)
}
