package main

import (
	"log"

	"github.com/sauerbraten/waiter/internal/client/playerstate"
	"github.com/sauerbraten/waiter/internal/enet"
	"github.com/sauerbraten/waiter/internal/protocol/definitions/disconnectreason"
	"github.com/sauerbraten/waiter/internal/protocol/definitions/nmc"
	"github.com/sauerbraten/waiter/internal/protocol/definitions/weapon"
	"github.com/sauerbraten/waiter/internal/protocol/packet"
)

// parses a packet and decides what to do based on the network message code at the front of the packet
func handlePacket(fromCN int32, channelID uint8, p *packet.Packet) {
	// this implementation does not support channel 2 (for coop edit purposes) yet.
	if fromCN < 0 || channelID > 1 {
		return
	}

	client := cm.GetClientByCN(fromCN)

outer:
	for p.HasRemaining() {
		packetType := nmc.NetMessCode(p.GetInt32())

		if !client.IsValidMessage(packetType) {
			log.Println("invalid network message code:", packetType)
			client.Disconnect(disconnectreason.MessageError)
			return
		}

		switch packetType {

		// channel 0 traffic

		case nmc.Position:
			// client sending his position and movement in the world
			if client.GameState.State == playerstate.Alive {
				client.GameState.Position = p
			}
			break outer

		case nmc.JumpPad:
			client.QueuedBroadcastMessages[0].Put(nmc.JumpPad, p.GetInt32(), p.GetInt32())
			log.Println("processed JUMPPAD")

		// channel 1 traffic

		case nmc.Join:
			// client sends intro and wants to join the game
			if client.TryToJoin(p.GetString(), p.GetInt32(), p.GetString(), p.GetString(), p.GetString(), s) {
				// send welcome packet
				client.SendWelcome(s.State)

				// inform other clients that a new client joined
				client.InformOthersOfJoin()
			}
			log.Println("processed JOIN")

		case nmc.AUTHTRY:
			//domain, name := p.GetString(), p.GetString()

		case nmc.AUTHANS:
			// client sends answer to auth challenge
			log.Println("received AUTHANS")

		case nmc.Ping:
			// client pinging server → send pong
			client.Send(enet.PACKET_FLAG_NONE, 1, packet.New(nmc.Pong, p.GetInt32()))
			//log.Println("processed PING")

		case nmc.ClientPing:
			// client sending the amount of lag he measured to the server → broadcast to other clients
			client.Ping = p.GetInt32()
			client.QueuedBroadcastMessages[1].Put(nmc.ClientPing, client.Ping)
			//log.Println("processed CLIENTPING")

		case nmc.ChatMessage:
			// client sending chat message → broadcast to other clients
			client.QueuedBroadcastMessages[1].Put(nmc.ChatMessage, p.GetString())
			log.Println("processed TEXT")

		case nmc.TeamChatMessage:
			// client sending team chat message → pass on to team immediatly
			client.SendToTeam(enet.PACKET_FLAG_RELIABLE, 1, packet.New(nmc.TeamChatMessage, client.CN, p.GetString()))
			log.Println("processed SAYTEAM")

		case nmc.MAPCRC:
			// client sends crc hash of his map file
			// TODO
			//clientMapName := p.GetString()
			//clientMapCRC := p.GetInt32()
			p.GetString()
			p.GetInt32()
			log.Println("processed MAPCRC")

		case nmc.Spawn:
			if client.TryToSpawn(p.GetInt32(), weapon.Weapon(p.GetInt32())) {
				client.QueuedBroadcastMessages[1].Put(nmc.Spawn, client.GameState.ToWire())
			}
			log.Println("processed SPAWN")

		case nmc.ChangeWeapon:
			// player changing weapon
			selectedWeapon := weapon.Weapon(p.GetInt32())
			client.GameState.SelectWeapon(selectedWeapon)

			// broadcast to other clients
			client.QueuedBroadcastMessages[1].Put(nmc.ChangeWeapon, selectedWeapon)
			//log.Println("processed WEAPONSELECT")

		case nmc.Sound:
			client.QueuedBroadcastMessages[1].Put(nmc.Sound, p.GetInt32())
			log.Println("processed SOUND")

		default:
			log.Println("received", packetType, p.Bytes(), "on channel", channelID)
			break outer
		}
	}

	return
}
