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
	TeamFrags(string) int
}

type NonTeamMode struct{}

func (_ *NonTeamMode) Init() {}

func (_ *NonTeamMode) Join(c *Client) {}

func (_ *NonTeamMode) CountFrag(fragger, victim *Client) int {
	if fragger == victim {
		return -1
	}
	return 1
}

func (_ *NonTeamMode) TeamFrags(_ string) int { return 0 }

type Effic struct {
	NonTeamMode
}

func (_ *Effic) ID() gamemode.ID { return gamemode.Effic }

type Insta struct {
	NonTeamMode
}

func (_ *Insta) ID() gamemode.ID { return gamemode.Insta }

type TeamMode struct {
	Teams map[string]*Team
}

func (t *TeamMode) selectWeakestTeam() string {
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

func (t *TeamMode) Init() {
	t.Teams = map[string]*Team{}
}

func (t *TeamMode) Join(c *Client) {
	team := t.selectWeakestTeam()
	c.Team = team
	t.Teams[team].AddPlayer(c)
}

func (_ *TeamMode) CountFrag(fragger, victim *Client) int {
	if fragger.Team == victim.Team {
		return -1
	}
	return 1
}

func (t *TeamMode) TeamFrags(name string) int {
	if team, ok := t.Teams[name]; ok {
		return team.Frags
	}
	return 0
}

type CTF struct {
	TeamMode
}

func (_ *CTF) ID() gamemode.ID { return gamemode.CTF }

func (ctf *CTF) Join(c *Client) {
	ctf.TeamMode.Join(c)
}

func (ctf *CTF) Init() {
	ctf.TeamMode.Init()
	ctf.Teams["good"] = &Team{}
	ctf.Teams["evil"] = &Team{}

	s.Clients.ForEach(func(c *Client) { ctf.Join(c) })
}

type EfficCTF struct {
	CTF
}

func (_ *EfficCTF) ID() gamemode.ID { return gamemode.EfficCTF }

type Capture struct {
	TeamMode
}

func (_ *Capture) ID() gamemode.ID { return gamemode.Capture }

func (cap *Capture) Join(c *Client) {
	cap.TeamMode.Join(c)
}

func (cap *Capture) Init() {
	cap.TeamMode.Init()
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
	default:
		return nil
	}
}
