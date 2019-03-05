package game

import (
	"github.com/sauerbraten/waiter/internal/utils"
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
	return utils.RNG.Intn(2) == 0
}

func (t *Team) Add(p *Player) {
	t.Players[p] = struct{}{}
	p.Team = t
}

func (t *Team) Remove(p *Player) {
	p.Team = NoTeam
	delete(t.Players, p)
}
