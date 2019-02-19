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
	End()
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

func newTimedMode() timedMode {
	return timedMode{StartTimer(s.GameDuration*time.Minute, s.Intermission)}
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

func (tm *timedMode) End() {
	tm.t.Stop()
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

func newDeathmatchMode() deathmatchMode {
	return deathmatchMode{
		timedMode: newTimedMode(),
	}
}

func (dm *deathmatchMode) End() {
	dm.timedMode.End()
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
}

func newTeamMode(otherTeamsAllowed bool, names ...string) teamMode {
	teams := map[string]*Team{}
	for _, name := range names {
		teams[name] = NewTeam(name)
	}
	return teamMode{
		teams:             teams,
		otherTeamsAllowed: otherTeamsAllowed,
	}
}

func (tm *teamMode) selectTeam(c *Client) *Team {
	if s.KeepTeams {
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

func newTeamDeathmatchMode() teamDeathmatchMode {
	return teamDeathmatchMode{
		teamMode:       newTeamMode(true, "good", "evil"),
		deathmatchMode: newDeathmatchMode(),
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

func NewEffic() *Effic {
	var effic *Effic
	effic = &Effic{
		deathmatchMode: newDeathmatchMode(),
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
	_ GameMode = &EfficTeam{}
	_ TeamMode = &EfficTeam{}
)

func NewEfficTeam() *EfficTeam {
	var efficTeam *EfficTeam
	efficTeam = &EfficTeam{
		teamDeathmatchMode: newTeamDeathmatchMode(),
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
	casualMode
	deathmatchMode
	instaMode
	noSpawnWaitMode
	noItemsMode
	teamlessMode
}

// assert interface implementations at compile time
var (
	_ GameMode = &Insta{}
)

func NewInsta() *Insta {
	var insta *Insta
	insta = &Insta{
		deathmatchMode: newDeathmatchMode(),
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
	_ GameMode = &InstaTeam{}
	_ TeamMode = &InstaTeam{}
)

func NewInstaTeam() *InstaTeam {
	var instaTeam *InstaTeam
	instaTeam = &InstaTeam{
		teamDeathmatchMode: newTeamDeathmatchMode(),
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
	casualMode
	deathmatchMode
	tacticsMode
	noSpawnWaitMode
	noItemsMode
	teamlessMode
}

// assert interface implementations at compile time
var (
	_ GameMode = &Tactics{}
)

func NewTactics() *Tactics {
	var tactics *Tactics
	tactics = &Tactics{
		deathmatchMode: newDeathmatchMode(),
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
	_ GameMode = &TacticsTeam{}
	_ TeamMode = &TacticsTeam{}
)

func NewTacticsTeam() *TacticsTeam {
	var tacticsTeam *TacticsTeam
	tacticsTeam = &TacticsTeam{
		teamDeathmatchMode: newTeamDeathmatchMode(),
	}
	return tacticsTeam
}

func (*TacticsTeam) ID() gamemode.ID { return gamemode.TacticsTeam }
