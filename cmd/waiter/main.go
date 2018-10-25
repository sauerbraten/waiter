package main

import (
	"log"
	"strconv"
	"time"

	"github.com/sauerbraten/jsonfile"

	"github.com/sauerbraten/waiter/protocol"

	"github.com/sauerbraten/waiter/internal/auth"
	"github.com/sauerbraten/waiter/internal/bans"
	"github.com/sauerbraten/waiter/internal/definitions/disconnectreason"
	"github.com/sauerbraten/waiter/internal/definitions/mastermode"
	"github.com/sauerbraten/waiter/internal/maprotation"
	"github.com/sauerbraten/waiter/internal/masterserver"
	"github.com/sauerbraten/waiter/internal/protocol/enet"
)

var (
	// global enet host var (to call Flush() on)
	host *enet.Host

	// global variable to indicate to the main loop that there are packets to be sent
	mustFlush = false

	// global server state
	s *Server

	// client manager
	cs *ClientManager

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

	cs = &ClientManager{}

	s = &Server{
		Config: conf,
		State: &State{
			MasterMode:  mastermode.Open,
			GameMode:    GameModeByID(conf.FallbackGameMode),
			Map:         maprotation.NextMap(conf.FallbackGameMode, ""),
			NotGotItems: false,
			UpSince:     time.Now(),
			NumClients:  cs.NumberOfClientsConnected,
		},
		GameTimer: NewGameTimer(conf.GameDuration*time.Minute, func() { s.Intermission() }),
		relay:     NewRelay(),
		Clients:   cs,
		Auth:      auth.NewManager(users),
	}

	ms, err = masterserver.New(s.Config.MasterServerAddress+":"+strconv.Itoa(s.Config.MasterServerPort), bm)
	if err != nil {
		log.Println("could not connect to master server:", err)
	}
}

func main() {
	var err error
	host, err = enet.NewHost(s.Config.ListenAddress, s.Config.ListenPort)
	if err != nil {
		log.Fatalln(err)
	}

	log.Println("server running on port", s.Config.ListenPort)

	extInfoServer := &ExtInfoServer{
		Config:    s.Config,
		State:     s.State,
		GameTimer: s.GameTimer,
		Clients:   s.Clients,
	}
	go extInfoServer.ServeStateInfoForever()

	go s.GameTimer.run()

	go s.relay.loop()

	if ms != nil {
		log.Println("registering at master server")
		err = ms.Register(s.Config.ListenPort)
		if err != nil {
			log.Println(err)
		}
	}

	for {
		event := host.Service()

		switch event.Type {
		case enet.EVENT_TYPE_CONNECT:
			log.Println("enet: connected:", event.Peer.Address.String())

			if ban, ok := bm.GetBan(event.Peer.Address.IP); ok {
				log.Println("peer's IP is banned:", ban)
				event.Peer.Disconnect(uint32(disconnectreason.Banned))
				continue
			}

			client := cs.Add(event.Peer)

			client.Position, client.Packets = s.relay.AddClient(client.CN, client.Peer.Send)

			cs.SendServerConfig(client, s.Config)

		case enet.EVENT_TYPE_DISCONNECT:
			log.Println("enet: disconnected:", event.Peer.Address.String())
			client := s.Clients.GetClientByPeer(event.Peer)
			if client == nil {
				continue
			}
			s.relay.RemoveClient(client.CN)
			cs.Leave(client)

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
