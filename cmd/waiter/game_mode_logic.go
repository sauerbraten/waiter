package main

import (
	"log"
	"sort"

	"github.com/sauerbraten/waiter/internal/definitions/gamemode"
	"github.com/sauerbraten/waiter/internal/definitions/nmc"
	"github.com/sauerbraten/waiter/internal/geom"
	"github.com/sauerbraten/waiter/internal/utils"
	"github.com/sauerbraten/waiter/pkg/protocol"
)

type GameMode interface {
	ID() gamemode.ID
	Init()
	Join(*Client)
	Leave(*Client)
	CountFrag(fragger, victim *Client) int
}

type teamlessMode struct{}

func (*teamlessMode) Init() {}

func (*teamlessMode) Join(c *Client) {}

func (*teamlessMode) Leave(c *Client) {}

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

func (*teamMode) Leave(c *Client) {}

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

type Flag struct {
	ID            int32
	Team          int32
	SpawnLocation *geom.Vector
}

type FlagMode struct {
	Flags []Flag
}

func (fm *FlagMode) handlePacket(client *Client, packetType nmc.ID, p protocol.Packet) {
	switch packetType {
	case nmc.InitFlags:
		fm.parseFlags(p)
	}
}
func (fm *FlagMode) parseFlags(p protocol.Packet) {
	numFlags, ok := p.GetInt()
	if !ok {
		log.Println("could not read number of flags from initflags packet (packet too short):", p)
		return
	}
	fm.Flags = []Flag{}
	var i int32
	for i < numFlags {
		team, ok := p.GetInt()
		if !ok {
			log.Println("could not read team from initflags packet (packet too short):", p)
			return
		}
		pos, ok := parseVector(p)
		if !ok {
			log.Println("could not read flag position from initflags packet (packet too short):", p)
			return
		}
		pos = pos.Mul(1 / geom.DMF)
		fm.Flags = append(fm.Flags, Flag{
			ID:            i,
			Team:          team,
			SpawnLocation: pos,
		})
	}
	return
}

type CTF struct {
	teamMode
}

func (*CTF) ID() gamemode.ID { return gamemode.CTF }

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
	case gamemode.EfficCTF:
		return &EfficCTF{}
	default:
		return nil
	}
}
