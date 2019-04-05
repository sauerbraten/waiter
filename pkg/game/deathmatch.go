package game

import (
	"github.com/sauerbraten/waiter/pkg/protocol/armour"
	"github.com/sauerbraten/waiter/pkg/protocol/gamemode"
	"github.com/sauerbraten/waiter/pkg/protocol/weapon"
)

// prints intermission stats based on frags
type deathmatchMode struct {
	Timed
}

func newDeathmatchMode(t Timed) deathmatchMode {
	return deathmatchMode{
		Timed: t,
	}
}

func (dm *deathmatchMode) End() {
	dm.Timed.End()
	// todo: print some stats
}

type teamDeathmatchMode struct {
	teamed
	deathmatchMode
}

func newTeamDeathmatchMode(s Server, keepTeams bool, t Timed) teamDeathmatchMode {
	return teamDeathmatchMode{
		teamed:         newTeamed(s, true, keepTeams, "good", "evil"),
		deathmatchMode: newDeathmatchMode(t),
	}
}

type efficMode struct{}

func (*efficMode) Spawn(p *Player) {
	p.ArmourType, p.Armour = armour.Green, 100
	p.Ammo, p.SelectedWeapon = weapon.SpawnAmmoEffic()
	p.Health = 100
}

type Effic struct {
	deathmatchMode
	efficMode
	noSpawnWaitMode
	noItemsMode
	teamlessMode
}

func NewEffic(s Server, t Timed) *Effic {
	var effic *Effic
	effic = &Effic{
		deathmatchMode: newDeathmatchMode(t),
		teamlessMode:   newTeamlessMode(s),
	}
	return effic
}

func (*Effic) ID() gamemode.ID { return gamemode.Effic }

type EfficTeam struct {
	teamDeathmatchMode
	efficMode
	noSpawnWaitMode
	noItemsMode
}

// assert interface implementations at compile time
var (
	_ Mode   = &EfficTeam{}
	_ Teamed = &EfficTeam{}
	_ Timed  = &EfficTeam{}
)

func NewEfficTeam(s Server, keepTeams bool, t Timed) *EfficTeam {
	var efficTeam *EfficTeam
	efficTeam = &EfficTeam{
		teamDeathmatchMode: newTeamDeathmatchMode(s, keepTeams, t),
	}
	return efficTeam
}

func (*EfficTeam) ID() gamemode.ID { return gamemode.EfficTeam }

type instaMode struct{}

func (*instaMode) Spawn(p *Player) {
	p.ArmourType, p.Armour = armour.None, 0
	p.Ammo, p.SelectedWeapon = weapon.SpawnAmmoInsta()
	p.Health, p.MaxHealth = 1, 1
}

type Insta struct {
	deathmatchMode
	instaMode
	noSpawnWaitMode
	noItemsMode
	teamlessMode
}

// assert interface implementations at compile time
var (
	_ Mode  = &Insta{}
	_ Timed = &Insta{}
)

func NewInsta(s Server, t Timed) *Insta {
	var insta *Insta
	insta = &Insta{
		deathmatchMode: newDeathmatchMode(t),
		teamlessMode:   newTeamlessMode(s),
	}
	return insta
}

func (*Insta) ID() gamemode.ID { return gamemode.Insta }

type InstaTeam struct {
	teamDeathmatchMode
	instaMode
	noSpawnWaitMode
	noItemsMode
}

// assert interface implementations at compile time
var (
	_ Mode   = &InstaTeam{}
	_ Teamed = &InstaTeam{}
	_ Timed  = &InstaTeam{}
)

func NewInstaTeam(s Server, keepTeams bool, t Timed) *InstaTeam {
	var instaTeam *InstaTeam
	instaTeam = &InstaTeam{
		teamDeathmatchMode: newTeamDeathmatchMode(s, keepTeams, t),
	}
	return instaTeam
}

func (*InstaTeam) ID() gamemode.ID { return gamemode.InstaTeam }

type tacticsMode struct{}

func (*tacticsMode) Spawn(p *Player) {
	p.ArmourType, p.Armour = armour.Green, 100
	p.Ammo, p.SelectedWeapon = weapon.SpawnAmmoTactics()
	p.Health, p.MaxHealth = 1, 1
}

type Tactics struct {
	deathmatchMode
	tacticsMode
	noSpawnWaitMode
	noItemsMode
	teamlessMode
}

// assert interface implementations at compile time
var (
	_ Mode  = &Tactics{}
	_ Timed = &Tactics{}
)

func NewTactics(s Server, t Timed) *Tactics {
	var tactics *Tactics
	tactics = &Tactics{
		deathmatchMode: newDeathmatchMode(t),
		teamlessMode:   newTeamlessMode(s),
	}
	return tactics
}

func (*Tactics) ID() gamemode.ID { return gamemode.Tactics }

type TacticsTeam struct {
	teamDeathmatchMode
	tacticsMode
	noSpawnWaitMode
	noItemsMode
}

// assert interface implementations at compile time
var (
	_ Mode   = &TacticsTeam{}
	_ Teamed = &TacticsTeam{}
	_ Timed  = &TacticsTeam{}
)

func NewTacticsTeam(s Server, keepTeams bool, t Timed) *TacticsTeam {
	var tacticsTeam *TacticsTeam
	tacticsTeam = &TacticsTeam{
		teamDeathmatchMode: newTeamDeathmatchMode(s, keepTeams, t),
	}
	return tacticsTeam
}

func (*TacticsTeam) ID() gamemode.ID { return gamemode.TacticsTeam }
