package game

import (
	"time"

	"github.com/sauerbraten/waiter/pkg/protocol/nmc"
)

type Server interface {
	GameDuration() time.Duration
	Broadcast(nmc.ID, ...interface{})
	Intermission()
	ForEach(func(*Player))
	UniqueName(*Player) string
}
