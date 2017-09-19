package main

import (
	"log"
	"runtime"
	"strconv"
	"time"

	"github.com/sauerbraten/jsonfile"

	"github.com/sauerbraten/waiter/internal/bans"
	"github.com/sauerbraten/waiter/internal/broadcast"
	"github.com/sauerbraten/waiter/internal/client"
	"github.com/sauerbraten/waiter/internal/enet"
	"github.com/sauerbraten/waiter/internal/extinfo"
	"github.com/sauerbraten/waiter/internal/masterserver"
	"github.com/sauerbraten/waiter/internal/protocol/definitions/disconnectreason"
	"github.com/sauerbraten/waiter/internal/protocol/definitions/gamemode"
	"github.com/sauerbraten/waiter/internal/protocol/definitions/mastermode"
	"github.com/sauerbraten/waiter/internal/protocol/definitions/nmc"
	"github.com/sauerbraten/waiter/internal/protocol/packet"
	"github.com/sauerbraten/waiter/internal/server"
	"github.com/sauerbraten/waiter/internal/server/config"
)

var (
	// global enet host var (to call Flush() on)
	host *enet.Host

	// global variable to indicate to the main loop that there are packets to be sent
	mustFlush = false

	// global server state
	s *server.Server

	// client manager
	cm *client.ClientManager

	// ban manager
	bm *bans.BanManager

	// master server
	ms *masterserver.MasterServer
)

func init() {
	var conf *config.Config
	err := jsonfile.ParseFile("config.json", &conf)
	if err != nil {
		log.Fatalln(err)
	}

	runtime.GOMAXPROCS(conf.CPUCores)

	bm, err = bans.FromFile("bans.json")
	if err != nil {
		log.Fatalln(err)
	}

	cm = &client.ClientManager{}

	s = &server.Server{
		Config: conf,
		State: &server.State{
			MasterMode:  mastermode.Open,
			GameMode:    gamemode.Effic,
			Map:         "hashi",
			TimeLeft:    MAP_TIME,
			NotGotItems: true,
			HasMaster:   false,
			UpSince:     time.Now(),
			NumClients:  cm.NumberOfClientsConnected,
		},
	}

	ms, err = masterserver.New(s.Config.MasterServerAddress+":"+strconv.Itoa(s.Config.MasterServerPort), bm)
	if err != nil {
		log.Fatalln(err)
	}
}

func main() {
	var err error
	host, err = enet.NewHost(s.Config.ListenAddress, s.Config.ListenPort)
	if err != nil {
		log.Fatalln(err)
	}

	log.Println("server running on port", s.Config.ListenPort)

	infoServer := extinfo.NewInfoServer(s, cm)

	go infoServer.ServeStateInfo()

	go countDown()

	log.Println("registering at master server")
	err = ms.Register(s.Config.ListenPort)
	if err != nil {
		log.Println(err)
	}

	buildChannel0Packet := func(c *client.Client) *packet.Packet {
		p := packet.New(c.GameState.Position, c.QueuedBroadcastMessages[0])
		c.ClearBroadcastMessageQueue(0)
		return p
	}

	buildChannel1Packet := func(c *client.Client) *packet.Packet {
		p := c.QueuedBroadcastMessages[1]
		if p.Len() == 0 {
			return packet.Empty
		}

		p = packet.New(nmc.Client, c.CN, p.Len(), p)
		c.ClearBroadcastMessageQueue(1)
		return p
	}

	go broadcast.Forever(33*time.Millisecond, enet.PACKET_FLAG_NO_ALLOCATE, 0, cm, buildChannel0Packet)
	go broadcast.Forever(33*time.Millisecond, enet.PACKET_FLAG_NO_ALLOCATE|enet.PACKET_FLAG_RELIABLE, 1, cm, buildChannel1Packet)

	for {
		event := host.Service(2)

		switch event.Type {
		case enet.EVENT_TYPE_CONNECT:
			log.Println("ENet: connected:", event.Peer.Address.String())
			client := cm.Add(event.Peer)

			err := event.Peer.SetData(&client.CN)
			if err != nil {
				log.Println("enet:", err)
			}

			if ban, ok := bm.GetBan(event.Peer.Address.IP); ok {
				log.Println("peer's IP is banned:", ban)
				event.Peer.Disconnect(uint32(disconnectreason.Banned))
				continue
			}

			client.SendServerConfig(s.Config)

		case enet.EVENT_TYPE_DISCONNECT:
			log.Println("ENet: disconnected:", event.Peer.Address.String())
			client := cm.GetClientByCN(*(*int32)(event.Peer.Data))
			client.Leave()

		case enet.EVENT_TYPE_RECEIVE:
			// TODO: fix this maybe?
			if len(event.Packet.Data) == 0 {
				continue
			}

			handlePacket(*(*int32)(event.Peer.Data), event.ChannelId, packet.New(event.Packet.Data))
		}

		host.Flush()
	}
}
