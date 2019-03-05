package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/sauerbraten/waiter/internal/relay"

	"github.com/sauerbraten/waiter/internal/game"

	"github.com/sauerbraten/waiter/internal/geom"
	"github.com/sauerbraten/waiter/internal/net/enet"
	"github.com/sauerbraten/waiter/internal/utils"
	"github.com/sauerbraten/waiter/pkg/auth"
	"github.com/sauerbraten/waiter/pkg/definitions/disconnectreason"
	"github.com/sauerbraten/waiter/pkg/definitions/gamemode"
	"github.com/sauerbraten/waiter/pkg/definitions/mastermode"
	"github.com/sauerbraten/waiter/pkg/definitions/nmc"
	"github.com/sauerbraten/waiter/pkg/definitions/playerstate"
	"github.com/sauerbraten/waiter/pkg/definitions/role"
	"github.com/sauerbraten/waiter/pkg/definitions/weapon"
	"github.com/sauerbraten/waiter/pkg/protocol"
	"github.com/sauerbraten/waiter/pkg/protocol/cubecode"
)

type Server struct {
	ENetHost *enet.Host
	*Config
	*State
	relay            *relay.Relay
	Clients          *ClientManager
	AuthManager      *auth.Manager
	MapRotation      *MapRotation
	PendingMapChange *time.Timer

	// non-standard stuff
	KeepTeams       bool
	CompetitiveMode bool
	ReportStats     bool
}

func (s *Server) GameDuration() time.Duration { return s.GameDurationInMinutes }

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
	client.Positions, client.Packets = s.relay.AddClient(client.CN, client.Peer.Send)
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
	c.Model = playerModel

	onAutoAuthSuccess := func(rol role.ID) {
		s.setAuthRole(c, rol, authDomain, authName)
	}

	onAutoAuthFailure := func(err error) {
		if err != nil {
			log.Println(err)
		}
		log.Printf("unsuccessful auth try at connect by %s as '%s' [%s]: %v", c, authName, authDomain)
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
			func(err error) {
				onAutoAuthFailure(err)
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
		c.State = playerstate.Spectator
	} else {
		c.State = playerstate.Dead
		s.Spawn(c)
	}

	s.GameMode.Join(&c.Player) // may set client's team
	s.Clients.SendWelcome(c)   // tells client about her team
	s.GameMode.Init(&c.Player) // may send additional welcome info like flags
	s.Clients.InformOthersOfJoin(c)

	go func() {
		uniqueName := s.Clients.UniqueName(c)
		country := utils.CountryByIP(c.Peer.Address.IP)
		if country != "" {
			s.Clients.Relay(c, nmc.ServerMessage, fmt.Sprintf("%s connected from %s", uniqueName, country))
		}

		c.Send(nmc.ServerMessage, s.MessageOfTheDay)

		log.Println(cubecode.SanitizeString(fmt.Sprintf("%s (%s) connected", uniqueName, c.Peer.Address.IP)))
	}()

	go c.Send(nmc.RequestAuth, s.StatsServerAuthDomain)
}

func (s *Server) Broadcast(typ nmc.ID, args ...interface{}) {
	s.Clients.Broadcast(typ, args...)
}

func (s *Server) Send(p *game.Player, typ nmc.ID, args ...interface{}) {
	s.Clients.GetClientByCN(p.CN).Send(typ, args...)
}

func (s *Server) UniqueName(p *game.Player) string {
	return s.Clients.UniqueName(s.Clients.GetClientByCN(p.CN))
}

func (s *Server) Spawn(client *Client) {
	client.Spawn()
	s.GameMode.Spawn(&client.Player)
}

func (s *Server) ConfirmSpawn(client *Client, lifeSequence, _weapon int32) {
	if client.State != playerstate.Dead || lifeSequence != client.LifeSequence || client.LastSpawnAttempt.IsZero() {
		// client may not spawn
		return
	}

	client.State = playerstate.Alive
	client.SelectedWeapon = weapon.ByID(weapon.ID(_weapon))
	client.LastSpawnAttempt = time.Time{}

	client.Packets.Publish(nmc.ConfirmSpawn, client.ToWire())

	s.GameMode.ConfirmSpawn(&client.Player)
}

func (s *Server) Disconnect(client *Client, reason disconnectreason.ID) {
	s.GameMode.Leave(&client.Player)
	s.relay.RemoveClient(client.CN)
	s.Clients.Disconnect(client, reason)
	s.ENetHost.Disconnect(client.Peer, disconnectreason.None)
	client.Reset()
	if len(s.Clients.PrivilegedUsers()) == 0 {
		s.Unsupervised()
	}
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
	s.Clients.Broadcast(nmc.ServerMessage, msg)
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
	s.Clients.Broadcast(nmc.ServerMessage, msg)
	s.Disconnect(victim, disconnectreason.Kick)
}

func (s *Server) Unsupervised() {
	s.GameMode.Resume(nil)
	s.MasterMode = mastermode.Open
	s.KeepTeams = false
	s.CompetitiveMode = false
	s.ReportStats = true
}

func (s *Server) Empty() {
	s.MapRotation.queue = s.MapRotation.queue[:0]
	if s.GameMode.ID() != s.FallbackGameMode {
		s.ChangeMap(s.FallbackGameMode, s.MapRotation.NextMap(NewGame(s.FallbackGameMode), s.Map))
	}
}

func (s *Server) Intermission() {
	s.GameMode.End()

	nextMap := s.MapRotation.NextMap(s.GameMode, s.Map)

	// start 5 second timer
	s.PendingMapChange = time.AfterFunc(5*time.Second, func() {
		s.ChangeMap(s.GameMode.ID(), nextMap)
	})

	s.Clients.Broadcast(nmc.ServerMessage, "next up: "+nextMap)

	if s.ReportStats {
		go s.ReportEndgameStats()
	}
}

func (s *Server) ReportEndgameStats() {
	stats := []string{}
	s.Clients.ForEach(func(c *Client) {
		if a, ok := c.Authentications[s.StatsServerAuthDomain]; ok {
			stats = append(stats, fmt.Sprintf("%d %s %d %d %d %d %d", a.reqID, a.name, c.Frags, c.Deaths, c.Damage, c.DamagePotential, c.Flags))
		}
	})

	statsAuth.Send("stats %d %s %s", s.GameMode.ID(), s.Map, strings.Join(stats, " "))
}

func (s *Server) HandleSuccStats(reqID uint32) {
	s.Clients.ForEach(func(c *Client) {
		if a, ok := c.Authentications[s.StatsServerAuthDomain]; ok && a.reqID == reqID {
			c.Send(nmc.ServerMessage, fmt.Sprintf("your game statistics were reported to %s", s.StatsServerAuthDomain))
		}
	})
}

func (s *Server) HandleFailStats(reqID uint32, reason string) {
	s.Clients.ForEach(func(c *Client) {
		if a, ok := c.Authentications[s.StatsServerAuthDomain]; ok && a.reqID == reqID {
			c.Send(nmc.ServerMessage, fmt.Sprintf("reporting your game statistics failed: %s", reason))
		}
	})
}

func (s *Server) ChangeMap(mode gamemode.ID, mapname string) {
	// cancel pending timers
	if s.GameMode != nil {
		s.GameMode.CleanUp()
	}

	// stop any pending map change
	if s.PendingMapChange != nil {
		s.PendingMapChange.Stop()
	}

	s.Map = mapname
	s.GameMode = NewGame(mode)

	s.ForEach(s.GameMode.Join)
	s.Clients.Broadcast(nmc.MapChange, s.Map, s.GameMode.ID(), s.GameMode.NeedMapInfo())
	s.GameMode.Start()
	s.Clients.MapChange()

	s.Clients.Broadcast(nmc.ServerMessage, s.MessageOfTheDay)
}

func (s *Server) SetMasterMode(c *Client, mm mastermode.ID) {
	if mm < mastermode.Open || mm > mastermode.Private {
		log.Println("invalid mastermode", mm, "requested")
		return
	}
	if c.Role == role.None {
		c.Send(nmc.ServerMessage, cubecode.Fail("you can't do that"))
		return
	}
	s.MasterMode = mm
	s.Clients.Broadcast(nmc.MasterMode, mm)
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
	client.LastShot = time.Now()
	client.DamagePotential += wpn.Damage * wpn.Rays // TODO: quad damage
	switch wpn.ID {
	case weapon.GrenadeLauncher, weapon.RocketLauncher:
		// TODO: save somewhere
	default:
		// apply damage
		rays := int32(0)
		for _, h := range hits {
			target := s.Clients.GetClientByCN(h.target)
			if target == nil ||
				target.State != playerstate.Alive ||
				target.LifeSequence != h.lifeSequence ||
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
			target.State != playerstate.Alive ||
			target.LifeSequence != h.lifeSequence ||
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
	victim.ApplyDamage(&attacker.Player, damage, wpnID, dir)
	s.Clients.Broadcast(nmc.Damage, victim.CN, attacker.CN, damage, victim.Armour, victim.Health)
	// TODO: setpushed ???
	if !dir.IsZero() {
		dir = dir.Scale(geom.DNF)
		typ, p := nmc.HitPush, []interface{}{victim.CN, wpnID, damage, dir.X(), dir.Y(), dir.Z()}
		if victim.Health <= 0 {
			s.Clients.Broadcast(typ, p...)
		} else {
			victim.Send(typ, p...)
		}
	}
	if victim.Health <= 0 {
		s.GameMode.HandleFrag(&attacker.Player, &victim.Player)
	}
}

func (s *Server) ForEach(f func(p *game.Player)) {
	s.Clients.ForEach(func(c *Client) {
		f(&c.Player)
	})
}
