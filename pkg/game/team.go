package game

import (
	"sort"

	"github.com/sauerbraten/waiter/internal/rng"
	"github.com/sauerbraten/waiter/pkg/protocol/nmc"
	"github.com/sauerbraten/waiter/pkg/protocol/playerstate"
)

var NoTeam = &Team{Name: "none"}

type Team struct {
	Name    string
	Frags   int
	Score   int
	Players map[*Player]struct{}
}

func NewTeam(name string) *Team {
	return &Team{
		Name:    name,
		Players: map[*Player]struct{}{},
	}
}

// sorts teams ascending by size, then score
type BySizeAndScore []*Team

func (teams BySizeAndScore) Len() int {
	return len(teams)
}

func (teams BySizeAndScore) Swap(i, j int) {
	teams[i], teams[j] = teams[j], teams[i]
}

func (teams BySizeAndScore) Less(i, j int) bool {
	if len(teams[i].Players) != len(teams[j].Players) {
		return len(teams[i].Players) < len(teams[j].Players)
	}
	if teams[i].Score != teams[j].Score {
		return teams[i].Score < teams[j].Score
	}
	if teams[i].Frags != teams[j].Frags {
		return teams[i].Frags < teams[j].Frags
	}
	return rng.RNG.Intn(2) == 0
}

func (t *Team) Add(p *Player) {
	t.Players[p] = struct{}{}
	p.Team = t
}

func (t *Team) Remove(p *Player) {
	p.Team = NoTeam
	delete(t.Players, p)
}

type TeamMode interface {
	Mode
	Teams() map[string]*Team
	ForEach(func(*Team))
	ChangeTeam(*Player, string, bool)
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
