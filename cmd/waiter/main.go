package main

import (
	"log"
	"math/rand"
	"time"

	"github.com/sauerbraten/jsonfile"
	"github.com/sauerbraten/maitred/v2/pkg/auth"
	mserver "github.com/sauerbraten/maitred/v2/pkg/client"

	"github.com/sauerbraten/waiter/pkg/bans"
	"github.com/sauerbraten/waiter/pkg/enet"
	"github.com/sauerbraten/waiter/pkg/protocol"
	"github.com/sauerbraten/waiter/pkg/protocol/disconnectreason"
	"github.com/sauerbraten/waiter/pkg/protocol/role"
	"github.com/sauerbraten/waiter/pkg/server"
)

var rng = rand.New(rand.NewSource(time.Now().UnixNano()))

var (
	// global server state
	s *server.Server

	// ban manager
	bm *bans.BanManager

	localAuth auth.Provider

	// master server
	ms *mserver.VanillaClient

	// stats server
	statsAuth *mserver.StatsClient

	// info server
	is      *infoServer
	infoInc <-chan infoRequest

	// callbacks (e.g. IP geolocation queries)
	callbacks <-chan func()

	// auth manager
	providers   = map[string]auth.Provider{}
	authManager *auth.Manager
)

func main() {
	var conf *server.Config
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

	host, err := enet.NewHost(conf.ListenAddress, conf.ListenPort)
	if err != nil {
		log.Fatalln(err)
	}

	s, callbacks = server.New(host, conf, bm,
		server.QueueMap,
		server.ToggleKeepTeams,
		server.ToggleCompetitiveMode,
		server.ToggleReportStats,
		server.LookupIPs,
		server.SetTimeLeft,
		server.RegisterPubkey,
	)

	s.GameMode = s.StartMode(conf.FallbackGameMode)
	s.Map = s.MapRotation.NextMap(conf.FallbackGameMode, conf.FallbackGameMode, "")
	s.GameMode.Start()
	s.Unsupervised()
	s.Empty()

	is, infoInc = StartListeningForInfoRequests(s)

	providers[conf.AuthDomain] = auth.NewInMemoryProvider(users)

	ms, err = mserver.NewVanilla(conf.MasterServerAddress, conf.ListenPort, bm, role.Auth, func() { s.ReAuth("") })
	if err != nil {
		log.Println("could not connect to master server:", err)
	}
	if ms != nil {
		go ms.Start()
		providers[""] = ms.RemoteProvider
	}

	statsAuth, err = mserver.NewStats(
		conf.StatsServerAddress,
		conf.ListenPort,
		func(reqID uint32) { s.HandleSuccStats(reqID) },
		func(reqID uint32, reason string) { s.HandleFailStats(reqID, reason) },
		func() { s.ReAuth(s.StatsServerAuthDomain) },
	)
	if err != nil {
		log.Println("could not connect to statsauth server:", err)
	}
	if statsAuth != nil {
		go statsAuth.Start()
		s.StatsServer, err = mserver.NewAdmin(statsAuth)
		if err != nil {
			log.Println("could not create stats auth admin client:", err)
		} else {
			providers[conf.StatsServerAuthDomain] = s.StatsServer
		}
	}
	s.AuthManager = auth.NewManager(providers)

	gameInc := host.Service()

	log.Println("server running on port", s.Config.ListenPort)

	for {
		select {
		case event := <-gameInc:
			handleEnetEvent(event)
		case req := <-infoInc:
			is.Handle(req)
		case msg := <-ms.Incoming():
			go ms.Handle(msg)
		case msg := <-statsAuth.Incoming():
			go s.StatsServer.Handle(msg)
		case <-time.Tick(1 * time.Hour):
			go ms.Register()
			go s.StatsServer.Register()
		case f := <-callbacks:
			f()
		}
	}
}

func handleEnetEvent(event enet.Event) {
	switch event.Type {
	case enet.EventTypeConnect:
		s.Connect(event.Peer)

	case enet.EventTypeDisconnect:
		client := s.Clients.GetClientByPeer(event.Peer)
		if client == nil {
			return
		}
		s.Disconnect(client, disconnectreason.None)

	case enet.EventTypeReceive:
		client := s.Clients.GetClientByPeer(event.Peer)
		if client == nil {
			return
		}
		s.HandlePacket(client, event.ChannelID, protocol.Packet(event.Packet.Data))
	}
}
