package main

import (
	"sort"

	"github.com/sauerbraten/waiter/internal/utils"
)

type GameMode interface {
	Init()
	Join(*Client)
}

type TeamMode struct {
	Teams map[string]*Team
}

func (t *TeamMode) selectWeakestTeam() string {
	teams := []*Team{}
	for _, team := range t.Teams {
		teams = append(teams, team)
	}

	sort.Sort(ByScore(teams))

	if teams[0].Score < teams[1].Score {
		return teams[0].Name
	}

	sort.Sort(BySize(teams))
	if len(teams[0].Players) < len(teams[0].Players) {
		return teams[0].Name
	}

	return teams[utils.RNG.Int31n(2)].Name
}

func (t *TeamMode) Init() {
	t.Teams = map[string]*Team{}
}

func (t *TeamMode) Join(c *Client) {
	team := t.selectWeakestTeam()
	c.Team = team
	t.Teams[team].AddPlayer(c)
}

type CTF struct {
	TeamMode
}

func (ctf *CTF) Join(c *Client) {
	ctf.TeamMode.Join(c)
}

func (ctf *CTF) Init() {
	ctf.TeamMode.Init()
	ctf.Teams["good"] = &Team{}
	ctf.Teams["evil"] = &Team{}

	s.Clients.ForEach(func(c *Client) { ctf.Join(c) })
}

type Capture struct {
	TeamMode
}

func (cap *Capture) Join(c *Client) {
	cap.TeamMode.Join(c)
}

func (cap *Capture) Init() {
	cap.TeamMode.Init()
	cap.Teams["guht"] = &Team{}
	cap.Teams["pöse"] = &Team{}

	s.Clients.ForEach(func(c *Client) { cap.Join(c) })

}
