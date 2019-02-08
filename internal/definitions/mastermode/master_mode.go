package mastermode

type ID int32

const (
	Auth ID = iota - 1
	Open
	Veto
	Locked
	Private
)
