package server

import (
	"time"

	"github.com/sauerbraten/waiter/pkg/game"
	"github.com/sauerbraten/waiter/pkg/protocol/mastermode"
)

type State struct {
	Clock      Clock
	MasterMode mastermode.ID
	GameMode   game.Mode
	Map        string
	UpSince    time.Time
	NumClients func() int // number of clients connected
}

type Clock interface {
	Start()
	Pause(*game.Player)
	Paused() bool
	Resume(*game.Player)
	Stop()
	Ended() bool
	TimeLeft() time.Duration
	SetTimeLeft(time.Duration)
	Join(*game.Player)
	Spawned(*game.Player)
	Leave(*game.Player)
}
