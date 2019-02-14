package nmc

type ID int32 // network message code

const (
	Join ID = iota // = CONNECT
	ServerInfo
	Welcome
	InitializeClient
	Position
	ChatMessage
	Sound
	Leave // = CDIS
	Shoot
	Explode
	Suicide // 10
	Died
	Damage
	HitPush
	ShotEffects
	ExplodeEffects
	TrySpawn
	SpawnState
	ConfirmSpawn
	ForceDeath
	ChangeWeapon // 20
	Taunt
	MapChange
	VoteMap
	TeamInfo
	ITEMSPAWN
	ITEMPICKUP
	ITEMACC
	Teleport
	JumpPad
	Ping // 30
	Pong
	ClientPing
	TimeLeft // = TIMEUP
	ForceIntermission
	ServerMessage
	ItemList
	Resume
	EDITMODE
	EDITENT
	EDITF // 40
	EDITT
	EDITM
	FLIP
	COPY
	PASTE
	ROTATE
	REPLACE
	DELCUBE
	REMIP
	NEWMAP // 50
	GETMAP
	SENDMAP
	CLIPBOARD
	EDITVAR
	MasterMode
	Kick
	ClearBans
	CurrentMaster
	Spectator
	SetMaster // 60
	SetTeam
	Bases
	BaseInfo
	BaseScore
	REPAMMO
	BASEREGEN
	ANNOUNCE
	ListDemos
	SendDemoList
	GetDemo // 70
	SendDemo
	DemoPlayback
	RecordDemo
	StopDemo
	ClearDemos
	TakeFlag
	ReturnFlag
	ResetFlag
	InvisibleFlag
	TryDropFlag // 80
	DropFlag
	ScoreFlag
	InitFlags
	TeamChatMessage
	Client
	AuthTry
	AuthKick
	AuthChallenge
	AuthAnswer
	REQAUTH // 90
	PauseGame
	GAMESPEED
	ADDBOT
	DELBOT
	INITAI
	FROMAI
	BOTLIMIT
	BOTBALANCE
	MapCRC
	CHECKMAPS   // 100
	ChangeName  // SWITCHNAME
	ChangeModel // SWITCHMODEL
	ChangeTeam  // SWITCHTEAM
	INITTOKENS
	TAKETOKEN
	EXPIRETOKENS
	DROPTOKENS
	DEPOSITTOKENS
	STEALTOKENS
	ServerCommand // 110
	DEMOPACKET
	//NUMMSG
)

// A list of NMCs which can only be sent by a server, never by a client.
var ServerOnlyNMCs = []ID{
	ServerInfo,
	InitializeClient,
	Welcome,
	MapChange,
	ServerMessage,
	Damage,
	HitPush,
	ShotEffects,
	ExplodeEffects,
	Died,
	SpawnState,
	ForceDeath,
	TeamInfo,
	ITEMACC,
	ITEMSPAWN,
	TimeLeft,
	Leave,
	CurrentMaster,
	Pong,
	Resume,
	BaseScore,
	BaseInfo,
	BASEREGEN,
	ANNOUNCE,
	SendDemoList,
	SendDemo,
	DemoPlayback,
	SENDMAP,
	DropFlag,
	ScoreFlag,
	ReturnFlag,
	ResetFlag,
	InvisibleFlag,
	Client,
	AuthChallenge,
	INITAI,
	EXPIRETOKENS,
	DROPTOKENS,
	STEALTOKENS,
	DEMOPACKET,
}
