package extinfo

import (
	"log"
	"net"
	"strconv"
	"time"

	"github.com/sauerbraten/waiter/internal/client"
	"github.com/sauerbraten/waiter/internal/protocol"
	"github.com/sauerbraten/waiter/internal/protocol/packet"
	"github.com/sauerbraten/waiter/internal/server"
	"github.com/sauerbraten/waiter/internal/utils"
)

// Protocol constants
const (
	// Constants describing the type of information to query for
	InfoTypeExtended int32 = 0

	// Constants used in responses to extended info queries
	ExtInfoACK     int32 = -1  // EXT_ACK
	ExtInfoVersion       = 105 // EXT_VERSION
	ExtInfoNoError       = 0   // EXT_NO_ERROR
	ExtInfoError         = 1   // EXT_ERROR

	// Constants describing the type of extended information to query for
	ExtInfoTypeUptime     int32 = 0 // EXT_UPTIME
	ExtInfoTypeClientInfo       = 1 // EXT_PLAYERSTATS
	ExtInfoTypeTeamScores       = 2 // EXT_TEAMSCORE

	// Constants used in responses to client info queries
	ClientInfoResponseTypeCNs  int32 = -10 // EXT_PLAYERSTATS_RESP_IDS
	ClientInfoResponseTypeInfo       = -11 // EXT_PLAYERSTATS_RESP_STATS
)

type InfoServer struct {
	serv *server.Server
	cm   *client.ClientManager
}

func NewInfoServer(serv *server.Server, clientManager *client.ClientManager) *InfoServer {
	return &InfoServer{
		serv: serv,
		cm:   clientManager,
	}
}
func (is *InfoServer) ServeStateInfo() {
	// listen for incoming traffic
	laddr, err := net.ResolveUDPAddr("udp", is.serv.Config.ListenAddress+":"+strconv.Itoa(is.serv.Config.ListenPort+1))
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
		buf := make([]byte, 16)
		n, raddr, err := conn.ReadFromUDP(buf)
		if err != nil {
			log.Println(err)
			continue
		}

		if n > 5 {
			log.Println("malformed info request:", buf)
			continue
		}

		// process requests

		p := packet.New(buf)

		reqType := p.GetInt32()

		switch reqType {
		case InfoTypeExtended:
			extReqType := p.GetInt32()
			switch extReqType {
			case ExtInfoTypeUptime:
				is.sendUptime(conn, raddr)
			case ExtInfoTypeClientInfo:
				is.sendPlayerStats(p.GetInt32(), conn, raddr)
			case ExtInfoTypeTeamScores:
				// TODO
			default:
				log.Println("erroneous extinfo type queried:", reqType)
			}
		default:
			// basic info was requested → reqType is the ping we need to play back to the client
			is.sendBasicInfo(conn, raddr, reqType)
		}
	}
}

func (is *InfoServer) sendBasicInfo(conn *net.UDPConn, raddr *net.UDPAddr, pong int32) {
	log.Println("basic info requested by", raddr.String())

	p := packet.New(pong)

	p.Put(is.serv.State.NumClients())
	p.Put(5) // this implementation never sends information about the server being paused or not and the gamespeed
	p.Put(protocol.Version)
	p.Put(is.serv.State.GameMode)
	p.Put(is.serv.State.TimeLeft / 1000)
	p.Put(is.serv.Config.MaxClients)
	p.Put(is.serv.State.MasterMode)
	p.Put(is.serv.State.Map)
	p.Put(is.serv.Config.ServerDescription)

	n, err := conn.WriteToUDP(p.Bytes(), raddr)
	if err != nil {
		log.Println(err)
	}

	if n != p.Len() {
		log.Println("packet length and sent length didn't match!", p.Bytes())
	}
}

func (is *InfoServer) sendUptime(conn *net.UDPConn, raddr *net.UDPAddr) {
	p := packet.New(0, ExtInfoTypeUptime, ExtInfoACK, ExtInfoVersion, int(time.Since(is.serv.State.UpSince)/time.Second))
	n, err := conn.WriteToUDP(p.Bytes(), raddr)
	if err != nil {
		log.Println(err)
	}

	if n != p.Len() {
		log.Println("packet length and sent length didn't match!", p.Bytes())
	}
}

func (is *InfoServer) sendPlayerStats(cn int32, conn *net.UDPConn, raddr *net.UDPAddr) {
	p := packet.New(0, ExtInfoTypeClientInfo, cn, ExtInfoACK, ExtInfoVersion)

	if cn < -1 || int(cn) > is.cm.NumClients() {
		p.Put(ExtInfoError)

		n, err := conn.WriteToUDP(p.Bytes(), raddr)
		if err != nil {
			log.Println(err)
		}

		if n != p.Len() {
			log.Println("packet length and sent length didn't match!", p.Bytes())
		}

		return
	}

	p.Put(ExtInfoNoError)

	n, err := conn.WriteToUDP(p.Bytes(), raddr)
	if err != nil {
		log.Println(err)
	}

	if n != p.Len() {
		log.Println("packet length and sent length didn't match!", p.Bytes())
	}

	p.Clear()

	p.Put(ClientInfoResponseTypeCNs)

	if cn == -1 {
		is.cm.ForEach(func(c *client.Client) {
			if c.Joined {
				p.Put(c.CN)
			}
		})
	} else {
		p.Put(cn)
	}

	n, err = conn.WriteToUDP(p.Bytes(), raddr)
	if err != nil {
		log.Println(err)
	}

	if n != p.Len() {
		log.Println("packet length and sent length didn't match!", p.Bytes())
	}

	p.Clear()

	if cn == -1 {
		is.cm.ForEach(func(c *client.Client) {
			if !c.Joined {
				return
			}
			p.Put(ClientInfoResponseTypeInfo, c.CN, c.Ping, c.Name, c.Team, c.GameState.Frags, c.GameState.Flags, c.GameState.Deaths, c.GameState.Teamkills, c.GameState.Damage*100/utils.Max(c.GameState.ShotDamage, 1), c.GameState.Health, c.GameState.Armour, c.GameState.SelectedWeapon, c.Privilege, c.GameState.State)
			if is.serv.Config.SendClientIPsViaExtinfo {
				p.Put(c.Peer.Address.IP[:2]) // only 3 first bytes
			} else {
				p.Put(0, 0, 0) // 3 times 0x0
			}

			n, err = conn.WriteToUDP(p.Bytes(), raddr)
			if err != nil {
				log.Println(err)
			}

			if n != p.Len() {
				log.Println("packet length and sent length didn't match!", p.Bytes())
			}

			p.Clear()
		})
	} else {
		c := is.cm.GetClientByCN(cn)
		p.Put(ClientInfoResponseTypeInfo, c.CN, c.Ping, c.Name, c.Team, c.GameState.Frags, c.GameState.Flags, c.GameState.Deaths, c.GameState.Teamkills, c.GameState.Damage*100/utils.Max(c.GameState.ShotDamage, 1), c.GameState.Health, c.GameState.Armour, c.GameState.SelectedWeapon, c.Privilege, c.GameState.State)

		if is.serv.Config.SendClientIPsViaExtinfo {
			p.Put(c.Peer.Address.IP[:2]) // only 3 first bytes
		} else {
			p.Put(0, 0, 0) // 3 times 0x0
		}

		n, err = conn.WriteToUDP(p.Bytes(), raddr)
		if err != nil {
			log.Println(err)
		}

		if n != p.Len() {
			log.Println("packet length and sent length didn't match!", p.Bytes())
		}
	}
}
