package mastermode

type MasterMode int32

const (
	Auth MasterMode = iota - 1
	Open
	Veto
	Locked
	Private
)
