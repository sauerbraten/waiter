package main

import (
	"time"

	"github.com/sauerbraten/waiter/internal/definitions/mastermode"
)

type State struct {
	MasterMode mastermode.ID
	GameMode   GameMode
	Map        string
	UpSince    time.Time
	NumClients func() int // number of clients connected
}
