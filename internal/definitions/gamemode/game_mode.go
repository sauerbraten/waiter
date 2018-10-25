package gamemode

type ID int32

const (
	FFA ID = iota
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
