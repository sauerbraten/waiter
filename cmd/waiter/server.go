package main

import (
	"fmt"
	"log"
	"time"

	"github.com/sauerbraten/waiter/internal/auth"
	"github.com/sauerbraten/waiter/internal/definitions/disconnectreason"
	"github.com/sauerbraten/waiter/internal/definitions/gamemode"
	"github.com/sauerbraten/waiter/internal/definitions/mastermode"
	"github.com/sauerbraten/waiter/internal/definitions/nmc"
	"github.com/sauerbraten/waiter/internal/definitions/playerstate"
	"github.com/sauerbraten/waiter/internal/definitions/weapon"
	"github.com/sauerbraten/waiter/internal/geom"
	"github.com/sauerbraten/waiter/internal/maprotation"
	"github.com/sauerbraten/waiter/internal/net/enet"
	"github.com/sauerbraten/waiter/pkg/protocol"
	"github.com/sauerbraten/waiter/pkg/protocol/cubecode"
)

type Server struct {
	*Config
	*State
	timer   *GameTimer
	relay   *Relay
	Clients *ClientManager
	Auth    *auth.Manager

	PendingMapChange *time.Timer
	KeepTeams        bool
}

func (s *Server) Connect(peer *enet.Peer) {
	client := s.Clients.Add(peer)
	client.Position, client.Packets = s.relay.AddClient(client.CN, client.Peer.Send)
	client.Send(
		nmc.ServerInfo,
		client.CN,
		protocol.Version,
		client.SessionID,
		false,
		s.ServerDescription,
		s.PrimaryAuthDomain,
	)
}

// Puts a client into the current game, using the data the client provided with his N_JOIN packet.
func (s *Server) Join(c *Client, name string, playerModel int32, authDomain, authName string) {
	c.Joined = true
	c.Name = name
	c.PlayerModel = playerModel

	if s.MasterMode == mastermode.Locked {
		c.GameState.State = playerstate.Spectator
	} else {
		c.GameState.State = playerstate.Dead
		s.Spawn(c)
	}

	s.GameMode.Join(c)       // may set client's team
	s.Clients.SendWelcome(c) // tells client about her team
	s.GameMode.Init(c)       // may send additional welcome info like flags
	s.Clients.InformOthersOfJoin(c)

	c.Send(nmc.ServerMessage, s.MessageOfTheDay)

	if authDomain != "" && authName != "" {
		s.handleAuthRequest(c, authDomain, authName)
	}

	log.Println(cubecode.SanitizeString(fmt.Sprintf("%s (%s) connected", s.Clients.UniqueName(c), c.Peer.Address.IP)))
}

func (s *Server) Spawn(client *Client) {
	client.GameState.Spawn()
	s.GameMode.Spawn(client.GameState)
}

func (s *Server) Disconnect(client *Client, reason disconnectreason.ID) {
	s.GameMode.Leave(client)
	s.relay.RemoveClient(client.CN)
	s.Clients.Disconnect(client, reason)
	if s.Clients.NumberOfClientsConnected() == 0 {
		s.Empty()
	}
}

func (s *Server) Empty() {
	s.KeepTeams = false
	s.MasterMode = mastermode.Open
	s.ChangeMap(s.FallbackGameMode, maprotation.NextMap(s.FallbackGameMode, s.Map))
}

func (s *Server) Intermission() {
	// notify all clients
	s.Clients.Broadcast(nil, nmc.TimeLeft, 0)

	// start 5 second timer
	s.PendingMapChange = time.AfterFunc(5*time.Second, func() {
		s.ChangeMap(s.GameMode.ID(), maprotation.NextMap(s.GameMode.ID(), s.Map))
	})

	// TODO: send server messages with some top stats
}

func (s *Server) ChangeMap(mode gamemode.ID, mapp string) {
	// stop any pending map change
	if s.PendingMapChange != nil {
		s.PendingMapChange.Stop()
	}

	s.Map = mapp
	s.GameMode = StartGame(mode)
	s.Clients.ForEach(s.GameMode.Join)

	s.Clients.Broadcast(nil, nmc.MapChange, s.Map, s.GameMode.ID(), s.GameMode.NeedMapInfo())
	s.timer.Restart()
	s.Clients.Broadcast(nil, nmc.TimeLeft, s.timer.TimeLeft/1000)
	s.Clients.MapChange()
	s.Clients.Broadcast(nil, nmc.ServerMessage, s.MessageOfTheDay)
}

func (s *Server) SetKeepTeams(keepTeams bool) {
	s.KeepTeams = keepTeams
	if keepTeams {
		s.Clients.Broadcast(nil, nmc.ServerMessage, "keeping teams")
	} else {
		s.Clients.Broadcast(nil, nmc.ServerMessage, "teams will be shuffled on map change")
	}
}

type hit struct {
	target       uint32
	lifeSequence int32
	distance     float64
	rays         int32
	dir          *geom.Vector
}

func (s *Server) HandleShoot(client *Client, wpn weapon.Weapon, id int32, from, to *geom.Vector, hits []hit) {
	from = from.Mul(geom.DMF)
	to = to.Mul(geom.DMF)

	s.Clients.Relay(
		client,
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

	s.Clients.Relay(
		client,
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
	s.Clients.Broadcast(nil, nmc.Damage, victim.CN, attacker.CN, damage, victim.GameState.Armour, victim.GameState.Health)
	// TODO: setpushed ???
	if !dir.IsZero() {
		dir = dir.Scale(geom.DNF)
		p := []interface{}{nmc.HitPush, victim.CN, wpnID, damage, dir.X(), dir.Y(), dir.Z()}
		if victim.GameState.Health <= 0 {
			s.Clients.Broadcast(nil, p...)
		} else {
			victim.Send(p...)
		}
	}
	if victim.GameState.Health <= 0 {
		s.handleDeath(attacker, victim)
	}
}

func (s *Server) handleDeath(fragger, victim *Client) {
	victim.Die()
	fragger.GameState.Frags += s.GameMode.FragValue(fragger, victim)
	// TODO: effectiveness
	s.GameMode.HandleDeath(fragger, victim)
	s.Clients.Broadcast(nil, nmc.Died, victim.CN, fragger.CN, fragger.GameState.Frags, fragger.Team.Frags)
	// TODO teamkills
}
