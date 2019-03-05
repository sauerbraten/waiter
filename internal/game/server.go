package game

import (
	"time"

	"github.com/sauerbraten/waiter/pkg/definitions/nmc"
)

type Server interface {
	GameDuration() time.Duration
	Broadcast(nmc.ID, ...interface{})
	Send(*Player, nmc.ID, ...interface{})
	Intermission()
	ForEach(func(*Player))
	UniqueName(*Player) string
	HandleDeath(*Player)
}
