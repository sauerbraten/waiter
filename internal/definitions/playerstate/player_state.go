package playerstate

type ID uint32

const (
	Alive ID = iota
	Dead
	Spawning
	Lagged
	Editing
	Spectator
)
