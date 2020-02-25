package client

import (
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	"github.com/sauerbraten/maitred/v2/pkg/protocol"
)

type VanillaClient struct {
	*conn
	pingFailed bool

	authInc chan<- string
	authOut <-chan string

	bansInc chan<- string

	onReconnect func(*VanillaClient) // executed when the game server reconnects to the remote master server
}

// New connects to the specified master server. Bans received from the master server are added to the given ban manager.
func NewVanilla(addr string, onConnect, onReconnect func(*VanillaClient)) (c *VanillaClient, authInc <-chan string, authOut chan<- string, bansInc <-chan string) {
	if onConnect == nil {
		onConnect = func(*VanillaClient) {}
	}
	if onReconnect == nil {
		onReconnect = func(*VanillaClient) {}
	}

	_authInc, _authOut, _bansInc := make(chan string), make(chan string), make(chan string)

	c = &VanillaClient{
		authInc: _authInc,
		authOut: _authOut,

		bansInc: _bansInc,

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

	return c, _authInc, _authOut, _bansInc
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

	case protocol.ClearBans, protocol.AddBan:
		c.bansInc <- msg

	case protocol.ChalAuth, protocol.SuccAuth, protocol.FailAuth:
		c.authInc <- msg

	default:
		c.Logf("unhandled message: %v", msg)
	}
}
