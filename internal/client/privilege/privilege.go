package privilege

import "strings"

type Privilege int32

const (
	None Privilege = iota
	Master
	Auth
	Admin
)

func Parse(s string) Privilege {
	switch strings.ToLower(s) {
	case "none":
		return None
	case "master":
		return Master
	case "auth":
		return Auth
	case "admin":
		return Admin
	default:
		return -1
	}
}

func (p Privilege) String() string {
	switch p {
	case None:
		return "none"
	case Master:
		return "master"
	case Auth:
		return "auth"
	case Admin:
		return "admin"
	default:
		return ""
	}
}
