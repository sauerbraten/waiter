package main

import (
	"fmt"
	"log"
	"time"

	"github.com/sauerbraten/waiter/cubecode"
	"github.com/sauerbraten/waiter/cubecode/sstrings"

	"github.com/sauerbraten/waiter/internal/client/playerstate"
	"github.com/sauerbraten/waiter/internal/client/privilege"
	"github.com/sauerbraten/waiter/internal/definitions/disconnectreason"
	"github.com/sauerbraten/waiter/internal/definitions/mastermode"
	"github.com/sauerbraten/waiter/internal/definitions/nmc"
	"github.com/sauerbraten/waiter/internal/definitions/weapon"
	"github.com/sauerbraten/waiter/internal/geom"
	"github.com/sauerbraten/waiter/internal/protocol/enet"
	"github.com/sauerbraten/waiter/internal/protocol/packet"
)

// parses a packet and decides what to do based on the network message code at the front of the packet
func (s *Server) handlePacket(client *Client, channelID uint8, p cubecode.Packet) {
	// this implementation does not support channel 2 (for coop edit purposes) yet.
	if client == nil || channelID > 1 {
		return
	}

outer:
	for len(p) > 0 {
		_nmc, ok := p.GetInt()
		if !ok {
			log.Println("could not read network message code (packet too short):", p)
			return
		}
		packetType := nmc.NetMessCode(_nmc)

		if !client.IsValidMessage(packetType) {
			log.Println("invalid network message code:", packetType)
			s.Clients.Disconnect(client, disconnectreason.MessageError)
			return
		}

		switch packetType {

		// channel 0 traffic

		case nmc.Position:
			// client sending his position and movement in the world
			if client.GameState.State == playerstate.Alive {
				client.Position.Publish(packet.Encode(nmc.Position, p))
			}
			break outer

		case nmc.JumpPad:
			cn, ok := p.GetInt()
			if !ok {
				log.Println("could not read CN from jump pad packet (packet too short):", p)
				return
			}
			jumppad, ok := p.GetInt()
			if !ok {
				log.Println("could not read jump pad ID from jump pad packet (packet too short):", p)
				return
			}
			s.relay.BroadcastAfterPosition(client.CN, packet.Encode(nmc.JumpPad, cn, jumppad))

		// channel 1 traffic

		case nmc.Join:
			name, ok := p.GetString()
			if !ok {
				log.Println("could not read name from join packet:", p)
				return
			}
			playerModel, ok := p.GetInt()
			if !ok {
				log.Println("could not read player model ID from join packet:", p)
				return
			}
			_, ok = p.GetString() // this server does not support a server password
			if !ok {
				log.Println("could not read hash from join packet:", p)
				return
			}
			authDomain, ok := p.GetString()
			if !ok {
				log.Println("could not read auth domain from join packet:", p)
				return
			}
			authName, ok := p.GetString()
			if !ok {
				log.Println("could not read auth name from join packet:", p)
				return
			}
			s.Clients.Join(client, name, playerModel)
			s.Clients.SendWelcome(client)
			s.Clients.InformOthersOfJoin(client)
			client.Peer.Send(1, enet.PACKET_FLAG_RELIABLE, packet.Encode(nmc.ServerMessage, s.MessageOfTheDay))
			if authDomain != "" && authName != "" {
				s.handleAuthRequest(client, authDomain, authName)
			}

		case nmc.AuthTry:
			log.Println("got auth try:", p)
			domain, ok := p.GetString()
			if !ok {
				log.Println("could not read domain from auth try packet:", p)
				continue
			}
			name, ok := p.GetString()
			if !ok {
				log.Println("could not read name from auth try packet:", p)
				return
			}
			if domain == "" {
				s.handleGlobalAuthRequest(client, name)
			} else {
				s.handleAuthRequest(client, domain, name)
			}

		case nmc.AuthAnswer:
			// client sends answer to auth challenge
			domain, ok := p.GetString()
			if !ok {
				log.Println("could not read domain from auth answer packet:", p)
				return
			}
			if domain == "" {
				s.handleGlobalAuthAnswer(client, &p)
			} else {
				s.handleAuthAnswer(client, domain, &p)
			}

		case nmc.SetMaster:
			_cn, ok := p.GetInt()
			if !ok {
				log.Println("could not read cn from setmaster packet:", p)
				return
			}
			cn := uint32(_cn)
			toggle, ok := p.GetInt()
			if !ok {
				log.Println("could not read toggle from setmaster packet:", p)
				return
			}
			_, ok = p.GetString() // password is not used in this implementation, only auth
			if !ok {
				log.Println("could not read password from setmaster packet:", p)
				return
			}
			if cn == client.CN && toggle != 0 {
				client.Peer.Send(1, enet.PACKET_FLAG_RELIABLE, packet.Encode(nmc.ServerMessage, sstrings.Fail("server only supports claiming master using /auth")))
				return
			}
			target := s.Clients.GetClientByCN(cn)
			if target == nil {
				client.Peer.Send(1, enet.PACKET_FLAG_RELIABLE, packet.Encode(nmc.ServerMessage, sstrings.Fail(fmt.Sprintf("no client with CN %d", cn))))
				return
			}
			if client != target && client.Privilege <= target.Privilege {
				client.Peer.Send(1, enet.PACKET_FLAG_RELIABLE, packet.Encode(nmc.ServerMessage, sstrings.Fail("you can't do that")))
				return
			}
			switch toggle {
			case 0:
				oldPrivilege := target.Privilege
				s.setPrivilege(target, privilege.None)
				var msg string
				if client != target {
					msg = fmt.Sprintf("%s took away %s privileges from %s", s.Clients.UniqueName(client), oldPrivilege, s.Clients.UniqueName(target))
				} else {
					msg = fmt.Sprintf("%s relinquished %s", s.Clients.UniqueName(client), oldPrivilege)
				}
				s.Clients.Broadcast(nil, 1, enet.PACKET_FLAG_RELIABLE, nmc.ServerMessage, msg)

			default:
				s.setPrivilege(target, privilege.Master)
				s.Clients.Broadcast(nil, 1, enet.PACKET_FLAG_RELIABLE, nmc.ServerMessage, fmt.Sprintf("%s gave %s privileges to %s", s.Clients.UniqueName(client), privilege.Master, s.Clients.UniqueName(target)))
			}

		case nmc.MasterMode:
			_mm, ok := p.GetInt()
			if !ok {
				log.Println("could not read mastermode from mastermode packet:", p)
				return
			}
			mm := mastermode.MasterMode(_mm)
			if mm < mastermode.Open || mm > mastermode.Private {
				log.Println("invalid mastermode", mm, "requested")
				return
			}
			if client.Privilege == privilege.None {
				client.Peer.Send(1, enet.PACKET_FLAG_RELIABLE, packet.Encode(nmc.ServerMessage, sstrings.Fail("you can't do that")))
				return
			}
			s.MasterMode = mm
			s.Clients.Broadcast(nil, 1, enet.PACKET_FLAG_RELIABLE, nmc.MasterMode, mm)

		case nmc.Ping:
			// client pinging server → send pong
			ping, ok := p.GetInt()
			if !ok {
				log.Println("could not read ping from ping packet:", p)
				return
			}
			client.Peer.Send(1, enet.PACKET_FLAG_NONE, packet.Encode(nmc.Pong, ping))

		case nmc.ClientPing:
			// client sending the amount of lag he measured to the server → broadcast to other clients
			ping, ok := p.GetInt()
			if !ok {
				log.Println("could not read ping from client ping packet:", p)
				return
			}
			client.Ping = ping
			client.Packets.Publish(nmc.ClientPing, client.Ping)

		case nmc.ChatMessage:
			// client sending chat message → broadcast to other clients
			msg, ok := p.GetString()
			if !ok {
				log.Println("could not read message from chat message packet:", p)
				return
			}
			client.Packets.Publish(nmc.ChatMessage, msg)

		case nmc.TeamChatMessage:
			// client sending team chat message → pass on to team immediatly
			msg, ok := p.GetString()
			if !ok {
				log.Println("could not read message from team chat message packet:", p)
				return
			}
			s.Clients.SendToTeam(client, 1, enet.PACKET_FLAG_RELIABLE, packet.Encode(nmc.TeamChatMessage, client.CN, msg))

		case nmc.MAPCRC:
			// client sends crc hash of his map file
			// TODO
			//clientMapName := p.GetString()
			//clientMapCRC := p.GetInt32()
			p.GetString()
			p.GetInt()
			log.Println("todo: MAPCRC")

		case nmc.TrySpawn:
			if client.GameState.State != playerstate.Dead || !client.GameState.LastSpawn.IsZero() {
				return
			}
			client.GameState.Respawn()
			client.GameState.Spawn(s.GameMode)
			client.Peer.Send(1, enet.PACKET_FLAG_RELIABLE, packet.Encode(nmc.SpawnState, client.CN, client.GameState.ToWire()))

		case nmc.ConfirmSpawn:
			lifeSequence, ok := p.GetInt()
			if !ok {
				log.Println("could not read life sequence from spawn packet:", p)
				return
			}
			_weapon, ok := p.GetInt()
			if !ok {
				log.Println("could not read weapon ID from spawn packet:", p)
				return
			}

			if (client.GameState.State != playerstate.Alive && client.GameState.State != playerstate.Dead) || lifeSequence != client.GameState.LifeSequence || client.GameState.LastSpawn.IsZero() {
				// client may not spawn
				return
			}

			client.GameState.State = playerstate.Alive
			client.GameState.SelectedWeapon = weapon.ByID[weapon.ID(_weapon)]
			client.GameState.LastSpawn = time.Time{}

			client.Packets.Publish(nmc.ConfirmSpawn, client.GameState.ToWire())

		case nmc.ChangeWeapon:
			// player changing weapon
			_weapon, ok := p.GetInt()
			if !ok {
				log.Println("could not read weapon ID from weapon change packet:", p)
				return
			}
			requested := weapon.ID(_weapon)
			selected, ok := client.GameState.SelectWeapon(requested)
			if !ok {
				break
			}
			client.Packets.Publish(nmc.ChangeWeapon, selected.ID)

		case nmc.Shoot:
			id, ok := p.GetInt()
			if !ok {
				log.Println("could not read shot ID from shoot packet:", p)
				return
			}

			weaponID, ok := p.GetInt()
			if !ok {
				log.Println("could not read weapon ID from shoot packet:", p)
				return
			}
			wpn, ok := weapon.ByID[weapon.ID(weaponID)]
			if !ok {
				log.Println("invalid weapon ID in shoot packet:", weaponID)
				return
			}

			if time.Now().Before(client.GameState.GunReloadEnd) || client.GameState.Ammo[wpn.ID] <= 0 {
				continue
			}

			from := &geom.Vector{}
			for i := 0; i < 3; i++ {
				coord, ok := p.GetInt()
				if !ok {
					log.Println("could not read shot origin ('from') from shoot packet:", p)
					return
				}
				from[i] = float64(coord) / 16.0
			}

			to := &geom.Vector{}
			for i := 0; i < 3; i++ {
				coord, ok := p.GetInt()
				if !ok {
					log.Println("could not read shot destination ('to') from shoot packet:", p)
					return
				}
				to[i] = float64(coord) / 16.0
			}

			if dist := geom.Distance(from, to); dist > wpn.Range+1.0 {
				log.Println("shot distance out of weapon's range: distane =", dist, "range =", wpn.Range+1)
			}

			numHits, ok := p.GetInt()
			if !ok {
				log.Println("could not read number of hits from shoot packet:", p)
				return
			}

			hits := make([]hit, 0, numHits)
			for i := int32(0); i < numHits; i++ {
				_target, ok := p.GetInt()
				if !ok {
					log.Println("could not read target of hit", i+1, "from shoot packet:", p)
					return
				}
				target := uint32(_target)

				lifeSequence, ok := p.GetInt()
				if !ok {
					log.Println("could not read life sequence of hit", i+1, "from shoot packet:", p)
					return
				}

				_distance, ok := p.GetInt()
				if !ok {
					log.Println("could not read distance of hit", i+1, "from shoot packet:", p)
					return
				}
				distance := float64(_distance) / 16.0

				rays, ok := p.GetInt()
				if !ok {
					log.Println("could not read rays of hit", i+1, "from shoot packet:", p)
					return
				}

				dir := &geom.Vector{}
				for i := 0; i < 3; i++ {
					angle, ok := p.GetInt()
					if !ok {
						log.Println("could not read shot destination ('to') from shoot packet:", p)
						return
					}
					dir[i] = float64(angle) / 100.0
				}

				hits = append(hits, hit{
					target:       target,
					lifeSequence: lifeSequence,
					distance:     distance,
					rays:         rays,
					dir:          dir,
				})
			}

			s.Clients.Broadcast(exclude(client), 1, enet.PACKET_FLAG_RELIABLE,
				nmc.ShotEffects,
				client.CN,
				wpn.ID,
				id,
				int32(from[0]*geom.DMF),
				int32(from[1]*geom.DMF),
				int32(from[2]*geom.DMF),
				int32(to[0]*geom.DMF),
				int32(to[1]*geom.DMF),
				int32(to[2]*geom.DMF),
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
					target.applyDamage(client, damage, wpn.ID, h.dir)
					s.Clients.Broadcast(nil, 1, enet.PACKET_FLAG_RELIABLE, nmc.Damage, target.CN, client.CN, damage, target.GameState.Armour, target.GameState.Health)
					// TODO: setpushed ???
					h.dir.Scale(geom.DNF)
					p := []interface{}{nmc.HitPush, target.CN, wpn.ID, damage, h.dir[0], h.dir[1], h.dir[2]}
					if target.GameState.Health <= 0 {
						s.Clients.Broadcast(nil, 1, enet.PACKET_FLAG_RELIABLE, p...)
						s.HandleDeath(client, target)
					} else {
						client.Peer.Send(1, enet.PACKET_FLAG_RELIABLE, packet.Encode(p...))
					}
				}
			}
			// TODO: apply damage of hits

		case nmc.Sound:
			sound, ok := p.GetInt()
			if !ok {
				log.Println("could not read sound ID from sound packet:", p)
				return
			}
			client.Packets.Publish(nmc.Sound, sound)

		case nmc.PAUSEGAME:
			// TODO: check client privilege
			pause, ok := p.GetInt()
			if !ok {
				log.Println("could not read pause toggle from pause packet:", p)
				return
			}
			if pause == 1 {
				log.Println("pausing game at", s.TimeLeft/1000, "seconds left")
				s.Clients.Broadcast(nil, 1, enet.PACKET_FLAG_RELIABLE, nmc.PAUSEGAME, 1, client.CN)
				s.Pause()
			} else {
				log.Println("resuming game at", s.TimeLeft/1000, "seconds left")
				s.Clients.Broadcast(nil, 1, enet.PACKET_FLAG_RELIABLE, nmc.PAUSEGAME, 0, client.CN)
				s.Resume()
			}

		case nmc.ItemList:
			// TODO: process and broadcast itemlist so clients are ok

		default:
			log.Println("received", packetType, p, "on channel", channelID)
			break outer
		}
	}

	return
}

type hit struct {
	target       uint32
	lifeSequence int32
	distance     float64
	rays         int32
	dir          *geom.Vector
}
