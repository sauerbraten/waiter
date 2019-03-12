package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/sauerbraten/waiter/internal/game"
	"github.com/sauerbraten/waiter/pkg/definitions/mastermode"
	"github.com/sauerbraten/waiter/pkg/definitions/nmc"
	"github.com/sauerbraten/waiter/pkg/definitions/role"
	"github.com/sauerbraten/waiter/pkg/protocol/cubecode"
)

func (s *Server) HandleCommand(c *Client, msg string) {
	msg = strings.TrimSpace(msg)
	parts := strings.Split(msg, " ")
	cmd := parts[0]

	switch cmd {
	case "help", "commands":
		switch c.Role {
		case role.Master, role.Auth:
			c.Send(nmc.ServerMessage, "available commands: keepteams 1|0 (=persist), queuemap <map>..., competitive 0|1")
		case role.Admin:
			c.Send(nmc.ServerMessage, "available commands: keepteams 1|0 (=persist), queuemap <map>..., competitive 0|1, ip <name|cn>")
		}

	case "queuemap", "queuedmap", "queuemaps", "queuedmaps", "mapqueue", "mapsqueue":
		queueMap(c, parts[1:])

	case "keepteams", "persist", "persistteams":
		toggleKeepTeams(c, parts[1:])

	case "competitive", "compet":
		toggleCompetitiveMode(c, parts[1:])

	case "reportstats":
		toggleReportStats(c, parts[1:])

	case "ip", "ips":
		lookupIPs(c, parts[1:])

	default:
		c.Send(nmc.ServerMessage, cubecode.Fail("unknown command"))
	}
}

func queueMap(c *Client, args []string) {
	if c.Role == role.None {
		return
	}
	for _, mapp := range args {
		err := s.MapRotation.queueMap(mapp)
		if err != "" {
			c.Send(nmc.ServerMessage, cubecode.Fail(err))
		}
	}
	switch len(s.MapRotation.queue) {
	case 0:
		c.Send(nmc.ServerMessage, "no maps queued")
	case 1:
		c.Send(nmc.ServerMessage, "queued map: "+s.MapRotation.queue[0])
	default:
		c.Send(nmc.ServerMessage, "queued maps: "+strings.Join(s.MapRotation.queue, ", "))
	}
}

func toggleKeepTeams(c *Client, args []string) {
	if c.Role == role.None {
		return
	}
	changed := false
	if len(args) >= 1 {
		val, err := strconv.Atoi(args[0])
		if err != nil || (val != 0 && val != 1) {
			return
		}
		changed = s.KeepTeams != (val == 1)
		s.KeepTeams = val == 1
	}
	if changed {
		if s.KeepTeams {
			s.Clients.Broadcast(nmc.ServerMessage, "teams will be kept")
		} else {
			s.Clients.Broadcast(nmc.ServerMessage, "teams will be shuffled")
		}
	} else {
		if s.KeepTeams {
			c.Send(nmc.ServerMessage, "teams will be kept")
		} else {
			c.Send(nmc.ServerMessage, "teams will be shuffled")
		}
	}
}

func toggleCompetitiveMode(c *Client, args []string) {
	if c.Role == role.None {
		return
	}
	changed := false
	if len(args) >= 1 {
		val, err := strconv.Atoi(args[0])
		if err != nil || (val != 0 && val != 1) {
			return
		}
		comp, active := s.GameMode.(*game.Competitive)
		changed = s.CompetitiveMode != (val == 1)
		switch val {
		case 1:
			// starts at next map
			s.CompetitiveMode = true
			// but lock server now
			s.SetMasterMode(c, mastermode.Locked)
		default:
			if active {
				// stops immediately
				s.GameMode = comp.ToCasual()
				s.CompetitiveMode = false
			}
		}
	}
	if changed {
		if s.CompetitiveMode {
			s.Clients.Broadcast(nmc.ServerMessage, "competitive mode will be enabled with next game")
		} else {
			s.Clients.Broadcast(nmc.ServerMessage, "competitive mode disabled")
		}
	} else {
		if s.CompetitiveMode {
			c.Send(nmc.ServerMessage, "competitive mode is on")
		} else {
			c.Send(nmc.ServerMessage, "competitive mode is off")
		}
	}
}

func toggleReportStats(c *Client, args []string) {
	if c.Role < role.Admin {
		return
	}
	changed := false
	if len(args) >= 1 {
		val, err := strconv.Atoi(args[0])
		if err != nil || (val != 0 && val != 1) {
			return
		}
		changed = s.ReportStats != (val == 1)
		s.ReportStats = val == 1
	}
	if changed {
		if s.ReportStats {
			s.Clients.Broadcast(nmc.ServerMessage, "stats will be reported at intermission")
		} else {
			s.Clients.Broadcast(nmc.ServerMessage, "stats will not be reported")
		}
	} else {
		if s.ReportStats {
			c.Send(nmc.ServerMessage, "stats reporting is on")
		} else {
			c.Send(nmc.ServerMessage, "stats reporting is off")
		}
	}
}

func lookupIPs(c *Client, args []string) {
	if c.Role < role.Admin || len(args) < 1 {
		return
	}
	for _, query := range args {
		var target *Client
		// try CN
		cn, err := strconv.Atoi(query)
		if err == nil {
			target = s.Clients.GetClientByCN(uint32(cn))
		}
		if err != nil || target == nil {
			target = s.Clients.FindClientByName(query)
		}

		if target != nil {
			c.Send(nmc.ServerMessage, fmt.Sprintf("%s has IP %s", s.Clients.UniqueName(target), target.Peer.Address.IP))
		} else {
			c.Send(nmc.ServerMessage, fmt.Sprintf("could not find a client matching '%s'", query))
		}
	}
}
