package client

import (
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	"github.com/sauerbraten/chef/pkg/ips"
	"github.com/sauerbraten/waiter/pkg/bans"
	"github.com/sauerbraten/waiter/pkg/protocol/role"

	"github.com/sauerbraten/maitred/v2/pkg/auth"
	"github.com/sauerbraten/maitred/v2/pkg/protocol"
)

type VanillaClient struct {
	listenPort int
	bans       *bans.BanManager

	*conn
	pingFailed bool

	*auth.RemoteProvider
	authInc chan<- string
	authOut <-chan string

	onReconnect func() // executed when the game server reconnects to the remote master server
}

// New connects to the specified master server. Bans received from the master server are added to the given ban manager.
func NewVanilla(addr string, listenPort int, bans *bans.BanManager, authRole role.ID, onReconnect func()) (*VanillaClient, error) {
	if onReconnect == nil {
		onReconnect = func() {}
	}

	authInc, authOut := make(chan string), make(chan string)

	c := &VanillaClient{
		listenPort: listenPort,
		bans:       bans,

		RemoteProvider: auth.NewRemoteProvider(authInc, authOut, authRole),
		authInc:        authInc,
		authOut:        authOut,

		onReconnect: onReconnect,
	}

	raddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("master (%s): error resolving server address (%s): %v", addr, addr, err)
	}

	onConnect := func() {
		go func() {
			for msg := range c.authOut {
				c.conn.Send(msg)
			}
		}()

		c.Register()
	}

	onDisconnect := func(err error) {
		c.Log("disconnected: %v", err)
		if !c.pingFailed {
			c.reconnect(err)
		}
	}

	c.conn = newConn(raddr, onConnect, onDisconnect)

	return c, nil
}

func (c *VanillaClient) Start() {
	err := c.conn.connect()
	if err != nil {
		c.Log("error connecting to %s: %v", c.conn.addr, err)
		c.reconnect(err)
	}
}

func (c *VanillaClient) reconnect(err error) {
	try, maxTries := 1, 10
	for err != nil && try <= maxTries {
		time.Sleep(time.Duration(try) * 30 * time.Second)
		c.Log("trying to reconnect (attempt %d)", try)

		err = c.conn.connect()
		if err != nil {
			c.Log("failed to reconnect (attempt %d)", try)
		}

		try++
	}

	if err == nil {
		c.onReconnect()
	} else {
		c.Log("could not reconnect: %v", err)
	}
}

func (c *VanillaClient) Log(format string, args ...interface{}) {
	log.Println(fmt.Sprintf("master (%s):", c.conn.addr), fmt.Sprintf(format, args...))
}

func (c *VanillaClient) Register() {
	if c.pingFailed {
		return
	}
	c.Log("registering")
	c.Send("%s %d", protocol.RegServ, c.listenPort)
}

func (c *VanillaClient) Handle(msg string) {
	cmd := strings.Split(msg, " ")[0]
	args := strings.TrimSpace(msg[len(cmd):])

	switch cmd {
	case protocol.SuccReg:
		c.Log("registration succeeded")

	case protocol.FailReg:
		c.Log("registration failed: %v", args)
		if args == "failed pinging server" {
			c.Log("disabling reconnecting")
			c.pingFailed = true // stop trying
		}

	case protocol.ClearBans:
		c.bans.ClearGlobalBans()

	case protocol.AddBan:
		c.handleAddGlobalBan(args)

	case protocol.ChalAuth, protocol.SuccAuth, protocol.FailAuth:
		c.authInc <- msg

	default:
		c.Log("received and not handled: %v", msg)
	}
}

func (c *VanillaClient) handleAddGlobalBan(args string) {
	var ip string
	_, err := fmt.Sscanf(args, "%s", &ip)
	if err != nil {
		c.Log("malformed %s message from game server: '%s': %v", protocol.AddBan, args, err)
		return
	}

	network := ips.GetSubnet(ip)

	c.bans.AddBan(network, fmt.Sprintf("banned by master server (%s)", c.conn.addr), time.Time{}, true)
}
