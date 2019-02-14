package main

import (
	"log"
	"sort"

	"github.com/sauerbraten/waiter/internal/definitions/armour"
	"github.com/sauerbraten/waiter/internal/definitions/gamemode"
	"github.com/sauerbraten/waiter/internal/definitions/nmc"
	"github.com/sauerbraten/waiter/internal/definitions/playerstate"
	"github.com/sauerbraten/waiter/internal/definitions/weapon"
	"github.com/sauerbraten/waiter/pkg/protocol"
)

type GameMode interface {
	ID() gamemode.ID
	NeedMapInfo() bool
	Join(*Client)
	Init(*Client)
	Leave(*Client)
	CanSpawn(*Client) bool
	Spawn(*GameState) // sets armour, ammo, and health
	HandleDeath(fragger, victim *Client)
	FragValue(fragger, victim *Client) int
	HandlePacket(*Client, nmc.ID, *protocol.Packet) bool
}

func StartGame(id gamemode.ID) GameMode {
	switch id {
	case gamemode.Insta:
		return NewInsta()
	case gamemode.InstaTeam:
		return NewInstaTeam(s.KeepTeams)
	case gamemode.Effic:
		return NewEffic()
	case gamemode.EfficTeam:
		return NewEfficTeam(s.KeepTeams)
	case gamemode.Tactics:
		return NewTactics()
	case gamemode.TacticsTeam:
		return NewTacticsTeam(s.KeepTeams)
	case gamemode.InstaCTF:
		return NewInstaCTF(s.KeepTeams)
	case gamemode.EfficCTF:
		return NewEfficCTF(s.KeepTeams)
	default:
		return nil
	}
}

type teamlessMode struct{}

func (*teamlessMode) Join(*Client) {}

func (*teamlessMode) Leave(*Client) {}

func (*teamlessMode) FragValue(fragger, victim *Client) int {
	if fragger == victim {
		return -1
	}
	return 1
}

// no spawn timeout
type deathmatchMode struct{}

func (*deathmatchMode) CanSpawn(*Client) bool { return true }

// no pick-ups, no flags, no bases
type noItemsMode struct{}

func (*noItemsMode) NeedMapInfo() bool { return false }

func (*noItemsMode) Init(*Client) {}

func (*noItemsMode) HandleDeath(*Client, *Client) {}

func (*noItemsMode) HandlePacket(*Client, nmc.ID, *protocol.Packet) bool { return false }

type TeamMode interface {
	GameMode
	Frags(*Team) int
	ForEach(func(*Team))
	ChangeTeam(*Client, string, bool)
}

type teamMode struct {
	Teams             map[string]*Team
	otherTeamsAllowed bool
	keepTeams         bool
}

func NewTeamMode(otherTeamsAllowed, keepTeams bool, names ...string) teamMode {
	teams := map[string]*Team{}
	for _, name := range names {
		teams[name] = NewTeam(name)
	}
	return teamMode{
		Teams:             teams,
		otherTeamsAllowed: otherTeamsAllowed,
		keepTeams:         keepTeams,
	}
}

func (tm *teamMode) selectTeam(c *Client) *Team {
	if tm.keepTeams {
		log.Println("trying to keep team")
		for _, t := range tm.Teams {
			if c.Team.Name == t.Name {
				log.Println("leaving", c, "in team", t.Name)
				return t
			}
		}
	}
	return tm.selectWeakestTeam()
}

func (tm *teamMode) selectWeakestTeam() *Team {
	teams := []*Team{}
	for _, team := range tm.Teams {
		teams = append(teams, team)
	}

	sort.Sort(BySizeAndScore(teams))
	return teams[0]
}

func (tm *teamMode) Join(c *Client) {
	team := tm.selectTeam(c)
	team.Add(c)
	s.Clients.Broadcast(nil, nmc.SetTeam, c.CN, c.Team.Name, -1)
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

func (tm *teamMode) ForEach(do func(t *Team)) {
	for _, team := range tm.Teams {
		do(team)
	}
}

func (tm *teamMode) Frags(t *Team) int { return t.Frags }

func (tm *teamMode) ChangeTeam(c *Client, newTeamName string, forced bool) {
	reason := -1 // = none = silent
	if c.GameState.State != playerstate.Spectator {
		if forced {
			reason = 1
		} else {
			reason = 0 // = voluntary
		}
	}

	setTeam := func(old, new *Team) {
		if c.GameState.State == playerstate.Alive {
			s.handleDeath(c, c)
		}
		old.Remove(c)
		new.Add(c)
		s.Clients.Broadcast(nil, nmc.SetTeam, c.CN, c.Team.Name, reason)
	}

	// try existing teams first
	for name, team := range tm.Teams {
		if name == newTeamName {
			// todo: check privileges and team balance
			setTeam(c.Team, team)
			return
		}
	}

	if tm.otherTeamsAllowed {
		newTeam := NewTeam(newTeamName)
		tm.Teams[newTeamName] = newTeam
		setTeam(c.Team, newTeam)
	}
}

type efficMode struct{}

func (*efficMode) Spawn(gs *GameState) {
	gs.ArmourType, gs.Armour = armour.Green, 100
	gs.Ammo, gs.SelectedWeapon = weapon.SpawnAmmoEffic()
}

type Effic struct {
	efficMode
	deathmatchMode
	noItemsMode
	teamlessMode
}

func NewEffic() GameMode { return &Effic{} }

func (*Effic) ID() gamemode.ID { return gamemode.Effic }

type EfficTeam struct {
	efficMode
	deathmatchMode
	noItemsMode
	teamMode
}

func NewEfficTeam(keepTeams bool) GameMode {
	return &EfficTeam{
		teamMode: NewTeamMode(true, keepTeams, "good", "evil"),
	}
}

func (*EfficTeam) ID() gamemode.ID { return gamemode.EfficTeam }

type instaMode struct{}

func (*instaMode) Spawn(gs *GameState) {
	gs.ArmourType, gs.Armour = armour.None, 0
	gs.Ammo, gs.SelectedWeapon = weapon.SpawnAmmoInsta()
	gs.Health, gs.MaxHealth = 1, 1
}

type Insta struct {
	instaMode
	deathmatchMode
	noItemsMode
	teamlessMode
}

func NewInsta() GameMode { return &Insta{} }

func (*Insta) ID() gamemode.ID { return gamemode.Insta }

type InstaTeam struct {
	instaMode
	deathmatchMode
	noItemsMode
	teamMode
}

func NewInstaTeam(keepTeams bool) GameMode {
	return &InstaTeam{
		teamMode: NewTeamMode(true, keepTeams, "good", "evil"),
	}
}

func (*InstaTeam) ID() gamemode.ID { return gamemode.InstaTeam }

type tacticsMode struct{}

func (*tacticsMode) Spawn(gs *GameState) {
	gs.ArmourType, gs.Armour = armour.Green, 100
	gs.Ammo, gs.SelectedWeapon = weapon.SpawnAmmoTactics()
	gs.Health, gs.MaxHealth = 1, 1
}

type Tactics struct {
	tacticsMode
	deathmatchMode
	noItemsMode
	teamlessMode
}

func NewTactics() GameMode { return &Tactics{} }

func (*Tactics) ID() gamemode.ID { return gamemode.Tactics }

type TacticsTeam struct {
	tacticsMode
	deathmatchMode
	noItemsMode
	teamMode
}

func NewTacticsTeam(keepTeams bool) GameMode {
	return &TacticsTeam{
		teamMode: NewTeamMode(true, keepTeams, "good", "evil"),
	}
}

func (*TacticsTeam) ID() gamemode.ID { return gamemode.TacticsTeam }
