package main

import (
	"log"
	"math/rand"
	"time"

	"github.com/sauerbraten/jsonfile"
	"github.com/sauerbraten/maitred/pkg/auth"
	mserver "github.com/sauerbraten/maitred/pkg/client"

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
	ms        *mserver.VanillaClient
	masterInc <-chan string

	// stats server
	statsAuth    *mserver.AdminClient
	statsAuthInc <-chan string

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
	providers[conf.AuthDomain] = auth.NewInMemoryProvider(users)

	ms, masterInc, err = mserver.NewVanilla(conf.MasterServerAddress, conf.ListenPort, bm, role.Auth, func() { s.ReAuth("") })
	if err != nil {
		log.Println("could not connect to master server:", err)
	}
	if ms != nil && ms.RemoteProvider != nil {
		providers[""] = ms.RemoteProvider
	}

	var _statsAuth *mserver.StatsClient
	_statsAuth, statsAuthInc, err = mserver.NewStats(
		conf.StatsServerAddress,
		conf.ListenPort,
		func(reqID uint32) { s.HandleSuccStats(reqID) },
		func(reqID uint32, reason string) { s.HandleFailStats(reqID, reason) },
		func() { s.ReAuth(s.StatsServerAuthDomain) },
	)
	if err != nil {
		log.Println("could not connect to statsauth server:", err)
	}
	if _statsAuth != nil {
		statsAuth = mserver.NewAdmin(_statsAuth)
		providers[conf.StatsServerAuthDomain] = statsAuth
	}

	host, err := enet.NewHost(conf.ListenAddress, conf.ListenPort)
	if err != nil {
		log.Fatalln(err)
	}

	s, callbacks = server.New(host, conf, auth.NewManager(providers), bm, statsAuth,
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

	gameInc := host.Service()

	log.Println("server running on port", s.Config.ListenPort)

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
