package main

import (
	"time"

	"github.com/sauerbraten/waiter/internal/auth"
	"github.com/sauerbraten/waiter/internal/client/playerstate"
	"github.com/sauerbraten/waiter/internal/definitions/gamemode"
	"github.com/sauerbraten/waiter/internal/definitions/nmc"
	"github.com/sauerbraten/waiter/internal/definitions/weapon"
	"github.com/sauerbraten/waiter/internal/geom"
	"github.com/sauerbraten/waiter/internal/maprotation"
	"github.com/sauerbraten/waiter/internal/net/enet"
	"github.com/sauerbraten/waiter/internal/net/packet"
)

type Server struct {
	*Config
	*State
	*GameTimer
	relay   *Relay
	Clients *ClientManager
	Auth    *auth.Manager
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

	// load next map
	s.ChangeMap(s.GameMode.ID(), maprotation.NextMap(s.GameMode.ID(), s.Map))
}

func (s *Server) ChangeMap(mode gamemode.ID, mapp string) {
	s.NotGotItems = true
	s.GameMode = GameModeByID(mode)
	s.Map = mapp
	s.Clients.Broadcast(nil, 1, enet.PACKET_FLAG_RELIABLE, nmc.MapChange, s.Map, s.GameMode.ID(), s.NotGotItems)
	s.Clients.Broadcast(nil, 1, enet.PACKET_FLAG_RELIABLE, nmc.TimeLeft, s.TimeLeft/1000)
	s.Clients.MapChange()
	s.Clients.Broadcast(nil, 1, enet.PACKET_FLAG_RELIABLE, nmc.ServerMessage, s.MessageOfTheDay)
}

type hit struct {
	target       uint32
	lifeSequence int32
	distance     float64
	rays         int32
	dir          *geom.Vector
}

func (s *Server) HandleShoot(client *Client, wpn weapon.Weapon, id int32, from, to *geom.Vector, hits []hit) {
	from.Mul(geom.DMF)
	to.Mul(geom.DMF)

	s.Clients.Broadcast(exclude(client), 1, enet.PACKET_FLAG_RELIABLE,
		nmc.ShotEffects,
		client.CN,
		wpn.ID,
		id,
		from.X(),
		from.Y(),
		from.Z(),
		to.X(),
		to.Y(),
		to.Z(),
	)
	client.GameState.LastShot = time.Now()
	client.GameState.ShotDamage += wpn.Damage * wpn.Rays // TODO: quad damage
	switch wpn.ID {
	case weapon.GrenadeLauncher, weapon.RocketLauncher:
		// TODO: save somewhere
	default:
		// apply damage
		rays := int32(0)
		for _, h := range hits {
			target := s.Clients.GetClientByCN(h.target)
			if target == nil ||
				target.GameState.State != playerstate.Alive ||
				target.GameState.LifeSequence != h.lifeSequence ||
				h.rays < 1 ||
				h.distance > wpn.Range+1.0 {
				continue
			}

			rays += h.rays
			if rays > wpn.Rays {
				continue
			}

			damage := h.rays * wpn.Damage
			// TODO: quad damage

			s.applyDamage(client, target, int32(damage), wpn.ID, h.dir)
		}
	}
}

func (s *Server) HandleExplode(client *Client, millis int32, wpn weapon.Weapon, id int32, hits []hit) {
	// TODO: delete stored projectile

	s.Clients.Broadcast(exclude(client), 1, enet.PACKET_FLAG_RELIABLE,
		nmc.ExplodeEffects,
		client.CN,
		wpn.ID,
		id,
	)

	// apply damage
hits:
	for i, h := range hits {
		target := s.Clients.GetClientByCN(h.target)
		if target == nil ||
			target.GameState.State != playerstate.Alive ||
			target.GameState.LifeSequence != h.lifeSequence ||
			h.distance < 0 ||
			h.distance > wpn.ExplosionRadius {
			continue
		}

		// avoid duplicates
		for j := range hits[:i] {
			if hits[j].target == h.target {
				continue hits
			}
		}

		damage := float64(wpn.Damage)
		// TODO: quad damage
		damage *= (1 - h.distance/weapon.ExplosionDistanceScale/wpn.ExplosionRadius)
		if target == client {
			damage *= weapon.ExplosionSelfDamageScale
		}

		s.applyDamage(client, target, int32(damage), wpn.ID, h.dir)
	}
}

func (s *Server) applyDamage(attacker, victim *Client, damage int32, wpnID weapon.ID, dir *geom.Vector) {
	victim.applyDamage(attacker, damage, wpnID, dir)
	s.Clients.Broadcast(nil, 1, enet.PACKET_FLAG_RELIABLE, nmc.Damage, victim.CN, attacker.CN, damage, victim.GameState.Armour, victim.GameState.Health)
	// TODO: setpushed ???
	if !dir.IsZero() {
		dir.Scale(geom.DNF)
		p := []interface{}{nmc.HitPush, victim.CN, wpnID, damage, dir.X(), dir.Y(), dir.Z()}
		if victim.GameState.Health <= 0 {
			s.Clients.Broadcast(nil, 1, enet.PACKET_FLAG_RELIABLE, p...)
		} else {
			attacker.Peer.Send(1, enet.PACKET_FLAG_RELIABLE, packet.Encode(p...))
		}
	}
	if victim.GameState.Health <= 0 {
		s.handleDeath(attacker, victim)
	}
}

func (s *Server) handleDeath(fragger, victim *Client) {
	victim.Die()
	fragger.GameState.Frags += s.GameMode.CountFrag(fragger, victim)
	// TODO: effectiveness
	s.Clients.Broadcast(nil, 1, enet.PACKET_FLAG_RELIABLE, nmc.Died, victim.CN, fragger.CN, fragger.GameState.Frags, s.GameMode.TeamFrags(fragger.Team))
	// TODO teamkills
}

func (s *Server) handleSuicide(client *Client) {
	s.handleDeath(client, client)
	client.GameState.Respawn()
}
