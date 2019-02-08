package privilege

import "strings"

type ID int32

const (
	None ID = iota
	Master
	Auth
	Admin
)

func Parse(s string) ID {
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

func (p ID) String() string {
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
