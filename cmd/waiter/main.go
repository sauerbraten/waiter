package main

import (
	"log"
	"time"

	"github.com/sauerbraten/jsonfile"

	"github.com/sauerbraten/waiter/internal/masterserver"
	"github.com/sauerbraten/waiter/internal/net/enet"
	"github.com/sauerbraten/waiter/pkg/auth"
	"github.com/sauerbraten/waiter/pkg/bans"
	"github.com/sauerbraten/waiter/pkg/definitions/disconnectreason"
	"github.com/sauerbraten/waiter/pkg/definitions/role"
	"github.com/sauerbraten/waiter/pkg/protocol"
)

var (
	// global server state
	s *Server

	// ban manager
	bm *bans.BanManager

	localAuth auth.Provider

	// master server
	ms        *masterserver.MasterServer
	masterInc <-chan string

	// stats server
	statsAuth    *masterserver.MasterServer
	statsAuthInc <-chan string

	// info server
	is      *infoServer
	infoInc <-chan infoRequest
)

func init() {
	var conf *Config
	err := jsonfile.ParseFile("config.json", &conf)
	if err != nil {
		log.Fatalln(err)
	}

	var mr MapRotation
	err = jsonfile.ParseFile("maps.json", &mr)
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
	localAuth = auth.NewInMemoryProvider(users)

	cs := &ClientManager{}

	s = &Server{
		Config: conf,
		State: &State{
			UpSince:    time.Now(),
			NumClients: cs.NumberOfClientsConnected,
		},
		relay:       NewRelay(),
		Clients:     cs,
		MapRotation: &mr,
	}
	s.GameDuration = s.GameDuration * time.Minute // duration is parsed without unit from config file
	s.GameMode = NewGame(conf.FallbackGameMode)
	s.Map = s.MapRotation.NextMap(s.GameMode, "")
	s.GameMode.Start()
	s.Unsupervised()
	s.Empty()

	is, infoInc = s.StartListeningForInfoRequests()

	ms, masterInc, err = masterserver.New(conf.MasterServerAddress, conf.ListenPort, bm, role.Auth)
	if err != nil {
		log.Println("could not connect to master server:", err)
	}

	statsAuth, statsAuthInc, err = masterserver.New(conf.StatsServerAddress, conf.ListenPort, bm, role.None)
	if err != nil {
		log.Println("could not connect to statsauth server:", err)
	}

	s.AuthManager = auth.NewManager(map[string]auth.Provider{
		"":                         ms.RemoteAuthProvider,
		conf.PrimaryAuthDomain:     localAuth,
		conf.StatsServerAuthDomain: statsAuth,
	})
}

func main() {
	host, err := enet.NewHost(s.Config.ListenAddress, s.Config.ListenPort)
	if err != nil {
		log.Fatalln(err)
	}

	log.Println("server running on port", s.Config.ListenPort)

	go s.relay.loop()

	gameInc := host.Service()

	for {
		select {
		case event := <-gameInc:
			handleEnetEvent(event)
		case req := <-infoInc:
			is.Handle(req)
		case msg := <-masterInc:
			go ms.Handle(msg)
		case msg := <-statsAuthInc:
			go statsAuth.Handle(msg)
		case <-time.Tick(1 * time.Hour):
			go ms.Register()
			go statsAuth.Register()
		}
	}
}

func handleEnetEvent(event enet.Event) {
	switch event.Type {
	case enet.EVENT_TYPE_CONNECT:
		s.Connect(event.Peer)

	case enet.EVENT_TYPE_DISCONNECT:
		client := s.Clients.GetClientByPeer(event.Peer)
		if client == nil {
			return
		}
		s.Disconnect(client, disconnectreason.None)

	case enet.EVENT_TYPE_RECEIVE:
		client := s.Clients.GetClientByPeer(event.Peer)
		if client == nil {
			return
		}
		s.handlePacket(client, event.ChannelID, protocol.Packet(event.Packet.Data))
	}
}
