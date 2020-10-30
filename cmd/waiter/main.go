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

	// master server
	ms *mserver.Client

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

	configuredBans, err := bans.FromFile("bans.json")
	if err != nil {
		log.Fatalln(err)
	}
	bm = bans.New(configuredBans...)

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

	s.Empty()
	s.Unsupervised()

	is, infoInc = StartListeningForInfoRequests(s)

	providers[conf.AuthDomain] = auth.NewInMemoryProvider(users)

	// regular master server
	var (
		authInc <-chan string
		authOut chan<- string
		bansInc <-chan string
	)
	ms, authInc, authOut, bansInc = mserver.New(
		conf.MasterServerAddress,
		func(c *mserver.Client) { c.Register(conf.ListenPort) },
		func(c *mserver.Client) { s.ReAuthClients("") },
	)
	providers[""] = auth.NewRemoteProvider(authInc, authOut, role.Auth)
	go bm.Handle(suffixed(bansInc, "sauerbraten.org"))
	ms.Start()

	// stats auth master server
	s.StatsServer, authInc, authOut, bansInc = mserver.New(
		conf.StatsServerAddress,
		func(c *mserver.Client) {
			c.Register(conf.ListenPort)

			mserver.ExtendWithStats(
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
				mserver.NewAdmin(s.StatsServer, adminName, adminKey,
					func(ac *mserver.Admin) {
						s.StatsServerAdmin = ac
						s.Commands.Register(server.RegisterPubkey)
					},
					func() { s.Commands.Unregister(server.RegisterPubkey) },
				)
			}
		},
		func(*mserver.Client) { s.ReAuthClients(conf.StatsServerAuthDomain) },
	)
	providers[conf.StatsServerAuthDomain] = auth.NewRemoteProvider(authInc, authOut, role.None)
	go bm.Handle(suffixed(bansInc, conf.StatsServerAuthDomain))
	s.StatsServer.Start()

	s.AuthManager = auth.NewManager(providers)

	log.Println("server running on port", s.Config.ListenPort)

	// don't put these inside the for loop below!
	enetInc := host.Service()
	reRegisterTicker := time.Tick(1 * time.Hour)

	for {
		select {
		case event := <-enetInc:
			handleEnetEvent(event)
		case req := <-infoInc:
			is.Handle(req)
		case msg := <-ms.Incoming():
			go ms.Handle(msg)
		case msg := <-s.StatsServer.Incoming():
			go s.StatsServer.Handle(msg)
		case <-reRegisterTicker:
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

func suffixed(inc <-chan string, suffix string) <-chan string {
	out := make(chan string)

	go func() {
		for msg := range inc {
			out <- msg + " " + suffix
		}
	}()

	return out
}
