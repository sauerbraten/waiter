package main

import "github.com/sauerbraten/waiter/internal/utils"

var NoTeam = &Team{Name: "none"}

type Team struct {
	Name    string
	Frags   int
	Score   int
	Players map[*Client]struct{}
}

func NewTeam(name string) *Team {
	return &Team{
		Name:    name,
		Players: map[*Client]struct{}{},
	}
}

// sorts teams ascending by score, then size
type ByScoreAndSize []*Team

func (teams ByScoreAndSize) Len() int {
	return len(teams)
}

func (teams ByScoreAndSize) Swap(i, j int) {
	teams[i], teams[j] = teams[j], teams[i]
}

func (teams ByScoreAndSize) Less(i, j int) bool {
	if teams[i].Score != teams[j].Score {
		return teams[i].Score < teams[j].Score
	}
	if len(teams[i].Players) != len(teams[j].Players) {
		return len(teams[i].Players) < len(teams[j].Players)
	}
	return utils.RNG.Intn(2) == 0
}

func (t *Team) Add(c *Client) {
	t.Players[c] = struct{}{}
	c.Team = t
}

func (t *Team) Remove(c *Client) {
	c.Team = NoTeam
	delete(t.Players, c)
}
