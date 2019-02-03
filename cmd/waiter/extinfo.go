package main

import (
	"log"
	"net"
	"strconv"
	"time"

	"github.com/sauerbraten/waiter/protocol"

	"github.com/sauerbraten/waiter/internal/protocol/packet"
	"github.com/sauerbraten/waiter/internal/utils"
)

// Protocol constants
const (
	// Constants describing the type of information to query for
	InfoTypeExtended int32 = 0

	// Constants used in responses to extended info queries
	ExtInfoACK     int32 = -1  // EXT_ACK
	ExtInfoVersion int32 = 105 // EXT_VERSION
	ExtInfoNoError int32 = 0   // EXT_NO_ERROR
	ExtInfoError   int32 = 1   // EXT_ERROR

	// Constants describing the type of extended information to query for
	ExtInfoTypeUptime     int32 = 0 // EXT_UPTIME
	ExtInfoTypeClientInfo int32 = 1 // EXT_PLAYERSTATS
	ExtInfoTypeTeamScores int32 = 2 // EXT_TEAMSCORE

	// Constants used in responses to client info queries
	ClientInfoResponseTypeCNs  int32 = -10 // EXT_PLAYERSTATS_RESP_IDS
	ClientInfoResponseTypeInfo int32 = -11 // EXT_PLAYERSTATS_RESP_STATS

	// ID to identify this server mod via extinfo
	ServerMod int32 = -9
)

type ExtInfoServer struct {
	*Config
	*State
	*GameTimer
	Clients *ClientManager
}

func (eis *ExtInfoServer) ServeStateInfoForever() {
	// listen for incoming traffic
	laddr, err := net.ResolveUDPAddr("udp", eis.ListenAddress+":"+strconv.Itoa(eis.ListenPort+1))
	if err != nil {
		log.Println(err)
		return
	}

	conn, err := net.ListenUDP("udp", laddr)
	if err != nil {
		log.Println(err)
		return
	}
	defer conn.Close()

	log.Println("listening for info requests on", laddr.String())

	for {
		p := make(protocol.Packet, 16)
		n, raddr, err := conn.ReadFromUDP(p)
		if err != nil {
			log.Println(err)
			continue
		}
		if n > 5 {
			log.Println("malformed info request:", p)
			continue
		}
		p = p[:n]

		// process requests

		reqType, ok := p.GetInt()
		if !ok {
			log.Println("extinfo: info request packet too short: could not read request type:", p)
			continue
		}

		switch reqType {
		case InfoTypeExtended:
			extReqType, ok := p.GetInt()
			if !ok {
				log.Println("malformed info request: could not read extinfo request type:", p)
				continue
			}
			switch extReqType {
			case ExtInfoTypeUptime:
				includeMod := false
				if len(p) != 0 {
					b, ok := p.GetByte()
					includeMod = ok && b > 0
				}
				eis.sendUptime(conn, raddr, includeMod)
			case ExtInfoTypeClientInfo:
				cn, ok := p.GetInt()
				if !ok {
					log.Println("malformed info request: could not read CN from client info request:", p)
					continue
				}
				log.Println("client info requested for", cn)
				eis.sendPlayerStats(cn, conn, raddr)
			case ExtInfoTypeTeamScores:
				// TODO
			default:
				log.Println("erroneous extinfo type queried:", reqType)
			}
		default:
			// basic info was requested → reqType is the ping we need to play back to the client
			eis.sendBasicInfo(conn, raddr, reqType)
		}
	}
}

func (eis *ExtInfoServer) sendBasicInfo(conn *net.UDPConn, raddr *net.UDPAddr, pong int32) {
	log.Println("basic info requested by", raddr.String())

	q := []interface{}{
		pong,
		eis.NumClients(),
	}

	if eis.IsPaused() {
		q = append(q, 7)
	} else {
		q = append(q, 5)
	}
	q = append(q,
		protocol.Version,
		eis.GameMode.ID(),
		eis.TimeLeft/1000,
		eis.MaxClients,
		eis.MasterMode,
	)
	if eis.IsPaused() {
		q = append(q,
			eis.IsPaused(), // paused?
			100,            // gamespeed
		)
	}
	q = append(q, eis.Map, eis.ServerDescription)

	p := packet.Encode(q...)
	n, err := conn.WriteToUDP(p, raddr)
	if err != nil {
		log.Println(err)
	}

	if n != len(p) {
		log.Println("packet length and sent length didn't match!", p)
	}
}

func (eis *ExtInfoServer) sendUptime(conn *net.UDPConn, raddr *net.UDPAddr, includeMod bool) {
	q := []interface{}{
		InfoTypeExtended,
		ExtInfoTypeUptime,
		ExtInfoACK,
		ExtInfoVersion,
		int32(time.Since(eis.UpSince) / time.Second),
	}

	if includeMod {
		q = append(q, ServerMod)
	}

	p := packet.Encode(q...)

	n, err := conn.WriteToUDP(p, raddr)
	if err != nil {
		log.Println(err)
	}

	if n != len(p) {
		log.Println("packet length and sent length didn't match!", p)
	}
}

func (eis *ExtInfoServer) sendPlayerStats(cn int32, conn *net.UDPConn, raddr *net.UDPAddr) {
	q := []interface{}{
		InfoTypeExtended,
		ExtInfoTypeClientInfo,
		cn,
		ExtInfoACK,
		ExtInfoVersion,
	}

	if cn < -1 || int(cn) > eis.NumClients() {
		q = append(q, ExtInfoError)
		p := packet.Encode(q...)

		n, err := conn.WriteToUDP(p, raddr)
		if err != nil {
			log.Println(err)
		}

		if n != len(p) {
			log.Println("packet length and sent length didn't match!", p)
		}

		return
	}

	q = append(q, ExtInfoNoError)

	headerLen := len(q)

	q = append(q, ClientInfoResponseTypeCNs)

	if cn == -1 {
		eis.Clients.ForEach(func(c *Client) { q = append(q, c.CN) })
	} else {
		q = append(q, cn)
	}

	p := packet.Encode(q...)
	n, err := conn.WriteToUDP(p, raddr)
	if err != nil {
		log.Println(err)
	}

	if n != len(p) {
		log.Println("packet length and sent length didn't match!", p)
	}

	q = q[:headerLen]
	p = nil

	if cn == -1 {
		eis.Clients.ForEach(func(c *Client) {
			q = append(q,
				ClientInfoResponseTypeInfo,
				c.CN,
				c.Ping,
				c.Name,
				c.Team,
				c.GameState.Frags,
				c.GameState.Flags,
				c.GameState.Deaths,
				c.GameState.Teamkills,
				c.GameState.Damage*100/utils.Max(c.GameState.ShotDamage, 1),
				c.GameState.Health,
				c.GameState.Armour,
				c.GameState.SelectedWeapon.ID,
				c.Privilege,
				c.GameState.State,
			)
			if eis.SendClientIPsViaExtinfo {
				q = append(q, []byte(c.Peer.Address.IP[:3])) // only 3 first bytes
			} else {
				q = append(q, 0, 0, 0) // 3 times 0x0
			}

			p = packet.Encode(q...)
			n, err = conn.WriteToUDP(p, raddr)
			if err != nil {
				log.Println(err)
			}

			if n != len(p) {
				log.Println("packet length and sent length didn't match!", p)
			}

			q = q[:headerLen]
			p = nil
		})
	} else {
		c := eis.Clients.GetClientByCN(uint32(cn))
		q = append(q,
			ClientInfoResponseTypeInfo,
			c.CN,
			c.Ping,
			c.Name,
			c.Team,
			c.GameState.Frags,
			c.GameState.Flags,
			c.GameState.Deaths,
			c.GameState.Teamkills,
			c.GameState.Damage*100/utils.Max(c.GameState.ShotDamage, 1),
			c.GameState.Health,
			c.GameState.Armour,
			c.GameState.SelectedWeapon,
			c.Privilege,
			c.GameState.State,
		)

		if eis.SendClientIPsViaExtinfo {
			q = append(q, c.Peer.Address.IP[:2]) // only 3 first bytes
		} else {
			q = append(q, 0, 0, 0) // 3 times 0x0
		}

		p = packet.Encode(q...)

		n, err = conn.WriteToUDP(p, raddr)
		if err != nil {
			log.Println(err)
		}

		if n != len(p) {
			log.Println("packet length and sent length didn't match!", p)
		}
	}
}
