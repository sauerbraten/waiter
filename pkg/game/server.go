package game

import (
	"time"

	"github.com/sauerbraten/waiter/pkg/protocol/nmc"
)

type Server interface {
	GameDuration() time.Duration
	Broadcast(nmc.ID, ...any)
	Intermission()
	ForEachPlayer(func(*Player))
	UniqueName(*Player) string
	NumberOfPlayers() int
}
