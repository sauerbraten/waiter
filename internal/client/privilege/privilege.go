package privilege

type Privilege int32

const (
	None Privilege = iota
	Master
	Auth
	Admin
)
