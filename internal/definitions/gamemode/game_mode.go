package gamemode

import "strconv"

type ID int32

const (
	FFA ID = iota
	CoopEdit
	Teamplay
	Insta
	InstaTeam
	Effic // 5
	EfficTeam
	Tactics
	TacticsTeam
	Capture
	RegenCapture // 10
	CTF
	InstaCTF
	Protect
	InstaProtect
	Hold // 15
	InstaHold
	EfficCTF
	EfficProtect
	EfficHold
	Collect // 20
	InstaCollect
	EfficCollect
)

func (gm ID) String() string {
	switch gm {
	case FFA:
		return "ffa"
	case CoopEdit:
		return "coop edit"
	case Teamplay:
		return "teamplay"
	case Insta:
		return "insta"
	case InstaTeam:
		return "insta team"
	case Effic:
		return "effic"
	case EfficTeam:
		return "effic team"
	case Tactics:
		return "tactics"
	case TacticsTeam:
		return "tactics team"
	case Capture:
		return "capture"
	case RegenCapture:
		return "regen capture"
	case CTF:
		return "ctf"
	case InstaCTF:
		return "insta ctf"
	case Protect:
		return "protect"
	case InstaProtect:
		return "insta protect"
	case Hold:
		return "hold"
	case InstaHold:
		return "insta hold"
	case EfficCTF:
		return "effic ctf"
	case EfficProtect:
		return "effic protect"
	case EfficHold:
		return "effic hold"
	case Collect:
		return "collect"
	case InstaCollect:
		return "insta collect"
	case EfficCollect:
		return "effic collect"
	default:
		return strconv.Itoa(int(gm))
	}
}

func Valid(gm ID) bool {
	switch gm {
	case Insta, InstaTeam,
		Effic, EfficTeam,
		Tactics, TacticsTeam,
		InstaCTF, EfficCTF:
		return true
	default:
		return false
	}
}
