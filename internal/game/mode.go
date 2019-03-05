package game

import (
	"sort"
	"time"

	"github.com/sauerbraten/waiter/pkg/definitions/armour"
	"github.com/sauerbraten/waiter/pkg/definitions/gamemode"
	"github.com/sauerbraten/waiter/pkg/definitions/nmc"
	"github.com/sauerbraten/waiter/pkg/definitions/playerstate"
	"github.com/sauerbraten/waiter/pkg/definitions/weapon"
	"github.com/sauerbraten/waiter/pkg/protocol"
)

type Mode interface {
	ID() gamemode.ID

	NeedMapInfo() bool
	Start()
	End()
	Ended() bool
	CleanUp()

	Pause(*Player)
	Paused() bool
	Resume(*Player)
	TimeLeft() time.Duration

	Join(*Player)
	Init(*Player)
	Leave(*Player)
	CanSpawn(*Player) bool
	Spawn(*Player) // sets armour, ammo, and health
	ConfirmSpawn(*Player)

	HandleFrag(fragger, victim *Player)
	HandlePacket(*Player, nmc.ID, *protocol.Packet) bool
}

type TeamMode interface {
	Mode
	Teams() map[string]*Team
	ForEach(func(*Team))
	ChangeTeam(*Player, string, bool)
}

type timedMode struct {
	t *Timer
	s Server
}

func newTimedMode(s Server) timedMode {
	return timedMode{
		s: s,
	}
}
func (tm *timedMode) Start() {
	tm.t = StartTimer(tm.s.GameDuration(), tm.s.Intermission)
	tm.s.Broadcast(nmc.TimeLeft, tm.s.GameDuration())
}

func (tm *timedMode) Pause(p *Player) {
	cn := -1
	if p != nil {
		cn = int(p.CN)
	}
	tm.s.Broadcast(nmc.PauseGame, 1, cn)
	tm.t.Pause()
}

func (tm *timedMode) Paused() bool { return tm.t.Paused() }

func (tm *timedMode) Resume(p *Player) {
	cn := -1
	if p != nil {
		cn = int(p.CN)
	}
	tm.s.Broadcast(nmc.PauseGame, 0, cn)
	tm.t.Resume()
}

func (tm *timedMode) End() {
	tm.s.Broadcast(nmc.TimeLeft, 0)
	tm.t.Stop()
}

func (tm *timedMode) Ended() bool { return tm.t.Stopped() }

func (tm *timedMode) CleanUp() {
	if tm.Paused() {
		tm.Resume(nil)
	}
	tm.t.Stop()
}

func (tm *timedMode) TimeLeft() time.Duration { return tm.t.TimeLeft }

// methods that are shadowed by CompetitiveMode so all modes implement GameMode
type casualMode struct {
	s Server
}

func newCasualMode(s Server) casualMode {
	return casualMode{
		s: s,
	}
}

func (*casualMode) ConfirmSpawn(*Player) {}

// prints intermission stats based on frags
type deathmatchMode struct {
	timedMode
}

func newDeathmatchMode(s Server) deathmatchMode {
	return deathmatchMode{
		timedMode: newTimedMode(s),
	}
}

func (dm *deathmatchMode) End() {
	dm.timedMode.End()
	// todo: print some stats
}

// no spawn timeout
type noSpawnWaitMode struct{}

func (*noSpawnWaitMode) CanSpawn(*Player) bool { return true }

// no pick-ups, no flags, no bases
type noItemsMode struct{}

func (*noItemsMode) NeedMapInfo() bool { return false }

func (*noItemsMode) Init(*Player) {}

func (*noItemsMode) HandlePacket(*Player, nmc.ID, *protocol.Packet) bool { return false }

type teamlessMode struct {
	s Server
}

func newTeamlessMode(s Server) teamlessMode {
	return teamlessMode{
		s: s,
	}
}

func (*teamlessMode) Join(*Player) {}

func (*teamlessMode) Leave(*Player) {}

func (tlm *teamlessMode) HandleFrag(actor, victim *Player) {
	victim.Die()
	if actor == victim {
		actor.Frags--
	}
	actor.Frags++
	tlm.s.Broadcast(nmc.Died, victim.CN, actor.CN, actor.Frags, actor.Team.Frags)
}

type teamMode struct {
	s                 Server
	teams             map[string]*Team
	otherTeamsAllowed bool
	keepTeams         bool
}

func newTeamMode(s Server, otherTeamsAllowed, keepTeams bool, names ...string) teamMode {
	teams := map[string]*Team{}
	for _, name := range names {
		teams[name] = NewTeam(name)
	}
	return teamMode{
		s:                 s,
		teams:             teams,
		otherTeamsAllowed: otherTeamsAllowed,
		keepTeams:         keepTeams,
	}
}

func (tm *teamMode) selectTeam(p *Player) *Team {
	if tm.keepTeams {
		for _, t := range tm.teams {
			if p.Team.Name == t.Name {
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

func (tm *teamMode) Join(p *Player) {
	team := tm.selectTeam(p)
	team.Add(p)
	tm.s.Broadcast(nmc.SetTeam, p.CN, p.Team.Name, -1)
}

func (*teamMode) Leave(p *Player) {
	p.Team.Remove(p)
}

func (tm *teamMode) HandleFrag(fragger, victim *Player) {
	victim.Die()
	if fragger.Team == victim.Team {
		fragger.Frags--
	} else {
		fragger.Frags++
	}
	tm.s.Broadcast(nmc.Died, victim.CN, fragger.CN, fragger.Frags, fragger.Team.Frags)
}

func (tm *teamMode) ForEach(do func(t *Team)) {
	for _, team := range tm.teams {
		do(team)
	}
}

func (tm *teamMode) Teams() map[string]*Team {
	return tm.teams
}

func (tm *teamMode) ChangeTeam(p *Player, newTeamName string, forced bool) {
	reason := -1 // = none = silent
	if p.State != playerstate.Spectator {
		if forced {
			reason = 1 // = forced
		} else {
			reason = 0 // = voluntary
		}
	}

	setTeam := func(old, new *Team) {
		if p.State == playerstate.Alive {
			tm.HandleFrag(p, p)
		}
		old.Remove(p)
		new.Add(p)
		tm.s.Broadcast(nmc.SetTeam, p.CN, p.Team.Name, reason)
	}

	// try existing teams first
	for name, team := range tm.teams {
		if name == newTeamName {
			// todo: check privileges and team balance
			setTeam(p.Team, team)
			return
		}
	}

	if tm.otherTeamsAllowed {
		newTeam := NewTeam(newTeamName)
		tm.teams[newTeamName] = newTeam
		setTeam(p.Team, newTeam)
	}
}

type teamDeathmatchMode struct {
	teamMode
	deathmatchMode
}

func newTeamDeathmatchMode(s Server, keepTeams bool) teamDeathmatchMode {
	return teamDeathmatchMode{
		teamMode:       newTeamMode(s, true, keepTeams, "good", "evil"),
		deathmatchMode: newDeathmatchMode(s),
	}
}

type efficMode struct{}

func (*efficMode) Spawn(p *Player) {
	p.ArmourType, p.Armour = armour.Green, 100
	p.Ammo, p.SelectedWeapon = weapon.SpawnAmmoEffic()
	p.Health = 100
}

type Effic struct {
	casualMode
	deathmatchMode
	efficMode
	noSpawnWaitMode
	noItemsMode
	teamlessMode
}

func NewEffic(s Server) *Effic {
	var effic *Effic
	effic = &Effic{
		deathmatchMode: newDeathmatchMode(s),
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
	_ Mode     = &EfficTeam{}
	_ TeamMode = &EfficTeam{}
)

func NewEfficTeam(s Server, keepTeams bool) *EfficTeam {
	var efficTeam *EfficTeam
	efficTeam = &EfficTeam{
		teamDeathmatchMode: newTeamDeathmatchMode(s, keepTeams),
	}
	return efficTeam
}

func (*EfficTeam) ID() gamemode.ID { return gamemode.EfficTeam }

type instaMode struct{}

func (*instaMode) Spawn(p *Player) {
	p.ArmourType, p.Armour = armour.None, 0
	p.Ammo, p.SelectedWeapon = weapon.SpawnAmmoInsta()
	p.Health, p.MaxHealth = 1, 1
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
	_ Mode = &Insta{}
)

func NewInsta(s Server) *Insta {
	var insta *Insta
	insta = &Insta{
		deathmatchMode: newDeathmatchMode(s),
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
	_ Mode     = &InstaTeam{}
	_ TeamMode = &InstaTeam{}
)

func NewInstaTeam(s Server, keepTeams bool) *InstaTeam {
	var instaTeam *InstaTeam
	instaTeam = &InstaTeam{
		teamDeathmatchMode: newTeamDeathmatchMode(s, keepTeams),
	}
	return instaTeam
}

func (*InstaTeam) ID() gamemode.ID { return gamemode.InstaTeam }

type tacticsMode struct{}

func (*tacticsMode) Spawn(p *Player) {
	p.ArmourType, p.Armour = armour.Green, 100
	p.Ammo, p.SelectedWeapon = weapon.SpawnAmmoTactics()
	p.Health, p.MaxHealth = 1, 1
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
	_ Mode = &Tactics{}
)

func NewTactics(s Server) *Tactics {
	var tactics *Tactics
	tactics = &Tactics{
		deathmatchMode: newDeathmatchMode(s),
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
	_ Mode     = &TacticsTeam{}
	_ TeamMode = &TacticsTeam{}
)

func NewTacticsTeam(s Server, keepTeams bool) *TacticsTeam {
	var tacticsTeam *TacticsTeam
	tacticsTeam = &TacticsTeam{
		teamDeathmatchMode: newTeamDeathmatchMode(s, keepTeams),
	}
	return tacticsTeam
}

func (*TacticsTeam) ID() gamemode.ID { return gamemode.TacticsTeam }
