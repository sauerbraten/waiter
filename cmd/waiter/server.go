package main

import (
	"time"

	"github.com/sauerbraten/waiter/internal/client"
	"github.com/sauerbraten/waiter/internal/client/playerstate"
	"github.com/sauerbraten/waiter/internal/enet"
	"github.com/sauerbraten/waiter/internal/protocol/definitions/nmc"
	"github.com/sauerbraten/waiter/internal/protocol/packet"
	"github.com/sauerbraten/waiter/internal/server/maprotation"
)

var (
	// channel to pause the game
	pauseChannel = make(chan bool)

	// channel to interrupt the game (for example when a master changes mode or map mid-game)
	interruptChannel = make(chan bool)
)

const (
	MAP_TIME int32 = 180000 // 3 minutes for testing and debugging purposes
)

func countDown() {
	endTimer := time.NewTimer(time.Duration(s.State.TimeLeft) * time.Millisecond)
	gameTicker := time.NewTicker(1 * time.Millisecond)
	paused := false

	for {
		select {
		case <-gameTicker.C:
			s.State.TimeLeft--

		case shouldPause := <-pauseChannel:
			if shouldPause && !paused {
				endTimer.Stop()
				gameTicker.Stop()
				paused = true
			} else if !shouldPause && paused {
				endTimer.Reset(time.Duration(s.State.TimeLeft) * time.Millisecond)
				gameTicker = time.NewTicker(1 * time.Millisecond)
				paused = false
			}

		case <-interruptChannel:
			endTimer.Stop()
			gameTicker.Stop()

		case <-endTimer.C:
			endTimer.Stop()
			gameTicker.Stop()
			go intermission()
			return
		}
	}
}

func intermission() {
	// notify all clients
	cm.Broadcast(enet.PACKET_FLAG_RELIABLE, 1, packet.New(nmc.TimeLeft, 0))

	// start 5 second timer
	end := time.After(5 * time.Second)

	// TODO: send server messages with some top stats

	// wait for timer to finish
	<-end

	// start new 10 minutes timer
	s.State.TimeLeft = MAP_TIME
	go countDown()

	// change map
	changeMap(maprotation.NextMap(s.State.Map))
}

func changeMap(mapName string) {
	s.State.NotGotItems = true
	s.State.Map = mapName
	cm.Broadcast(enet.PACKET_FLAG_RELIABLE, 1, packet.New(nmc.MapChange, s.State.Map, s.State.GameMode, s.State.NotGotItems))
	cm.Broadcast(enet.PACKET_FLAG_RELIABLE, 1, packet.New(nmc.TimeLeft, s.State.TimeLeft/1000))
	cm.ForEach(func(c *client.Client) {
		if c.InUse && c.GameState.State != playerstate.Spectator {

			c.GameState.Reset()
			c.SendSpawnState(s.State)
		}
	})

	sendMOTD()
}

func sendMOTD() {
	// send development notice
	p := packet.New(nmc.ServerMessage, "This server is written in Go and still under development. \f6Most things do not work!")
	p.Put(nmc.ServerMessage, "Visit github.com/sauerbraten/waiter for more information and source code.")
	cm.Broadcast(enet.PACKET_FLAG_RELIABLE, 1, p)
}
