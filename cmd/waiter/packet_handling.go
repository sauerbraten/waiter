package main

import (
	"log"
	"time"

	"github.com/sauerbraten/waiter/cubecode"

	"github.com/sauerbraten/waiter/internal/client/playerstate"
	"github.com/sauerbraten/waiter/internal/definitions/disconnectreason"
	"github.com/sauerbraten/waiter/internal/definitions/nmc"
	"github.com/sauerbraten/waiter/internal/definitions/weapon"
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

			hash, ok := p.GetString()
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

			if !s.IsAllowedToJoin(client, hash, authDomain, authName) {
				s.Clients.Disconnect(client, disconnectreason.Password) // TODO: check if correct
			}

			s.Clients.Join(client, name, playerModel)
			s.Clients.SendWelcome(client)
			s.Clients.InformOthersOfJoin(client)
			client.Peer.Send(1, enet.PACKET_FLAG_RELIABLE, packet.Encode(nmc.ServerMessage, s.MessageOfTheDay))

		case nmc.AuthTry:
			log.Println("got auth try:", p)
			domain, ok := p.GetString()
			if !ok {
				log.Println("could not read domain from auth try packet:", p)
				continue
			}
			if domain == "" {
				s.handleGlobalAuthRequest(client, &p)
			} else {
				s.handleAuthRequest(client, domain, &p)
			}

		case nmc.AuthAnswer:
			log.Println("got auth answer:", p)

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

		case nmc.Spawn:
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
			if client.TryToSpawn(lifeSequence, weapon.Weapon(_weapon)) {
				client.Packets.Publish(nmc.Spawn, client.GameState.ToWire())
			}

		case nmc.ChangeWeapon:
			// player changing weapon
			_weapon, ok := p.GetInt()
			if !ok {
				log.Println("could not read weapon ID from weapon change packet:", p)
				return
			}
			requestedWeapon := weapon.Weapon(_weapon)
			selectedWeapon, ok := client.GameState.SelectWeapon(requestedWeapon)
			if !ok {
				break
			}
			client.Packets.Publish(nmc.ChangeWeapon, selectedWeapon)

		case nmc.Shoot:
			id, ok := p.GetInt()
			if !ok {
				log.Println("could not read shot ID from shoot packet:", p)
				return
			}

			weapon, ok := p.GetInt()
			if !ok {
				log.Println("could not read weapon ID from shoot packet:", p)
				return
			}

			// TODO: check weapon reload time
			// TODO: check ammo
			// TODO: check weapon range

			from := [3]float32{}
			for i := 0; i < 3; i++ {
				coord, ok := p.GetInt()
				if !ok {
					log.Println("could not read shot origin ('from') from shoot packet:", p)
					return
				}
				from[i] = float32(coord) / 16.0
			}

			to := [3]float32{}
			for i := 0; i < 3; i++ {
				coord, ok := p.GetInt()
				if !ok {
					log.Println("could not read shot destination ('to') from shoot packet:", p)
					return
				}
				to[i] = float32(coord) / 16.0
			}

			numHits, ok := p.GetInt()
			if !ok {
				log.Println("could not read number of hits from shoot packet:", p)
				return
			}

			log.Println("processed shot with id =", id, "weapon =", weapon, "from =", from, "to =", to, "numHits =", numHits)

			for i := int32(0); i < numHits; i++ {
				target, ok := p.GetInt()
				if !ok {
					log.Println("could not read target of hit", i+1, "from shoot packet:", p)
					return
				}

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
				distance := float32(_distance) / 16.0

				rays, ok := p.GetInt()
				if !ok {
					log.Println("could not read rays of hit", i+1, "from shoot packet:", p)
					return
				}

				dir := [3]float32{}
				for i := 0; i < 3; i++ {
					angle, ok := p.GetInt()
					if !ok {
						log.Println("could not read shot destination ('to') from shoot packet:", p)
						return
					}
					dir[i] = float32(angle) / 100.0
				}

				log.Println("  hit =", i+1, "target =", target, "life sequence =", lifeSequence, "distance =", distance, "rays =", rays, "dir =", dir)
			}

			s.Clients.Broadcast(exclude(client), 1, enet.PACKET_FLAG_RELIABLE,
				nmc.ShotEffects,
				client.CN,
				weapon,
				id,
				int32(from[0]*16.0),
				int32(from[1]*16.0),
				int32(from[2]*16.0),
				int32(to[0]*16.0),
				int32(to[1]*16.0),
				int32(to[2]*16.0),
			)

			client.GameState.LastShot = time.Now()

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
				log.Println("pausing game at", s.TimeLeft/100, "seconds left")
				s.Clients.Broadcast(nil, 1, enet.PACKET_FLAG_RELIABLE, nmc.PAUSEGAME, 1, client.CN)
				s.Pause()
			} else {
				log.Println("resuming game at", s.TimeLeft/100, "seconds left")
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
