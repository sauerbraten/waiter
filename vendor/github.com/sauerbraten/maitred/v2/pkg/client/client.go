package client

import (
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/sauerbraten/maitred/v2/pkg/protocol"
)

type Client struct {
	*conn
	pingFailed bool

	authInc chan<- string
	authOut <-chan string

	bansInc chan<- string

	onReconnect func(*Client) // executed when the game server reconnects to the remote master server

	extLock    sync.RWMutex
	extensions map[string]func(args string)
}

func New(addr string, onConnect, onReconnect func(*Client)) (c *Client, authInc <-chan string, authOut chan<- string, bansInc <-chan string) {
	if onConnect == nil {
		onConnect = func(*Client) {}
	}
	if onReconnect == nil {
		onReconnect = func(*Client) {}
	}

	_authInc, _authOut, _bansInc := make(chan string), make(chan string), make(chan string)

	c = &Client{
		authInc: _authInc,
		authOut: _authOut,

		bansInc: _bansInc,

		onReconnect: onReconnect,

		extensions: map[string]func(string){},
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

func (c *Client) Start() {
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

func (c *Client) reconnect(err error) {
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

func (c *Client) Logf(format string, args ...interface{}) {
	log.Println(fmt.Sprintf("master (%s):", c.conn.addr), fmt.Sprintf(format, args...))
}

func (c *Client) Register(listenPort int) {
	if c.pingFailed {
		return
	}
	c.Logf("registering")
	c.Send("%s %d", protocol.RegServ, listenPort)
}

func (c *Client) Handle(msg string) {
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
		c.extLock.RLock()
		defer c.extLock.RUnlock()
		if handler, ok := c.extensions[cmd]; ok {
			handler(args)
		} else {
			c.Logf("unhandled message: %v", msg)
		}
	}
}

func (c *Client) RegisterExtension(cmd string, handler func(args string)) {
	c.extLock.Lock()
	defer c.extLock.Unlock()
	c.extensions[cmd] = handler
}

func (c *Client) UnregisterExtension(cmd string) {
	c.extLock.Lock()
	defer c.extLock.Unlock()
	delete(c.extensions, cmd)
}

func (c *Client) HasExtension(cmd string) bool {
	c.extLock.RLock()
	defer c.extLock.RUnlock()
	_, ok := c.extensions[cmd]
	return ok
}
