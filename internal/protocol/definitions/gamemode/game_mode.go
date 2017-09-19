package gamemode

type GameMode int32

const (
	FFA GameMode = iota
	CoopEdit
	Teamplay
	Insta
	InstaTeam
	Effic
	EfficTeam
	Tactics
	TacticsTeam
	Capture
	RegenCapture
	CTF
	InstaCTF
	Protect
	InstaProtect
	Hold
	InstaHold
	EfficCTF
	EfficProtect
	EfficHold
	Collect
	InstaCollect
	EfficCollect
)
