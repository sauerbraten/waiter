package main

type Team struct {
	Name    string
	Score   int
	Players []*Client
}

// sorts teams ascending by score
type ByScore []*Team

func (teams ByScore) Len() int           { return len(teams) }
func (teams ByScore) Swap(i, j int)      { teams[i], teams[j] = teams[j], teams[i] }
func (teams ByScore) Less(i, j int) bool { return teams[i].Score < teams[j].Score }

// sorts teams ascending by the number of players
type BySize []*Team

func (teams BySize) Len() int           { return len(teams) }
func (teams BySize) Swap(i, j int)      { teams[i], teams[j] = teams[j], teams[i] }
func (teams BySize) Less(i, j int) bool { return len(teams[i].Players) < len(teams[j].Players) }

func (t *Team) AddPlayer(c *Client) {
	for _, p := range t.Players {
		if p == c {
			return
		}
	}

	t.Players = append(t.Players, c)
}
