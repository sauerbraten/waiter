package game

import (
	"sort"

	"github.com/sauerbraten/waiter/pkg/protocol/nmc"
	"github.com/sauerbraten/waiter/pkg/protocol/playerstate"
)

type Teamed interface {
	Teams() map[string]*Team
	ForEach(func(*Team))
	ChangeTeam(*Player, string, bool)
}

type teamed struct {
	s                 Server
	teams             map[string]*Team
	otherTeamsAllowed bool
	keepTeams         bool
}

var _ Teamed = &teamed{}

func newTeamed(s Server, otherTeamsAllowed, keepTeams bool, names ...string) teamed {
	teams := map[string]*Team{}
	for _, name := range names {
		teams[name] = NewTeam(name)
	}
	return teamed{
		s:                 s,
		teams:             teams,
		otherTeamsAllowed: otherTeamsAllowed,
		keepTeams:         keepTeams,
	}
}

func (tm *teamed) selectTeam(p *Player) *Team {
	if tm.keepTeams {
		for _, t := range tm.teams {
			if p.Team.Name == t.Name {
				return t
			}
		}
	}
	return tm.selectWeakestTeam()
}

func (tm *teamed) selectWeakestTeam() *Team {
	teams := []*Team{}
	for _, team := range tm.teams {
		teams = append(teams, team)
	}

	sort.Sort(BySizeAndScore(teams))
	return teams[0]
}

func (tm *teamed) Join(p *Player) {
	team := tm.selectTeam(p)
	team.Add(p)
	tm.s.Broadcast(nmc.SetTeam, p.CN, p.Team.Name, -1)
}

func (*teamed) Leave(p *Player) {
	p.Team.Remove(p)
}

func (tm *teamed) HandleFrag(fragger, victim *Player) {
	victim.Die()
	if fragger.Team == victim.Team {
		fragger.Frags--
	} else {
		fragger.Frags++
	}
	tm.s.Broadcast(nmc.Died, victim.CN, fragger.CN, fragger.Frags, fragger.Team.Frags)
}

func (tm *teamed) ForEach(do func(t *Team)) {
	for _, team := range tm.teams {
		do(team)
	}
}

func (tm *teamed) Teams() map[string]*Team {
	return tm.teams
}

func (tm *teamed) ChangeTeam(p *Player, newTeamName string, forced bool) {
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
