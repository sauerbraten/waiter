package main

import (
	"strconv"
	"strings"

	"github.com/sauerbraten/waiter/internal/definitions/nmc"
	"github.com/sauerbraten/waiter/internal/definitions/role"
	"github.com/sauerbraten/waiter/pkg/protocol/cubecode"
)

func (s *Server) HandleCommand(c *Client, msg string) {
	msg = strings.TrimSpace(msg)
	parts := strings.Split(msg, " ")
	cmd := parts[0]

	switch cmd {
	case "help":
		if c.Role > role.None {
			c.Send(nmc.ServerMessage, "available commands: keepteams (=persist)")
		}
	case "persist", "keepteams":
		if c.Role == role.None {
			return
		}
		if len(parts) > 1 {
			val, err := strconv.Atoi(parts[1])
			if err != nil || (val != 0 && val != 1) {
				return
			}
			s.KeepTeams = val == 1
		}
		if s.KeepTeams {
			c.Send(nmc.ServerMessage, "teams will be kept")
		} else {
			c.Send(nmc.ServerMessage, "teams will be shuffled")
		}

	default:
		c.Send(nmc.ServerMessage, cubecode.Fail("unknown command"))
	}
}
