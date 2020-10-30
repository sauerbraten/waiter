package client

import (
	"fmt"
	"strings"

	"github.com/sauerbraten/maitred/v2/pkg/auth"
	"github.com/sauerbraten/maitred/v2/pkg/protocol"
)

type Admin struct {
	*Client
	name      string
	privKey   auth.PrivateKey
	ids       *protocol.IDCycle
	callbacks map[uint32]func(string)
}

func NewAdmin(c *Client, name string, privKey auth.PrivateKey, onSuccess func(*Admin), onFailure func()) *Admin {
	ac := &Admin{
		Client:    c,
		name:      name,
		privKey:   privKey,
		ids:       new(protocol.IDCycle),
		callbacks: map[uint32]func(string){},
	}

	extensions := map[string]func(string){
		protocol.ChalAdmin:   ac.handleChalAdmin,
		protocol.SuccAdmin:   ac.handleSuccAdmin,
		protocol.FailAdmin:   ac.handleFailAdmin,
		protocol.SuccAddAuth: ac.handleSuccAddAuth,
		protocol.FailAddAuth: ac.handleFailAddAuth,
		protocol.SuccDelAuth: ac.handleSuccDelAuth,
		protocol.FailDelAuth: ac.handleFailDelAuth,
	}

	for cmd, handler := range extensions {
		c.RegisterExtension(cmd, handler)
	}

	ac.Logf("trying to upgrade to admin connection")
	reqID := ac.ids.Next()
	ac.callbacks[reqID] = func(reason string) {
		if reason == "" {
			ac.Logf("successfully upgraded connection to admin connection")
			if onSuccess != nil {
				onSuccess(ac)
			}
			return
		}
		ac.Logf("upgrading connection to admin connection failed: %s", reason)
		for cmd := range extensions {
			c.UnregisterExtension(cmd)
		}
		if onFailure != nil {
			onFailure()
		}
	}
	ac.Client.Send("%s %d %s", protocol.ReqAdmin, reqID, ac.name)

	return ac
}

func (c *Admin) AddAuth(name, pubkey string, callback func(string)) {
	reqID := c.ids.Next()
	c.callbacks[reqID] = callback
	c.Send("%s %d %s %s", protocol.AddAuth, reqID, name, pubkey)
}

func (c *Admin) DelAuth(name string, callback func(string)) {
	reqID := c.ids.Next()
	c.callbacks[reqID] = callback
	c.Send("%s %d %s", protocol.DelAuth, reqID, name)
}

func (c *Admin) handleChalAdmin(args string) {
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

func (c *Admin) handleSuccAdmin(args string) {
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

func (c *Admin) handleFailAdmin(args string) {
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

func (c *Admin) handleSuccAddAuth(args string) {
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

func (c *Admin) handleFailAddAuth(args string) {
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

func (c *Admin) handleSuccDelAuth(args string) {
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

func (c *Admin) handleFailDelAuth(args string) {
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
