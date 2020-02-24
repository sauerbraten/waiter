package client

import (
	"fmt"
	"strings"

	"github.com/sauerbraten/maitred/v2/pkg/auth"
	"github.com/sauerbraten/maitred/v2/pkg/protocol"
)

type AdminClient struct {
	Client
	name      string
	privKey   auth.PrivateKey
	ids       *protocol.IDCycle
	callbacks map[uint32]func(string)
}

func NewAdmin(c Client, name string, privKey auth.PrivateKey) *AdminClient {
	if ac, ok := c.(*AdminClient); ok {
		return ac
	}

	return &AdminClient{
		Client:    c,
		name:      name,
		privKey:   privKey,
		ids:       new(protocol.IDCycle),
		callbacks: map[uint32]func(string){},
	}
}

func (c *AdminClient) Upgrade(onSuccess, onFailure func()) {
	c.Logf("trying to upgrade to admin connection")
	reqID := c.ids.Next()
	c.callbacks[reqID] = func(reason string) {
		if reason == "" {
			c.Logf("successfully upgraded connection to admin connection")
			if onSuccess != nil {
				onSuccess()
			}
			return
		}
		c.Logf("upgrading connection to admin connection failed: %s", reason)
		if onFailure != nil {
			onFailure()
		}
	}
	c.Client.Send("%s %d %s", protocol.ReqAdmin, reqID, c.name)
}

func (c *AdminClient) AddAuth(name, pubkey string, callback func(string)) {
	reqID := c.ids.Next()
	c.callbacks[reqID] = callback
	c.Send("%s %d %s %s", protocol.AddAuth, reqID, name, pubkey)
}

func (c *AdminClient) DelAuth(name string, callback func(string)) {
	reqID := c.ids.Next()
	c.callbacks[reqID] = callback
	c.Send("%s %d %s", protocol.DelAuth, reqID, name)
}

func (c *AdminClient) Handle(msg string) {
	cmd := strings.Split(msg, " ")[0]
	args := strings.TrimSpace(msg[len(cmd):])

	switch cmd {
	case protocol.ChalAdmin:
		c.handleChalAdmin(args)

	case protocol.SuccAdmin:
		c.handleSuccAdmin(args)

	case protocol.FailAdmin:
		c.handleFailAdmin(args)

	case protocol.SuccAddAuth:
		c.handleSuccAddAuth(args)

	case protocol.FailAddAuth:
		c.handleFailAddAuth(args)

	case protocol.SuccDelAuth:
		c.handleSuccDelAuth(args)

	case protocol.FailDelAuth:
		c.handleFailDelAuth(args)

	default:
		c.Client.Handle(msg)
	}
}

func (c *AdminClient) handleChalAdmin(args string) {
	var reqID uint32
	var challenge string
	_, err := fmt.Sscanf(args, "%d %s", &reqID, &challenge)
	if err != nil {
		c.Logf("malformed %s message from stats server: '%s': %v", protocol.ChalAdmin, args, err)
		return
	}

	answer, err := auth.Solve(challenge, c.privKey)
	if err != nil {
		c.Logf("could not solve admin challenge: %v", err)
		return
	}

	c.Client.Send("%s %d %s", protocol.ConfAdmin, reqID, answer)
}

func (c *AdminClient) handleSuccAdmin(args string) {
	var reqID uint32
	_, err := fmt.Sscanf(args, "%d", &reqID)
	if err != nil {
		c.Logf("malformed %s message from stats server: '%s': %v", protocol.SuccAdmin, args, err)
		return
	}

	if callback, ok := c.callbacks[reqID]; ok {
		callback("")
	}
}

func (c *AdminClient) handleFailAdmin(args string) {
	r := strings.NewReader(strings.TrimSpace(args))

	var reqID uint32
	_, err := fmt.Fscanf(r, "%d", &reqID)
	if err != nil {
		c.Logf("malformed %s message from stats server: '%s': %v", protocol.FailAdmin, args, err)
		return
	}
	reason := args[len(args)-r.Len():] // unread portion of args
	reason = strings.TrimSpace(reason)

	if callback, ok := c.callbacks[reqID]; ok {
		callback(reason)
	}
}

func (c *AdminClient) handleSuccAddAuth(args string) {
	var reqID uint32
	_, err := fmt.Sscanf(args, "%d", &reqID)
	if err != nil {
		c.Logf("malformed %s message from stats server: '%s': %v", protocol.SuccAddAuth, args, err)
		return
	}

	if callback, ok := c.callbacks[reqID]; ok {
		callback("")
	}
}

func (c *AdminClient) handleFailAddAuth(args string) {
	r := strings.NewReader(strings.TrimSpace(args))

	var reqID uint32
	_, err := fmt.Fscanf(r, "%d", &reqID)
	if err != nil {
		c.Logf("malformed %s message from stats server: '%s': %v", protocol.FailAddAuth, args, err)
		return
	}
	reason := args[len(args)-r.Len():] // unread portion of args
	reason = strings.TrimSpace(reason)

	if callback, ok := c.callbacks[reqID]; ok {
		callback(reason)
	}
}

func (c *AdminClient) handleSuccDelAuth(args string) {
	var reqID uint32
	_, err := fmt.Sscanf(args, "%d", &reqID)
	if err != nil {
		c.Logf("malformed %s message from stats server: '%s': %v", protocol.SuccDelAuth, args, err)
		return
	}

	if callback, ok := c.callbacks[reqID]; ok {
		callback("")
	}
}

func (c *AdminClient) handleFailDelAuth(args string) {
	r := strings.NewReader(strings.TrimSpace(args))

	var reqID uint32
	_, err := fmt.Fscanf(r, "%d", &reqID)
	if err != nil {
		c.Logf("malformed %s message from stats server: '%s': %v", protocol.FailDelAuth, args, err)
		return
	}
	reason := args[len(args)-r.Len():] // unread portion of args
	reason = strings.TrimSpace(reason)

	if callback, ok := c.callbacks[reqID]; ok {
		callback(reason)
	}
}
