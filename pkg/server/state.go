package server

import (
	"time"

	"github.com/sauerbraten/waiter/pkg/game"
	"github.com/sauerbraten/waiter/pkg/protocol/mastermode"
)

type State struct {
	MasterMode mastermode.ID
	GameMode   game.TimedMode // server only supports timed modes atm
	Map        string
	UpSince    time.Time
	NumClients func() int // number of clients connected
}
