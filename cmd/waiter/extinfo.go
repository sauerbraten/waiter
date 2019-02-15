package main

import (
	"log"
	"net"
	"strconv"
	"time"

	"github.com/sauerbraten/waiter/internal/net/packet"
	"github.com/sauerbraten/waiter/internal/utils"
	"github.com/sauerbraten/waiter/pkg/protocol"
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

func (s *Server) handleExtinfoRequests() {
	laddr, err := net.ResolveUDPAddr("udp", s.ListenAddress+":"+strconv.Itoa(s.ListenPort+1))
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
		req := make(protocol.Packet, 16)
		n, raddr, err := conn.ReadFromUDP(req)
		if err != nil {
			log.Println(err)
			continue
		}
		if n > 5 {
			log.Println("malformed info request:", req[:n], "from", raddr)
			continue
		}
		req = req[:n]

		// prepare response header (we need to replay the request)
		respHeader := req

		// interpret request as packet
		p := protocol.Packet(req)

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
				s.send(conn, raddr, s.uptime(respHeader))
			case ExtInfoTypeClientInfo:
				cn, ok := p.GetInt()
				if !ok {
					log.Println("malformed info request: could not read CN from client info request:", p)
					continue
				}
				s.send(conn, raddr, s.clientInfo(cn, respHeader)...)
			case ExtInfoTypeTeamScores:
				s.send(conn, raddr, s.teamScores(respHeader))
			default:
				log.Println("erroneous extinfo type queried:", reqType)
			}
		default:
			s.send(conn, raddr, s.basicInfo(respHeader))
		}
	}
}

func (s *Server) send(conn *net.UDPConn, raddr *net.UDPAddr, packets ...protocol.Packet) {
	for _, p := range packets {
		n, err := conn.WriteToUDP(p, raddr)
		if err != nil {
			log.Println(err)
		}

		if n != len(p) {
			log.Println("packet length and sent length didn't match!", p)
		}
	}
}

func (s *Server) basicInfo(respHeader []byte) protocol.Packet {
	q := []interface{}{
		respHeader,
		s.NumClients(),
	}

	paused := s.timer.Paused()

	if paused {
		q = append(q, 7)
	} else {
		q = append(q, 5)
	}

	q = append(q,
		protocol.Version,
		s.GameMode.ID(),
		s.timer.TimeLeft/1000,
		s.MaxClients,
		s.MasterMode,
	)

	if paused {
		q = append(q,
			paused, // paused?
			100,    // gamespeed
		)
	}

	q = append(q, s.Map, s.ServerDescription)

	return packet.Encode(q...)
}

func (s *Server) uptime(respHeader []byte) protocol.Packet {
	q := []interface{}{
		respHeader,
		ExtInfoACK,
		ExtInfoVersion,
		int32(time.Since(s.UpSince) / time.Second),
	}

	if len(respHeader) > 2 {
		q = append(q, ServerMod)
	}

	return packet.Encode(q...)
}

func (s *Server) clientInfo(cn int32, respHeader []byte) (packets []protocol.Packet) {
	q := []interface{}{
		respHeader,
		ExtInfoACK,
		ExtInfoVersion,
	}

	if cn < -1 || int(cn) > s.NumClients() {
		q = append(q, ExtInfoError)
		packets = append(packets, packet.Encode(q...))
		return
	}

	q = append(q, ExtInfoNoError)

	header := q

	q = append(q, ClientInfoResponseTypeCNs)

	if cn == -1 {
		s.Clients.ForEach(func(c *Client) { q = append(q, c.CN) })
	} else {
		q = append(q, cn)
	}

	packets = append(packets, packet.Encode(q...))

	if cn == -1 {
		s.Clients.ForEach(func(c *Client) {
			packets = append(packets, s.clientPacket(c, header))
		})
	} else {
		c := s.Clients.GetClientByCN(uint32(cn))
		packets = append(packets, s.clientPacket(c, header))
	}

	return
}

func (s *Server) clientPacket(c *Client, header []interface{}) protocol.Packet {
	q := header

	q = append(q,
		ClientInfoResponseTypeInfo,
		c.CN,
		c.Ping,
		c.Name,
		c.Team.Name,
		c.GameState.Frags,
		c.GameState.Flags,
		c.GameState.Deaths,
		c.GameState.Teamkills,
		c.GameState.Damage*100/utils.Max(c.GameState.ShotDamage, 1),
		c.GameState.Health,
		c.GameState.Armour,
		c.GameState.SelectedWeapon.ID,
		c.Role,
		c.GameState.State,
	)

	if s.SendClientIPsViaExtinfo {
		q = append(q, []byte(c.Peer.Address.IP.To4()[:3]))
	} else {
		q = append(q, 0, 0, 0)
	}

	return packet.Encode(q...)
}

func (s *Server) teamScores(respHeader []byte) protocol.Packet {
	q := []interface{}{
		respHeader,
		ExtInfoACK,
		ExtInfoVersion,
	}

	teamMode, isTeamMode := s.GameMode.(TeamMode)
	if isTeamMode {
		q = append(q, ExtInfoNoError)
	} else {
		q = append(q, ExtInfoError)
	}

	q = append(q, s.GameMode.ID(), s.timer.TimeLeft/1000)

	if !isTeamMode {
		return packet.Encode(q...)
	}

	captureMode, isCaptureMode := teamMode.(CaptureMode)

	for name, team := range teamMode.Teams() {
		if team.Score <= 0 && len(team.Players) == 0 {
			continue
		}
		q = append(q, name, team.Score)
		if isCaptureMode {
			bases := captureMode.Bases(team)
			q = append(q, len(bases), bases)
		} else {
			q = append(q, -1)
		}
	}

	return packet.Encode(q...)
}
