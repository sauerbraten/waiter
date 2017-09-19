package server

import (
	"time"

	"github.com/sauerbraten/waiter/internal/protocol/definitions/gamemode"
	"github.com/sauerbraten/waiter/internal/protocol/definitions/mastermode"
)

type State struct {
	MasterMode  mastermode.MasterMode
	GameMode    gamemode.GameMode
	Map         string
	TimeLeft    int32 // in milliseconds
	Paused      bool
	NotGotItems bool
	HasMaster   bool // true if one or more clients have master privilege or higher
	UpSince     time.Time
	NumClients  func() int // number of clients connected
}
