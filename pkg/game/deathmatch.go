package game

import (
	"github.com/sauerbraten/waiter/pkg/protocol/armour"
	"github.com/sauerbraten/waiter/pkg/protocol/gamemode"
	"github.com/sauerbraten/waiter/pkg/protocol/weapon"
)

// prints intermission stats based on frags
type deathmatchMode struct {
	timedMode
}

func newDeathmatchMode(s Server) deathmatchMode {
	return deathmatchMode{
		timedMode: newTimedMode(s),
	}
}

func (dm *deathmatchMode) End() {
	dm.timedMode.End()
	// todo: print some stats
}

type teamDeathmatchMode struct {
	teamMode
	deathmatchMode
}

func newTeamDeathmatchMode(s Server, keepTeams bool) teamDeathmatchMode {
	return teamDeathmatchMode{
		teamMode:       newTeamMode(s, true, keepTeams, "good", "evil"),
		deathmatchMode: newDeathmatchMode(s),
	}
}

type efficMode struct{}

func (*efficMode) Spawn(p *Player) {
	p.ArmourType, p.Armour = armour.Green, 100
	p.Ammo, p.SelectedWeapon = weapon.SpawnAmmoEffic()
	p.Health = 100
}

type Effic struct {
	casualMode
	deathmatchMode
	efficMode
	noSpawnWaitMode
	noItemsMode
	teamlessMode
}

func NewEffic(s Server) *Effic {
	var effic *Effic
	effic = &Effic{
		deathmatchMode: newDeathmatchMode(s),
		teamlessMode:   newTeamlessMode(s),
	}
	return effic
}

func (*Effic) ID() gamemode.ID { return gamemode.Effic }

type EfficTeam struct {
	casualMode
	teamDeathmatchMode
	efficMode
	noSpawnWaitMode
	noItemsMode
}

// assert interface implementations at compile time
var (
	_ Mode      = &EfficTeam{}
	_ TeamMode  = &EfficTeam{}
	_ TimedMode = &EfficTeam{}
)

func NewEfficTeam(s Server, keepTeams bool) *EfficTeam {
	var efficTeam *EfficTeam
	efficTeam = &EfficTeam{
		teamDeathmatchMode: newTeamDeathmatchMode(s, keepTeams),
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
	casualMode
	deathmatchMode
	instaMode
	noSpawnWaitMode
	noItemsMode
	teamlessMode
}

// assert interface implementations at compile time
var (
	_ Mode      = &Insta{}
	_ TimedMode = &Insta{}
)

func NewInsta(s Server) *Insta {
	var insta *Insta
	insta = &Insta{
		deathmatchMode: newDeathmatchMode(s),
		teamlessMode:   newTeamlessMode(s),
	}
	return insta
}

func (*Insta) ID() gamemode.ID { return gamemode.Insta }

type InstaTeam struct {
	casualMode
	teamDeathmatchMode
	instaMode
	noSpawnWaitMode
	noItemsMode
}

// assert interface implementations at compile time
var (
	_ Mode      = &InstaTeam{}
	_ TeamMode  = &InstaTeam{}
	_ TimedMode = &InstaTeam{}
)

func NewInstaTeam(s Server, keepTeams bool) *InstaTeam {
	var instaTeam *InstaTeam
	instaTeam = &InstaTeam{
		teamDeathmatchMode: newTeamDeathmatchMode(s, keepTeams),
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
	casualMode
	deathmatchMode
	tacticsMode
	noSpawnWaitMode
	noItemsMode
	teamlessMode
}

// assert interface implementations at compile time
var (
	_ Mode      = &Tactics{}
	_ TimedMode = &Tactics{}
)

func NewTactics(s Server) *Tactics {
	var tactics *Tactics
	tactics = &Tactics{
		deathmatchMode: newDeathmatchMode(s),
		teamlessMode:   newTeamlessMode(s),
	}
	return tactics
}

func (*Tactics) ID() gamemode.ID { return gamemode.Tactics }

type TacticsTeam struct {
	casualMode
	teamDeathmatchMode
	tacticsMode
	noSpawnWaitMode
	noItemsMode
}

// assert interface implementations at compile time
var (
	_ Mode      = &TacticsTeam{}
	_ TeamMode  = &TacticsTeam{}
	_ TimedMode = &TacticsTeam{}
)

func NewTacticsTeam(s Server, keepTeams bool) *TacticsTeam {
	var tacticsTeam *TacticsTeam
	tacticsTeam = &TacticsTeam{
		teamDeathmatchMode: newTeamDeathmatchMode(s, keepTeams),
	}
	return tacticsTeam
}

func (*TacticsTeam) ID() gamemode.ID { return gamemode.TacticsTeam }
