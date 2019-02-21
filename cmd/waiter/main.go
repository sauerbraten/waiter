package main

import (
	"log"
	"strconv"
	"time"

	"github.com/sauerbraten/jsonfile"

	"github.com/sauerbraten/waiter/internal/auth"
	"github.com/sauerbraten/waiter/internal/definitions/disconnectreason"
	"github.com/sauerbraten/waiter/internal/masterserver"
	"github.com/sauerbraten/waiter/internal/net/enet"
	"github.com/sauerbraten/waiter/pkg/bans"
	"github.com/sauerbraten/waiter/pkg/protocol"
)

var (
	// global server state
	s *Server

	// ban manager
	bm *bans.BanManager

	// master server
	ms *masterserver.MasterServer
)

func init() {
	var conf *Config
	err := jsonfile.ParseFile("config.json", &conf)
	if err != nil {
		log.Fatalln(err)
	}

	bm, err = bans.FromFile("bans.json")
	if err != nil {
		log.Fatalln(err)
	}

	var users []*auth.User
	err = jsonfile.ParseFile("users.json", &users)
	if err != nil {
		log.Fatalln(err)
	}

	var mr MapRotation
	err = jsonfile.ParseFile("maps.json", &mr)
	if err != nil {
		log.Fatalln(err)
	}

	cs := &ClientManager{}

	s = &Server{
		Config: conf,
		State: &State{
			UpSince:    time.Now(),
			NumClients: cs.NumberOfClientsConnected,
		},
		relay:       NewRelay(),
		Clients:     cs,
		Auth:        auth.NewManager(users),
		MapRotation: &mr,
	}
	s.GameDuration = s.GameDuration * time.Minute // duration is parsed without unit from config file
	s.GameMode = NewGame(conf.FallbackGameMode)
	s.Map = s.MapRotation.NextMap(s.GameMode, "")
	s.GameMode.Start()

	ms, err = masterserver.New(s.Config.MasterServerAddress+":"+strconv.Itoa(s.Config.MasterServerPort), s.Config.ListenPort, bm)
	if err != nil {
		log.Println("could not connect to master server:", err)
	}
}

func main() {
	host, err := enet.NewHost(s.Config.ListenAddress, s.Config.ListenPort)
	if err != nil {
		log.Fatalln(err)
	}

	log.Println("server running on port", s.Config.ListenPort)

	go s.handleExtinfoRequests()

	go s.relay.loop()

	for {
		event := host.Service()

		switch event.Type {
		case enet.EVENT_TYPE_CONNECT:
			s.Connect(event.Peer)

		case enet.EVENT_TYPE_DISCONNECT:
			client := s.Clients.GetClientByPeer(event.Peer)
			if client == nil {
				continue
			}
			s.Disconnect(client, disconnectreason.None)

		case enet.EVENT_TYPE_RECEIVE:
			// TODO: fix this maybe?
			if len(event.Packet.Data) == 0 {
				log.Println("received empty packet on channel", event.ChannelID, "from", event.Peer.Address)
				continue
			}

			client := s.Clients.GetClientByPeer(event.Peer)
			if client == nil {
				continue
			}

			s.handlePacket(client, event.ChannelID, protocol.Packet(event.Packet.Data))
		}

		host.Flush()
	}
}
