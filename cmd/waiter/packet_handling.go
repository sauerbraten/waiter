package main

import (
	"fmt"
	"log"
	"time"

	"github.com/sauerbraten/waiter/internal/definitions/disconnectreason"
	"github.com/sauerbraten/waiter/internal/definitions/gamemode"
	"github.com/sauerbraten/waiter/internal/definitions/mastermode"
	"github.com/sauerbraten/waiter/internal/definitions/nmc"
	"github.com/sauerbraten/waiter/internal/definitions/playerstate"
	"github.com/sauerbraten/waiter/internal/definitions/role"
	"github.com/sauerbraten/waiter/internal/definitions/weapon"
	"github.com/sauerbraten/waiter/internal/geom"
	"github.com/sauerbraten/waiter/internal/net/packet"
	"github.com/sauerbraten/waiter/pkg/protocol"
	"github.com/sauerbraten/waiter/pkg/protocol/cubecode"
)

// parses a packet and decides what to do based on the network message code at the front of the packet
func (s *Server) handlePacket(client *Client, channelID uint8, p protocol.Packet) {
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
		packetType := nmc.ID(_nmc)

		if !isValidMessage(client, packetType) {
			log.Println("invalid network message code", packetType, "from CN", client.CN)
			s.Disconnect(client, disconnectreason.MessageError)
			return
		}

		switch packetType {

		// channel 0 traffic

		case nmc.Position:
			// client sending his position and movement in the world
			if client.GameState.State == playerstate.Alive {
				q := p
				client.Position.Publish(packet.Encode(nmc.Position, q))
				parsePosition(client, &p)
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
			if client.GameState.State == playerstate.Alive {
				s.relay.FlushPositionAndSend(client.CN, packet.Encode(nmc.JumpPad, cn, jumppad))
			}

		case nmc.Teleport:
			_cn, ok := p.GetInt()
			if !ok {
				log.Println("could not read CN from teleport packet (packet too short):", p)
				return
			}
			cn := uint32(_cn)
			if cn != client.CN {
				// we don't support bots
				return
			}
			teleport, ok := p.GetInt()
			if !ok {
				log.Println("could not read teleport ID from teleport packet (packet too short):", p)
				return
			}
			teledest, ok := p.GetInt()
			if !ok {
				log.Println("could not read teledest ID from teleport packet (packet too short):", p)
				return
			}
			if client.GameState.State == playerstate.Alive {
				s.relay.FlushPositionAndSend(client.CN, packet.Encode(nmc.Teleport, cn, teleport, teledest))
			}

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
			s.GameMode.Join(client)
			s.Clients.SendWelcome(client)
			s.GameMode.Init(client)
			s.Clients.InformOthersOfJoin(client)
			client.Send(nmc.ServerMessage, s.MessageOfTheDay)
			if authDomain != "" && authName != "" {
				s.handleAuthRequest(client, authDomain, authName)
			}

		case nmc.AuthTry:
			// client wants us to send him a challenge
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
			switch toggle {
			case 0:
				s.setRole(client, cn, role.None)
			default:
				s.setRole(client, cn, role.Master)
			}

		case nmc.Kick:
			_cn, ok := p.GetInt()
			if !ok {
				log.Println("could not read cn from kick packet:", p)
				return
			}
			cn := uint32(_cn)
			reason, ok := p.GetString()
			if !ok {
				log.Println("could not read reason from kick packet:", p)
				return
			}
			victim := s.Clients.GetClientByCN(cn)
			if client.Role <= victim.Role {
				client.Send(nmc.ServerMessage, cubecode.Fail("you can't do that"))
				return
			}
			if reason != "" {
				s.Clients.Broadcast(nil, nmc.ServerMessage, fmt.Sprintf("%s kicked %s for: %s", s.Clients.UniqueName(client), s.Clients.UniqueName(victim), reason))
			} else {
				s.Clients.Broadcast(nil, nmc.ServerMessage, fmt.Sprintf("%s kicked %s", s.Clients.UniqueName(client), s.Clients.UniqueName(victim)))
			}
			s.Disconnect(victim, disconnectreason.Kick)

		case nmc.MasterMode:
			log.Println(p)
			_mm, ok := p.GetInt()
			if !ok {
				log.Println("could not read mastermode from mastermode packet:", p)
				return
			}
			mm := mastermode.ID(_mm)
			if mm < mastermode.Open || mm > mastermode.Private {
				log.Println("invalid mastermode", mm, "requested")
				return
			}
			if client.Role == role.None {
				client.Send(nmc.ServerMessage, cubecode.Fail("you can't do that"))
				return
			}
			s.MasterMode = mm
			s.Clients.Broadcast(nil, nmc.MasterMode, mm)

		case nmc.Spectator:
			_spectator, ok := p.GetInt()
			if !ok {
				log.Println("could not read CN from spectator packet:", p)
				return
			}
			spectator := s.Clients.GetClientByCN(uint32(_spectator))
			if spectator == nil {
				return
			}
			toggle, ok := p.GetInt()
			if !ok {
				log.Println("could not read toggle from spectator packet:", p)
				return
			}
			if client.Role == role.None {
				// unprivileged clients can never change spec state of others
				if spectator != client {
					client.Send(nmc.ServerMessage, cubecode.Fail("you can't do that"))
					return
				}
				// unprivileged clients can not unspec themselves in mm>=2
				if client.GameState.State == playerstate.Spectator && s.MasterMode >= mastermode.Locked {
					client.Send(nmc.ServerMessage, cubecode.Fail("you can't do that"))
					return
				}
			}
			if (spectator.GameState.State == playerstate.Spectator) == (toggle != 0) {
				// nothing to do
				return
			}
			if toggle != 0 {
				if client.GameState.State == playerstate.Alive {
					s.handleDeath(spectator, spectator)
				}
				s.GameMode.Leave(spectator)
				spectator.GameState.State = playerstate.Spectator
			} else {
				spectator.GameState.State = playerstate.Dead
				s.GameMode.Join(spectator)
				// todo: checkmap
			}
			s.Clients.Broadcast(nil, nmc.Spectator, spectator.CN, toggle)

		case nmc.VoteMap:
			mapp, ok := p.GetString()
			if !ok {
				log.Println("could not read map from map vote packet:", p)
				return
			}
			if mapp == "" {
				mapp = s.Map
			}

			_mode, ok := p.GetInt()
			if !ok {
				log.Println("could not read mode from map vote packet:", p)
				return
			}
			mode := gamemode.ID(_mode)

			if !gamemode.Valid(mode) {
				client.Send(nmc.ServerMessage, cubecode.Fail(fmt.Sprintf("%s is not implemented on this server", mode)))
				log.Println("invalid gamemode", mode, "requested")
				return
			}

			if s.MasterMode < mastermode.Veto {
				client.Send(nmc.ServerMessage, cubecode.Fail("this server does not support map voting"))
				return
			}

			if client.Role < role.Master {
				client.Send(nmc.ServerMessage, cubecode.Fail("you can't do that"))
				return
			}

			s.ChangeMap(mode, mapp)
			log.Println(client, "forced", mode, "on", mapp)

		case nmc.Ping:
			// client pinging server → send pong
			ping, ok := p.GetInt()
			if !ok {
				log.Println("could not read ping from ping packet:", p)
				return
			}
			client.Send(nmc.Pong, ping)

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
			// client sending team chat message → pass on to team immediately
			msg, ok := p.GetString()
			if !ok {
				log.Println("could not read message from team chat message packet:", p)
				return
			}
			s.Clients.SendToTeam(client, nmc.TeamChatMessage, client.CN, msg)

		case nmc.ChangeName:
			newName, ok := p.GetString()
			if !ok {
				log.Println("could not read name from name change packet:", p)
				return
			}

			newName = cubecode.Filter(newName, false)

			if len(newName) == 0 || len(newName) > 20 {
				return
			}

			client.Name = newName
			client.Packets.Publish(nmc.ChangeName, newName)

		case nmc.MapCRC:
			// client sends crc hash of his map file
			// TODO
			//clientMapName := p.GetString()
			//clientMapCRC := p.GetInt32()
			p.GetString()
			p.GetInt()
			log.Println("todo: MAPCRC")

		case nmc.TrySpawn:
			if !client.Joined || client.GameState.State != playerstate.Dead || !client.GameState.LastSpawnAttempt.IsZero() || !s.GameMode.CanSpawn(client) {
				return
			}
			client.GameState.Spawn(s.GameMode.ID())
			client.Send(nmc.SpawnState, client.CN, client.GameState.ToWire())

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

			if client.GameState.State != playerstate.Dead || lifeSequence != client.GameState.LifeSequence || client.GameState.LastSpawnAttempt.IsZero() {
				// client may not spawn
				return
			}

			client.GameState.State = playerstate.Alive
			client.GameState.SelectedWeapon = weapon.ByID(weapon.ID(_weapon))
			client.GameState.LastSpawnAttempt = time.Time{}

			client.Packets.Publish(nmc.ConfirmSpawn, client.GameState.ToWire())

		case nmc.ChangeWeapon:
			// player changing weapon
			_requested, ok := p.GetInt()
			if !ok {
				log.Println("could not read weapon ID from weapon change packet:", p)
				return
			}
			requested := weapon.ID(_requested)
			selected, ok := client.GameState.SelectWeapon(requested)
			if !ok {
				break
			}
			client.Packets.Publish(nmc.ChangeWeapon, selected.ID)

		case nmc.Shoot:
			wpn, id, from, to, hits, ok := parseShoot(client, &p)
			if !ok {
				return
			}
			s.HandleShoot(client, wpn, id, from, to, hits)

		case nmc.Explode:
			millis, wpn, id, hits, ok := parseExplode(client, &p)
			if !ok {
				return
			}
			s.HandleExplode(client, millis, wpn, id, hits)

		case nmc.Suicide:
			s.handleDeath(client, client)

		case nmc.Sound:
			sound, ok := p.GetInt()
			if !ok {
				log.Println("could not read sound ID from sound packet:", p)
				return
			}
			client.Packets.Publish(nmc.Sound, sound)

		case nmc.PauseGame:
			if s.MasterMode < mastermode.Locked {
				if client.Role == role.None {
					return
				}
			}
			pause, ok := p.GetInt()
			if !ok {
				log.Println("could not read pause toggle from pause packet:", p)
				return
			}
			if pause == 1 {
				log.Println("pausing game at", s.timer.TimeLeft/1000, "seconds left")
				s.Clients.Broadcast(nil, nmc.PauseGame, 1, client.CN)
				s.timer.Pause()
			} else {
				s.timer.ResumeWithCountdown(client)
			}

		case nmc.ItemList:
			// TODO: process and broadcast itemlist so clients are ok

		default:
			log.Println("received", packetType, p, "on channel", channelID)
			ok := s.GameMode.HandlePacket(client, packetType, &p)
			if !ok {
				log.Println("could not handle packet", packetType, p, "on channel", channelID)
				break outer
			}
		}
	}

	return
}

// checks if the client is allowed to send a certain type of message to us.
func isValidMessage(c *Client, networkMessageCode nmc.ID) bool {
	if !c.Joined {
		if c.AuthRequiredBecause > disconnectreason.None {
			return networkMessageCode == nmc.AuthAnswer || networkMessageCode == nmc.Ping
		}
		return networkMessageCode == nmc.Join || networkMessageCode == nmc.Ping
	} else if networkMessageCode == nmc.Join {
		return false
	}

	for _, soNMC := range nmc.ServerOnlyNMCs {
		if soNMC == networkMessageCode {
			return false
		}
	}

	return true
}

func parsePosition(client *Client, p *protocol.Packet) {
	// parse position out of packet
	_, ok := p.GetUint() // we don't support bots so we know it's from the client themselves
	if !ok {
		log.Println("could not read CN from position packet (packet too short):", p)
		return
	}
	p.GetByte() // state, not used
	flags, ok := p.GetByte()

	xyz := [3]float64{}
	for i := range xyz {
		c1, ok := p.GetByte()
		if !ok {
			log.Printf("could not read first byte of %s coordinate from position packet (packet too short): %v", string("xyz"[i]), p)
			return
		}
		c2, ok := p.GetByte()
		if !ok {
			log.Printf("could not read second byte of %s coordinate from position packet (packet too short): %v", string("xyz"[i]), p)
			return
		}
		c := int32(c1) | int32(c2)<<8
		if flags&(1<<uint(i)) != 0 {
			c3, ok := p.GetByte()
			if !ok {
				log.Printf("could not read third byte of %s coordinate from position packet (packet too short): %v", string("xyz"[i]), p)
				return
			}
			c |= int32(c3) << 16
			if c&0x800000 != 0 {
				c |= -16777216 // 0xFF000000
			}
		}
		xyz[i] = float64(c)
	}
	client.CurrentPos = geom.NewVector(xyz[0], xyz[1], xyz[2]).Mul(1 / geom.DMF)

	// rest of packet is not neede yet
}

func parseShoot(client *Client, p *protocol.Packet) (wpn weapon.Weapon, id int32, from, to *geom.Vector, hits []hit, success bool) {
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
	wpn = weapon.ByID(weapon.ID(weaponID))
	if time.Now().Before(client.GameState.GunReloadEnd) || client.GameState.Ammo[wpn.ID] <= 0 {
		return
	}
	from, ok = parseVector(p)
	if !ok {
		log.Println("could not read shot origin vector ('from') from shoot packet:", p)
		return
	}
	from = from.Mul(1 / geom.DMF)
	to, ok = parseVector(p)
	if !ok {
		log.Println("could not read shot destination vector ('to') from shoot packet:", p)
		return
	}
	to = to.Mul(1 / geom.DMF)
	if dist := geom.Distance(from, to); dist > wpn.Range+1.0 {
		log.Println("shot distance out of weapon's range: distane =", dist, "range =", wpn.Range+1)
	}
	numHits, ok := p.GetInt()
	if !ok {
		log.Println("could not read number of hits from shoot packet:", p)
		return
	}
	hits, success = parseHits(numHits, p)
	return
}

func parseExplode(client *Client, p *protocol.Packet) (millis int32, wpn weapon.Weapon, id int32, hits []hit, success bool) {
	millis, ok := p.GetInt()
	if !ok {
		log.Println("could not read millis from explode packet:", p)
		return
	}
	weaponID, ok := p.GetInt()
	if !ok {
		log.Println("could not read weapon ID from explode packet:", p)
		return
	}
	wpn = weapon.ByID(weapon.ID(weaponID))
	_, ok = p.GetInt() // TODO: use projectile ID to link to shot
	if !ok {
		log.Println("could not read projectile ID from explode packet:", p)
		return
	}
	numHits, ok := p.GetInt()
	if !ok {
		log.Println("could not read number of hits from explode packet:", p)
		return
	}
	hits, success = parseHits(numHits, p)
	return
}

func parseHits(num int32, p *protocol.Packet) (hits []hit, ok bool) {
	hits = make([]hit, num)
	for i := range hits {
		_target, ok := p.GetInt()
		if !ok {
			log.Println("could not read target of hit", i+1, "from shoot/explode packet:", p)
			return nil, false
		}
		target := uint32(_target)
		lifeSequence, ok := p.GetInt()
		if !ok {
			log.Println("could not read life sequence of hit", i+1, "from shoot/explode packet:", p)
			return nil, false
		}
		_distance, ok := p.GetInt()
		if !ok {
			log.Println("could not read distance of hit", i+1, "from shoot/explode packet:", p)
			return nil, false
		}
		distance := float64(_distance) / geom.DMF
		rays, ok := p.GetInt()
		if !ok {
			log.Println("could not read rays of hit", i+1, "from shoot/explode packet:", p)
			return nil, false
		}
		_dir, ok := parseVector(p)
		if !ok {
			log.Println("could not read direction vector of hit", i+1, "from shoot/explode packet:", p)
			return nil, false
		}
		dir := _dir.Mul(1 / geom.DNF)
		hits[i] = hit{
			target:       target,
			lifeSequence: lifeSequence,
			distance:     distance,
			rays:         rays,
			dir:          dir,
		}
	}
	return hits, true
}

func parseVector(p *protocol.Packet) (*geom.Vector, bool) {
	xyz := [3]float64{}
	for i := range xyz {
		coord, ok := p.GetInt()
		if !ok {
			log.Printf("could not read %s coordinate from packet: %v", string("xzy"[i]), p)
			return nil, false
		}
		xyz[i] = float64(coord)
	}
	return geom.NewVector(xyz[0], xyz[1], xyz[2]), true
}
