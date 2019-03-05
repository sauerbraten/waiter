package main

import (
	"time"

	"github.com/sauerbraten/waiter/internal/game"

	"github.com/sauerbraten/waiter/pkg/definitions/mastermode"
)

type State struct {
	MasterMode mastermode.ID
	GameMode   game.Mode
	Map        string
	UpSince    time.Time
	NumClients func() int // number of clients connected
}
