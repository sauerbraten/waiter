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
	"github.com/sauerbraten/waiter/internal/definitions/role"
	"github.com/sauerbraten/waiter/internal/definitions/weapon"
	"github.com/sauerbraten/waiter/internal/geom"
	"github.com/sauerbraten/waiter/internal/net/enet"
	"github.com/sauerbraten/waiter/pkg/protocol"
	"github.com/sauerbraten/waiter/pkg/protocol/cubecode"
)

type Server struct {
	*Config
	*State
	timer            *GameTimer
	relay            *Relay
	Clients          *ClientManager
	Auth             *auth.Manager
	MapRotation      *MapRotation
	PendingMapChange *time.Timer

	// non-standard stuff
	KeepTeams       bool
	CompetitiveMode bool
}

func (s *Server) AuthRequiredBecause(c *Client) disconnectreason.ID {
	if s.NumClients() >= s.MaxClients {
		return disconnectreason.Full
	}
	if s.MasterMode >= mastermode.Private {
		return disconnectreason.PrivateMode
	}
	if ban, ok := bm.GetBan(c.Peer.Address.IP); ok {
		log.Println("connecting client", c, "is banned:", ban)
		return disconnectreason.IPBanned
	}
	return disconnectreason.None
}

func (s *Server) Connect(peer *enet.Peer) {
	client := s.Clients.Add(peer)
	client.Position, client.Packets = s.relay.AddClient(client.CN, client.Peer.Send)
	client.Send(
		nmc.ServerInfo,
		client.CN,
		protocol.Version,
		client.SessionID,
		false, // password protection is not used by this implementation
		s.ServerDescription,
		s.PrimaryAuthDomain,
	)
}

// checks the server state and decides wether the client has to authenticate to join the game.
func (s *Server) TryJoin(c *Client, name string, playerModel int32, authDomain, authName string) {
	c.Name = name
	c.PlayerModel = playerModel

	onAutoAuthSuccess := func(rol role.ID) {
		s.setAuthRole(c, rol, authDomain, authName)
	}

	onAutoAuthFailure := func() {
		log.Println("unsuccessful auth try at connect by", c, "as", authName, "["+authDomain+"]")
	}

	c.AuthRequiredBecause = s.AuthRequiredBecause(c)

	if c.AuthRequiredBecause == disconnectreason.None {
		s.Join(c)
		if authDomain == s.PrimaryAuthDomain && authName != "" {
			go s.handleAuthRequest(c, authDomain, authName, onAutoAuthSuccess, onAutoAuthFailure)
		}
	} else if authDomain == s.PrimaryAuthDomain && authName != "" {
		// not in a new goroutine, so client does not get confused and sends nmc.ClientPing before the player joined
		s.handleAuthRequest(c, authDomain, authName,
			func(rol role.ID) {
				if rol == role.None {
					return
				}
				c.AuthRequiredBecause = disconnectreason.None
				s.Join(c)
				onAutoAuthSuccess(rol)
			},
			func() {
				onAutoAuthFailure()
				s.Disconnect(c, c.AuthRequiredBecause)
			},
		)
	} else {
		s.Disconnect(c, c.AuthRequiredBecause)
	}
}

// Puts a client into the current game, using the data the client provided with his nmc.TryJoin packet.
func (s *Server) Join(c *Client) {
	c.Joined = true

	if s.MasterMode == mastermode.Locked {
		c.GameState.State = playerstate.Spectator
	} else {
		c.GameState.State = playerstate.Dead
		s.Spawn(c)
	}

	s.Mode().Join(c)         // may set client's team
	s.Clients.SendWelcome(c) // tells client about her team
	s.Mode().Init(c)         // may send additional welcome info like flags
	s.Clients.InformOthersOfJoin(c)

	c.Send(nmc.ServerMessage, s.MessageOfTheDay)

	log.Println(cubecode.SanitizeString(fmt.Sprintf("%s (%s) connected", s.Clients.UniqueName(c), c.Peer.Address.IP)))
}

func (s *Server) Spawn(client *Client) {
	client.GameState.Spawn()
	s.Mode().Spawn(client.GameState)
}

func (s *Server) ConfirmSpawn(client *Client, lifeSequence, _weapon int32) {
	if client.GameState.State != playerstate.Dead || lifeSequence != client.GameState.LifeSequence || client.GameState.LastSpawnAttempt.IsZero() {
		// client may not spawn
		return
	}

	client.GameState.State = playerstate.Alive
	client.GameState.SelectedWeapon = weapon.ByID(weapon.ID(_weapon))
	client.GameState.LastSpawnAttempt = time.Time{}

	client.Packets.Publish(nmc.ConfirmSpawn, client.GameState.ToWire())

	s.Game.ConfirmSpawn(client)
}

func (s *Server) Disconnect(client *Client, reason disconnectreason.ID) {
	s.Game.Leave(client)
	s.relay.RemoveClient(client.CN)
	s.Clients.Disconnect(client, reason)
	if s.Clients.NumberOfClientsConnected() == 0 {
		s.Empty()
	}
}

func (s *Server) Kick(client *Client, victim *Client, reason string) {
	if client.Role <= victim.Role {
		client.Send(nmc.ServerMessage, cubecode.Fail("you can't do that"))
		return
	}
	msg := fmt.Sprintf("%s kicked %s", s.Clients.UniqueName(client), s.Clients.UniqueName(victim))
	if reason != "" {
		msg += " for: " + reason
	}
	s.Clients.Broadcast(nil, nmc.ServerMessage, msg)
	s.Disconnect(victim, disconnectreason.Kick)
}

func (s *Server) AuthKick(client *Client, rol role.ID, domain, name string, victim *Client, reason string) {
	if rol <= victim.Role {
		client.Send(nmc.ServerMessage, cubecode.Fail("you can't do that"))
		return
	}
	msg := fmt.Sprintf("%s as '%s' [%s] kicked %s", s.Clients.UniqueName(client), cubecode.Magenta(name), cubecode.Green(domain), s.Clients.UniqueName(victim))
	if reason != "" {
		msg += " for: " + reason
	}
	s.Clients.Broadcast(nil, nmc.ServerMessage, msg)
	s.Disconnect(victim, disconnectreason.Kick)
}

func (s *Server) Empty() {
	s.MapRotation.queue = s.MapRotation.queue[:0]
	s.KeepTeams = false
	s.CompetitiveMode = false
	s.MasterMode = mastermode.Open
	if s.Mode().ID() != s.FallbackGameMode {
		s.ChangeMap(s.FallbackGameMode, s.MapRotation.NextMap(NewGame(s.FallbackGameMode).Mode(), s.Map))
	}
}

func (s *Server) Intermission() {
	// notify all clients
	s.Clients.Broadcast(nil, nmc.TimeLeft, 0)

	nextMap := s.MapRotation.NextMap(s.Mode(), s.Map)

	// start 5 second timer
	s.PendingMapChange = time.AfterFunc(5*time.Second, func() {
		s.ChangeMap(s.Mode().ID(), nextMap)
	})

	// TODO: send server messages with some top stats

	s.Clients.Broadcast(nil, nmc.ServerMessage, "next up: "+nextMap)
}

func (s *Server) ChangeMap(mode gamemode.ID, mapp string) {
	// cancel pending game mode goroutines
	if s.Mode() != nil {
		s.Mode().CleanUp()
	}

	// stop any pending map change
	if s.PendingMapChange != nil {
		s.PendingMapChange.Stop()
	}

	s.Map = mapp
	s.Game = NewGame(mode)
	s.Clients.ForEach(s.Mode().Join)

	s.Clients.Broadcast(nil, nmc.MapChange, s.Map, s.Mode().ID(), s.Mode().NeedMapInfo())
	s.timer.Restart()
	s.Clients.Broadcast(nil, nmc.TimeLeft, s.timer.TimeLeft/1000)
	s.Clients.MapChange()

	s.Game.Start()

	s.Clients.Broadcast(nil, nmc.ServerMessage, s.MessageOfTheDay)
}

func (s *Server) PauseGame(c *Client) {
	log.Println("pausing game at", s.timer.TimeLeft/1000, "seconds left")
	cn := -1
	if c != nil {
		cn = int(c.CN)
	}
	s.Clients.Broadcast(nil, nmc.PauseGame, 1, cn)
	s.timer.Pause()
	s.Mode().Pause()
}

func (s *Server) ResumeGame(c *Client) {
	cn := -1
	if c != nil {
		cn = int(c.CN)
	}
	s.timer.ResumeWithCountdown(cn)
	s.Mode().Resume()
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
	fragger.GameState.Frags += s.Mode().FragValue(fragger, victim)
	// TODO: effectiveness
	s.Mode().HandleDeath(fragger, victim)
	s.Clients.Broadcast(nil, nmc.Died, victim.CN, fragger.CN, fragger.GameState.Frags, fragger.Team.Frags)
	// TODO teamkills
}
