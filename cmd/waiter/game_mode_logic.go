package main

import (
	"sort"

	"github.com/sauerbraten/waiter/internal/definitions/gamemode"
	"github.com/sauerbraten/waiter/internal/utils"
)

type GameMode interface {
	ID() gamemode.ID
	Init()
	Join(*Client)
	CountFrag(fragger, victim *Client) int
}

type teamlessMode struct{}

func (*teamlessMode) Init() {}

func (*teamlessMode) Join(c *Client) {}

func (*teamlessMode) CountFrag(fragger, victim *Client) int {
	if fragger == victim {
		return -1
	}
	return 1
}

type Effic struct {
	teamlessMode
}

func (*Effic) ID() gamemode.ID { return gamemode.Effic }

type Insta struct {
	teamlessMode
}

func (*Insta) ID() gamemode.ID { return gamemode.Insta }

type Tactics struct {
	teamlessMode
}

func (*Tactics) ID() gamemode.ID { return gamemode.Tactics }

type TeamMode interface {
	GameMode
	Frags(string) int
	ForEach(func(*Team))
}

type teamMode struct {
	Teams map[string]*Team
}

func (t *teamMode) selectWeakestTeam() string {
	teams := []*Team{}
	for _, team := range t.Teams {
		teams = append(teams, team)
	}

	sort.Sort(ByScore(teams))
	if teams[0].Score() < teams[1].Score() {
		return teams[0].Name
	}

	sort.Sort(BySize(teams))
	if len(teams[0].Players) < len(teams[0].Players) {
		return teams[0].Name
	}

	return teams[utils.RNG.Int31n(2)].Name
}

func (t *teamMode) Init() {
	t.Teams = map[string]*Team{}
}

func (t *teamMode) Join(c *Client) {
	team := t.selectWeakestTeam()
	c.Team = team
	t.Teams[team].AddPlayer(c)
}

func (*teamMode) CountFrag(fragger, victim *Client) int {
	if fragger.Team == victim.Team {
		return -1
	}
	return 1
}

func (t *teamMode) ForEach(do func(t *Team)) {
	for _, team := range t.Teams {
		do(team)
	}
}

func (t *teamMode) Frags(name string) int {
	if team, ok := t.Teams[name]; ok {
		return team.Frags
	}
	return 0
}

type CTF struct {
	teamMode
}

func (*CTF) ID() gamemode.ID { return gamemode.CTF }

func (ctf *CTF) Join(c *Client) {
	ctf.teamMode.Join(c)
}

func (ctf *CTF) Init() {
	ctf.teamMode.Init()
	ctf.Teams["good"] = &Team{}
	ctf.Teams["evil"] = &Team{}

	s.Clients.ForEach(func(c *Client) { ctf.Join(c) })
}

type EfficCTF struct {
	CTF
}

func (*EfficCTF) ID() gamemode.ID { return gamemode.EfficCTF }

type Capture struct {
	teamMode
}

func (*Capture) ID() gamemode.ID { return gamemode.Capture }

func (cap *Capture) Join(c *Client) {
	cap.teamMode.Join(c)
}

func (cap *Capture) Init() {
	cap.teamMode.Init()
	cap.Teams["guht"] = &Team{}
	cap.Teams["pöse"] = &Team{}

	s.Clients.ForEach(func(c *Client) { cap.Join(c) })

}

func GameModeByID(id gamemode.ID) GameMode {
	switch id {
	case gamemode.Insta:
		return &Insta{}
	case gamemode.Effic:
		return &Effic{}
	case gamemode.Tactics:
		return &Tactics{}
	default:
		return nil
	}
}
