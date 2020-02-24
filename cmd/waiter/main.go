package main

import (
	"log"
	"math/rand"
	"os"
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
	)

	s.GameMode = s.StartMode(conf.FallbackGameMode)
	s.Map = s.MapRotation.NextMap(conf.FallbackGameMode, conf.FallbackGameMode, "")
	s.GameMode.Start()
	s.Unsupervised()
	s.Empty()

	is, infoInc = StartListeningForInfoRequests(s)

	providers[conf.AuthDomain] = auth.NewInMemoryProvider(users)

	// regular master server
	var (
		authInc <-chan string
		authOut chan<- string
	)
	ms, authInc, authOut = mserver.NewVanilla(
		conf.MasterServerAddress,
		bm,
		func(c *mserver.VanillaClient) { c.Register(conf.ListenPort) },
		func(c *mserver.VanillaClient) { s.ReAuthClients("") },
	)
	providers[""] = auth.NewRemoteProvider(authInc, authOut, role.None)
	ms.Start()

	// stats auth master server
	statsMS, authInc, authOut := mserver.NewVanilla(
		conf.StatsServerAddress,
		nil,
		func(c *mserver.VanillaClient) {
			c.Register(conf.ListenPort)

			s.StatsServer = mserver.NewStats(
				c,
				func(reqID uint32) { s.HandleSuccStats(reqID) },
				func(reqID uint32, reason string) { s.HandleFailStats(reqID, reason) },
			)

			adminName, _adminKey := os.Getenv("STATSAUTH_ADMIN_NAME"), os.Getenv("STATSAUTH_ADMIN_KEY")
			if adminName != "" && _adminKey != "" {
				adminKey, err := auth.ParsePrivateKey(_adminKey)
				if err != nil {
					log.Fatalln(err)
				}
				s.StatsServer = mserver.NewAdmin(s.StatsServer, adminName, adminKey)
				s.StatsServer.(*mserver.AdminClient).Upgrade(
					func() { s.Commands.Register(server.RegisterPubkey) },
					func() { s.Commands.Unregister(server.RegisterPubkey) },
				)
			}
		},
		func(*mserver.VanillaClient) { s.ReAuthClients(conf.StatsServerAuthDomain) },
	)
	providers[conf.StatsServerAuthDomain] = auth.NewRemoteProvider(authInc, authOut, role.None)
	statsMS.Start()

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
		case msg := <-s.StatsServer.Incoming():
			go s.StatsServer.Handle(msg)
		case <-time.Tick(1 * time.Hour):
			go ms.Register(conf.ListenPort)
			go s.StatsServer.Register(conf.ListenPort)
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
