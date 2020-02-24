package client

import (
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	"github.com/sauerbraten/chef/pkg/ips"
	"github.com/sauerbraten/waiter/pkg/bans"

	"github.com/sauerbraten/maitred/v2/pkg/protocol"
)

type VanillaClient struct {
	bans *bans.BanManager

	*conn
	pingFailed bool

	authInc chan<- string
	authOut <-chan string

	onReconnect func(*VanillaClient) // executed when the game server reconnects to the remote master server
}

// New connects to the specified master server. Bans received from the master server are added to the given ban manager.
func NewVanilla(addr string, bans *bans.BanManager, onConnect, onReconnect func(*VanillaClient)) (c *VanillaClient, authInc <-chan string, authOut chan<- string) {
	if onConnect == nil {
		onConnect = func(*VanillaClient) {}
	}
	if onReconnect == nil {
		onReconnect = func(*VanillaClient) {}
	}

	_authInc, _authOut := make(chan string), make(chan string)

	c = &VanillaClient{
		bans: bans,

		authInc: _authInc,
		authOut: _authOut,

		onReconnect: onReconnect,
	}

	_onConnect := func() {
		go func() {
			for msg := range c.authOut {
				c.conn.Send(msg)
			}
		}()

		onConnect(c)
	}

	onDisconnect := func(err error) {
		c.Logf("disconnected: %v", err)
		if !c.pingFailed {
			c.reconnect(err)
		}
	}

	c.conn = newConn(addr, _onConnect, onDisconnect)

	return c, _authInc, _authOut
}

func (c *VanillaClient) Start() {
	err := c.conn.connect()
	if err != nil {
		c.Logf("error connecting to %s: %v", c.addr, err)
		if netErr, ok := err.(*net.OpError); ok && !netErr.Temporary() {
			c.Logf("not a temporary error, not retrying")
		} else {
			c.reconnect(err)
		}
	}
}

func (c *VanillaClient) reconnect(err error) {
	try, maxTries := 1, 10
	for err != nil && try <= maxTries {
		time.Sleep(time.Duration(try) * 30 * time.Second)
		c.Logf("trying to reconnect (attempt %d)", try)

		err = c.conn.connect()
		if err != nil {
			c.Logf("failed to reconnect (attempt %d)", try)
		}

		try++
	}

	if err == nil {
		c.onReconnect(c)
	} else {
		c.Logf("could not reconnect: %v", err)
	}
}

func (c *VanillaClient) Logf(format string, args ...interface{}) {
	log.Println(fmt.Sprintf("master (%s):", c.conn.addr), fmt.Sprintf(format, args...))
}

func (c *VanillaClient) Register(listenPort int) {
	if c.pingFailed {
		return
	}
	c.Logf("registering")
	c.Send("%s %d", protocol.RegServ, listenPort)
}

func (c *VanillaClient) Handle(msg string) {
	cmd := strings.Split(msg, " ")[0]
	args := strings.TrimSpace(msg[len(cmd):])

	switch cmd {
	case protocol.SuccReg:
		c.Logf("registration succeeded")

	case protocol.FailReg:
		c.Logf("registration failed: %v", args)
		if args == "failed pinging server" {
			c.Logf("disabling reconnecting")
			c.pingFailed = true // stop trying
		}

	case protocol.ClearBans:
		if c.bans != nil {
			c.bans.ClearGlobalBans()
		}

	case protocol.AddBan:
		if c.bans != nil {
			c.handleAddGlobalBan(args)
		}

	case protocol.ChalAuth, protocol.SuccAuth, protocol.FailAuth:
		c.authInc <- msg

	default:
		c.Logf("received and not handled: %v", msg)
	}
}

func (c *VanillaClient) handleAddGlobalBan(args string) {
	var ip string
	_, err := fmt.Sscanf(args, "%s", &ip)
	if err != nil {
		c.Logf("malformed %s message from game server: '%s': %v", protocol.AddBan, args, err)
		return
	}

	network := ips.GetSubnet(ip)

	c.bans.AddBan(network, fmt.Sprintf("banned by master server (%s)", c.conn.addr), time.Time{}, true)
}
