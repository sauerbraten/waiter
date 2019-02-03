package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/sauerbraten/waiter/protocol"
	"github.com/sauerbraten/waiter/protocol/cubecode"

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
		packetType := nmc.NetMessCode(_nmc)

		if !client.IsValidMessage(packetType) {
			log.Println("invalid network message code", packetType, "from CN", client.CN)
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
			s.Clients.SendWelcome(client)
			s.Clients.InformOthersOfJoin(client)
			client.Peer.Send(1, enet.PACKET_FLAG_RELIABLE, packet.Encode(nmc.ServerMessage, s.MessageOfTheDay))
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
			if cn == client.CN && toggle != 0 {
				client.Peer.Send(1, enet.PACKET_FLAG_RELIABLE, packet.Encode(nmc.ServerMessage, cubecode.Fail("server only supports claiming master using /auth")))
				return
			}
			target := s.Clients.GetClientByCN(cn)
			if target == nil {
				client.Peer.Send(1, enet.PACKET_FLAG_RELIABLE, packet.Encode(nmc.ServerMessage, cubecode.Fail(fmt.Sprintf("no client with CN %d", cn))))
				return
			}
			if client != target && client.Privilege <= target.Privilege {
				client.Peer.Send(1, enet.PACKET_FLAG_RELIABLE, packet.Encode(nmc.ServerMessage, cubecode.Fail("you can't do that")))
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
				client.Peer.Send(1, enet.PACKET_FLAG_RELIABLE, packet.Encode(nmc.ServerMessage, cubecode.Fail("you can't do that")))
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
			if strings.HasPrefix(msg, "!rev") {
				client.Peer.Send(1, enet.PACKET_FLAG_NONE, packet.Encode(nmc.ServerMessage, "running "+gitRevision))
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
			client.GameState.Spawn(s.GameMode.ID())
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
			s.handleSuicide(client)

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
	wpn, ok = weapon.ByID[weapon.ID(weaponID)]
	if !ok {
		log.Println("invalid weapon ID in shoot packet:", weaponID)
		return
	}
	if time.Now().Before(client.GameState.GunReloadEnd) || client.GameState.Ammo[wpn.ID] <= 0 {
		return
	}
	_from, ok := parseVector(p)
	if !ok {
		log.Println("could not read shot origin vector ('from') from shoot packet:", p)
		return
	}
	from = _from.Mul(1 / geom.DMF)
	_to, ok := parseVector(p)
	if !ok {
		log.Println("could not read shot destination vector ('to') from shoot packet:", p)
		return
	}
	to = _to.Mul(1 / geom.DMF)
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
	wpn, ok = weapon.ByID[weapon.ID(weaponID)]
	id, ok = p.GetInt()
	if !ok {
		log.Println("could not read explosion ID from explode packet:", p)
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
			log.Println("could not read", "xzy"[i], "coordinate from packet:", p)
			return nil, false
		}
		xyz[i] = float64(coord)
	}
	return geom.NewVector(xyz[0], xyz[1], xyz[2]), true
}
