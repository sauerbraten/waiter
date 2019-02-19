package main

import (
	"log"
	"sort"
	"time"

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
	Start()
	Intermission()
	CleanUp()

	Pause(*Client)
	Paused() bool
	Resume(*Client)
	TimeLeft() time.Duration

	Join(*Client)
	Init(*Client)
	Leave(*Client)
	CanSpawn(*Client) bool
	Spawn(*GameState) // sets armour, ammo, and health
	ConfirmSpawn(*Client)

	FragValue(fragger, victim *Client) int
	HandleDeath(fragger, victim *Client)
	HandlePacket(*Client, nmc.ID, *protocol.Packet) bool
}

type TeamMode interface {
	GameMode
	Teams() map[string]*Team
	ForEach(func(*Team))
	ChangeTeam(*Client, string, bool)
}

type timedMode struct {
	t *GameTimer
}

func newTimedMode(d time.Duration, intermission func()) timedMode {
	return timedMode{StartTimer(d, intermission)}
}

func (tm *timedMode) Pause(c *Client) {
	cn := -1
	if c != nil {
		cn = int(c.CN)
	}
	s.Clients.Broadcast(nil, nmc.PauseGame, 1, cn)
	tm.t.Pause()
}

func (tm *timedMode) Paused() bool { return tm.t.Paused() }

func (tm *timedMode) Resume(c *Client) {
	cn := -1
	if c != nil {
		cn = int(c.CN)
	}
	s.Clients.Broadcast(nil, nmc.PauseGame, 0, cn)
	tm.t.Resume()
}

func (tm *timedMode) TimeLeft() time.Duration { return tm.t.TimeLeft }

// methods that are shadowed by CompetitiveMode so all modes implement GameMode
type casualMode struct{}

func (*casualMode) Start() {}

func (*casualMode) ConfirmSpawn(*Client) {}

// prints intermission stats based on frags
type deathmatchMode struct {
	timedMode
}

func newDeathmatchMode(duration time.Duration, intermission func()) deathmatchMode {
	return deathmatchMode{
		timedMode: newTimedMode(duration, intermission),
	}
}

func (*deathmatchMode) Intermission() {
	// todo: print some stats
}

// no spawn timeout
type noSpawnWaitMode struct{}

func (*noSpawnWaitMode) CanSpawn(*Client) bool { return true }

// no pick-ups, no flags, no bases
type noItemsMode struct{}

func (*noItemsMode) NeedMapInfo() bool { return false }

func (*noItemsMode) Init(*Client) {}

func (*noItemsMode) HandleDeath(*Client, *Client) {}

func (*noItemsMode) HandlePacket(*Client, nmc.ID, *protocol.Packet) bool { return false }

func (*noItemsMode) CleanUp() {}

type teamlessMode struct{}

func (*teamlessMode) Join(*Client) {}

func (*teamlessMode) Leave(*Client) {}

func (*teamlessMode) FragValue(fragger, victim *Client) int {
	if fragger == victim {
		return -1
	}
	return 1
}

type teamMode struct {
	teams             map[string]*Team
	otherTeamsAllowed bool
	keepTeams         bool
}

func newTeamMode(otherTeamsAllowed, keepTeams bool, names ...string) teamMode {
	teams := map[string]*Team{}
	for _, name := range names {
		teams[name] = NewTeam(name)
	}
	return teamMode{
		teams:             teams,
		otherTeamsAllowed: otherTeamsAllowed,
		keepTeams:         keepTeams,
	}
}

func (tm *teamMode) selectTeam(c *Client) *Team {
	if tm.keepTeams {
		log.Println("trying to keep team")
		for _, t := range tm.teams {
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
	for _, team := range tm.teams {
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
	for _, team := range tm.teams {
		do(team)
	}
}

func (tm *teamMode) Teams() map[string]*Team {
	return tm.teams
}

func (tm *teamMode) ChangeTeam(c *Client, newTeamName string, forced bool) {
	reason := -1 // = none = silent
	if c.GameState.State != playerstate.Spectator {
		if forced {
			reason = 1 // = forced
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
	for name, team := range tm.teams {
		if name == newTeamName {
			// todo: check privileges and team balance
			setTeam(c.Team, team)
			return
		}
	}

	if tm.otherTeamsAllowed {
		newTeam := NewTeam(newTeamName)
		tm.teams[newTeamName] = newTeam
		setTeam(c.Team, newTeam)
	}
}

type teamDeathmatchMode struct {
	teamMode
	deathmatchMode
}

func newTeamDeathmatchMode(duration time.Duration, keepTeams bool, intermission func()) teamDeathmatchMode {
	return teamDeathmatchMode{
		teamMode:       newTeamMode(true, keepTeams, "good", "evil"),
		deathmatchMode: newDeathmatchMode(duration, intermission),
	}
}

type efficMode struct{}

func (*efficMode) Spawn(gs *GameState) {
	gs.ArmourType, gs.Armour = armour.Green, 100
	gs.Ammo, gs.SelectedWeapon = weapon.SpawnAmmoEffic()
}

type Effic struct {
	casualMode
	deathmatchMode
	efficMode
	noSpawnWaitMode
	noItemsMode
	teamlessMode
}

func NewEffic(duration time.Duration) *Effic {
	var effic *Effic
	effic = &Effic{
		deathmatchMode: newDeathmatchMode(duration, func() { effic.Intermission() }),
	}
	return effic
}

func (*Effic) ID() gamemode.ID { return gamemode.Effic }

type EfficTeam struct {
	casualMode
	teamDeathmatchMode
	efficMode
	noSpawnWaitMode
	noItemsMode
}

// assert interface implementations at compile time
var (
	_ GameMode = NewEfficTeam(1*time.Minute, false)
	_ TeamMode = NewEfficTeam(1*time.Minute, false)
)

func NewEfficTeam(duration time.Duration, keepTeams bool) *EfficTeam {
	var efficTeam *EfficTeam
	efficTeam = &EfficTeam{
		teamDeathmatchMode: newTeamDeathmatchMode(duration, keepTeams, func() { efficTeam.Intermission() }),
	}
	return efficTeam
}

func (*EfficTeam) ID() gamemode.ID { return gamemode.EfficTeam }

type instaMode struct{}

func (*instaMode) Spawn(gs *GameState) {
	gs.ArmourType, gs.Armour = armour.None, 0
	gs.Ammo, gs.SelectedWeapon = weapon.SpawnAmmoInsta()
	gs.Health, gs.MaxHealth = 1, 1
}

type Insta struct {
	timedMode
	casualMode
	deathmatchMode
	instaMode
	noSpawnWaitMode
	noItemsMode
	teamlessMode
}

// assert interface implementations at compile time
var (
	_ GameMode = NewInsta(1 * time.Minute)
)

func NewInsta(duration time.Duration) *Insta {
	var insta *Insta
	insta = &Insta{
		timedMode: newTimedMode(duration, func() { insta.Intermission() }),
	}
	return insta
}

func (*Insta) ID() gamemode.ID { return gamemode.Insta }

type InstaTeam struct {
	casualMode
	teamDeathmatchMode
	instaMode
	noSpawnWaitMode
	noItemsMode
}

// assert interface implementations at compile time
var (
	_ GameMode = NewInstaTeam(1*time.Minute, false)
	_ TeamMode = NewInstaTeam(1*time.Minute, false)
)

func NewInstaTeam(duration time.Duration, keepTeams bool) *InstaTeam {
	var instaTeam *InstaTeam
	instaTeam = &InstaTeam{
		teamDeathmatchMode: newTeamDeathmatchMode(duration, keepTeams, func() { instaTeam.Intermission() }),
	}
	return instaTeam
}

func (*InstaTeam) ID() gamemode.ID { return gamemode.InstaTeam }

type tacticsMode struct{}

func (*tacticsMode) Spawn(gs *GameState) {
	gs.ArmourType, gs.Armour = armour.Green, 100
	gs.Ammo, gs.SelectedWeapon = weapon.SpawnAmmoTactics()
	gs.Health, gs.MaxHealth = 1, 1
}

type Tactics struct {
	timedMode
	casualMode
	deathmatchMode
	tacticsMode
	noSpawnWaitMode
	noItemsMode
	teamlessMode
}

// assert interface implementations at compile time
var (
	_ GameMode = NewTactics(1 * time.Minute)
)

func NewTactics(duration time.Duration) *Tactics {
	var tactics *Tactics
	tactics = &Tactics{
		timedMode: newTimedMode(duration, func() { tactics.Intermission() }),
	}
	return tactics
}

func (*Tactics) ID() gamemode.ID { return gamemode.Tactics }

type TacticsTeam struct {
	casualMode
	teamDeathmatchMode
	tacticsMode
	noSpawnWaitMode
	noItemsMode
}

// assert interface implementations at compile time
var (
	_ GameMode = NewTacticsTeam(1*time.Minute, false)
	_ TeamMode = NewTacticsTeam(1*time.Minute, false)
)

func NewTacticsTeam(duration time.Duration, keepTeams bool) *TacticsTeam {
	var tacticsTeam *TacticsTeam
	tacticsTeam = &TacticsTeam{
		teamDeathmatchMode: newTeamDeathmatchMode(duration, keepTeams, func() { tacticsTeam.Intermission() }),
	}
	return tacticsTeam
}

func (*TacticsTeam) ID() gamemode.ID { return gamemode.TacticsTeam }
