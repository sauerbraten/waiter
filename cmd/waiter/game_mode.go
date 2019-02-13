package main

import (
	"log"
	"sort"

	"github.com/sauerbraten/waiter/internal/definitions/gamemode"
	"github.com/sauerbraten/waiter/internal/definitions/nmc"
	"github.com/sauerbraten/waiter/pkg/protocol"
)

type GameMode interface {
	ID() gamemode.ID
	NeedMapInfo() bool
	Join(*Client)
	Init(*Client)
	Leave(*Client)
	CanSpawn(*Client) bool
	HandleDeath(fragger, victim *Client)
	FragValue(fragger, victim *Client) int
	HandlePacket(*Client, nmc.ID, *protocol.Packet) bool
}

type teamlessMode struct{}

func (*teamlessMode) Join(*Client) {}

func (*teamlessMode) Init(*Client) {}

func (*teamlessMode) Leave(*Client) {}

func (*teamlessMode) CanSpawn(*Client) bool { return true }

func (*teamlessMode) HandleDeath(*Client, *Client) {}

func (*teamlessMode) FragValue(fragger, victim *Client) int {
	if fragger == victim {
		return -1
	}
	return 1
}

type noItemsMode struct{}

func (*noItemsMode) NeedMapInfo() bool { return false }

func (*noItemsMode) HandlePacket(*Client, nmc.ID, *protocol.Packet) bool { return false }

type Effic struct {
	noItemsMode
	teamlessMode
}

func (*Effic) ID() gamemode.ID { return gamemode.Effic }

type Insta struct {
	noItemsMode
	teamlessMode
}

func (*Insta) ID() gamemode.ID { return gamemode.Insta }

type Tactics struct {
	noItemsMode
	teamlessMode
}

func (*Tactics) ID() gamemode.ID { return gamemode.Tactics }

type TeamMode interface {
	GameMode
	Frags(*Team) int
	ForEach(func(*Team))
}

type teamMode struct {
	Teams map[string]*Team
}

func NewTeamMode(names ...string) teamMode {
	teams := map[string]*Team{}
	for _, name := range names {
		teams[name] = NewTeam(name)
	}
	return teamMode{
		Teams: teams,
	}
}

func (t *teamMode) selectWeakestTeam() *Team {
	teams := []*Team{}
	for _, team := range t.Teams {
		teams = append(teams, team)
	}

	sort.Sort(ByScoreAndSize(teams))
	return teams[0]
}

func (t *teamMode) Join(c *Client) {
	team := t.selectWeakestTeam()
	team.Add(c)
	log.Println("weakest team:", team.Name)
}

func (*teamMode) Leave(c *Client) {
	c.Team.Remove(c)
}

func (*teamMode) FragValue(fragger, victim *Client) int {
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

func (tm *teamMode) Frags(t *Team) int { return t.Frags }

func GameModeByID(id gamemode.ID) GameMode {
	switch id {
	case gamemode.Insta:
		return &Insta{}
	case gamemode.Effic:
		return &Effic{}
	case gamemode.Tactics:
		return &Tactics{}
	case gamemode.EfficCTF:
		return NewEfficCTF()
	default:
		return nil
	}
}
