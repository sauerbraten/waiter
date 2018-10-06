package main

import (
	"time"

	"github.com/sauerbraten/waiter/internal/auth"
	"github.com/sauerbraten/waiter/internal/definitions/disconnectreason"
	"github.com/sauerbraten/waiter/internal/definitions/nmc"
	"github.com/sauerbraten/waiter/internal/maprotation"
	"github.com/sauerbraten/waiter/internal/protocol/enet"
)

type Server struct {
	*Config
	*State
	*GameTimer
	relay   *Relay
	Clients *ClientManager
	Auth    *auth.Manager
}

func (s *Server) IsAllowedToJoin(c *Client, hash string, authDomain string, authName string) bool {
	// TODO: check server password hash

	// check for mandatory connect auth
	if c.AuthRequiredBecause > disconnectreason.None {
		if authDomain != s.AuthDomains[0] {
			return false
		}

		// TODO: try to authenticate the client
	}

	return true
}

func (s *Server) Intermission() {
	// notify all clients
	s.Clients.Broadcast(nil, 1, enet.PACKET_FLAG_RELIABLE, nmc.TimeLeft, 0)

	// start 5 second timer
	end := time.After(5 * time.Second)

	// TODO: send server messages with some top stats

	// wait for timer to finish
	<-end

	// start new 10 minutes timer
	s.GameTimer.Reset()
	go s.GameTimer.run()

	// change map
	s.ChangeMap(maprotation.NextMap(s.GameMode, s.Map))
}

func (s *Server) ChangeMap(mapName string) {
	s.NotGotItems = true
	s.Map = mapName
	s.Clients.Broadcast(nil, 1, enet.PACKET_FLAG_RELIABLE, nmc.MapChange, s.Map, s.GameMode, s.NotGotItems)
	s.Clients.Broadcast(nil, 1, enet.PACKET_FLAG_RELIABLE, nmc.TimeLeft, s.TimeLeft/1000)
	s.Clients.MapChange()
	s.Clients.Broadcast(nil, 1, enet.PACKET_FLAG_RELIABLE, nmc.ServerMessage, s.MessageOfTheDay)
}
