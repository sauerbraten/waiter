package main

import (
	"time"

	"github.com/sauerbraten/waiter/internal/definitions/gamemode"
	"github.com/sauerbraten/waiter/internal/definitions/mastermode"
)

type State struct {
	MasterMode mastermode.MasterMode
	GameMode   gamemode.GameMode
	Map        string

	NotGotItems bool
	UpSince     time.Time
	NumClients  func() int // number of clients connected
}
