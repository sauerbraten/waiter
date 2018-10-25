package armour

import "github.com/sauerbraten/waiter/internal/definitions/gamemode"

type Armour int32

const (
	Blue Armour = iota
	Green
	Yellow

	None = -1
)

func Absorption(typ Armour) int32 {
	switch typ {
	case Blue:
		return 25
	case Green:
		return 50
	case Yellow:
		return 75
	default:
		return 0
	}
}

func SpawnArmour(mode gamemode.ID) (Armour, int32) {
	switch mode {
	case gamemode.Insta, gamemode.InstaTeam, gamemode.InstaCTF, gamemode.InstaProtect, gamemode.InstaHold, gamemode.InstaCollect:
		return None, 0
	case gamemode.Tactics, gamemode.TacticsTeam, gamemode.Effic, gamemode.EfficTeam, gamemode.EfficCTF, gamemode.EfficProtect, gamemode.EfficHold, gamemode.EfficCollect:
		return Green, 100
	case gamemode.CTF, gamemode.Protect, gamemode.Hold, gamemode.Collect:
		return Blue, 50
	case gamemode.Capture, gamemode.RegenCapture, gamemode.FFA, gamemode.Teamplay:
		return Blue, 25
	default:
		println("unhandled gamemode:", mode)
		panic("fix this!")
	}
}
