package main

import (
	"time"

	"github.com/sauerbraten/waiter/internal/client/playerstate"
	"github.com/sauerbraten/waiter/internal/definitions/armour"
	"github.com/sauerbraten/waiter/internal/definitions/gamemode"
	"github.com/sauerbraten/waiter/internal/definitions/weapon"
	"github.com/sauerbraten/waiter/internal/protocol/packet"
)

// The game state of a client.
type GameState struct {
	// fields that change at spawn
	State          uint32
	Health         int32
	MaxHealth      int32
	Armour         int32
	ArmourType     armour.Armour
	QuadTimeLeft   int32 // in milliseconds
	SelectedWeapon weapon.Weapon
	GunReloadTime  int32
	Ammo           map[weapon.Weapon]int32 // weapon → ammo
	Tokens         int32                   // skulls

	LastSpawn    int32
	LifeSequence int32
	LastShot     time.Time
	LastDeath    time.Time

	// fields that change at intermission
	Frags      int32
	Deaths     int32
	Teamkills  int32
	ShotDamage int32
	Damage     int32
	Flags      int32
}

func NewGameState() *GameState {
	gs := &GameState{}
	gs.Reset()
	return gs
}

func (gs *GameState) ToWire() []byte {
	return packet.Encode(
		gs.LifeSequence, gs.Health, gs.MaxHealth, gs.Armour, gs.ArmourType, gs.SelectedWeapon,
		weapon.FlattenAmmo(gs.Ammo, weapon.WeaponsWithAmmo),
	)
}

// Sets GameState properties to the initial values depending on the mode.
func (gs *GameState) Spawn(mode gamemode.GameMode) {
	gs.QuadTimeLeft = 0
	gs.GunReloadTime = 0
	gs.State = playerstate.Alive
	gs.Tokens = 0
	gs.ArmourType, gs.Armour = armour.SpawnArmour(mode)
	gs.Ammo = weapon.SpawnAmmo(mode)

	gs.Health = gs.MaxHealth

	switch mode {
	case gamemode.Insta, gamemode.InstaTeam, gamemode.InstaCTF, gamemode.InstaProtect, gamemode.InstaHold, gamemode.InstaCollect:
		gs.Health, gs.MaxHealth = 1, 1
		gs.SelectedWeapon = weapon.Rifle

	case gamemode.RegenCapture:
		gs.SelectedWeapon = weapon.Random()

	case gamemode.Effic, gamemode.EfficTeam, gamemode.EfficCTF, gamemode.EfficProtect, gamemode.EfficHold, gamemode.EfficCollect:
		gs.SelectedWeapon = weapon.Minigun

	case gamemode.FFA, gamemode.Teamplay, gamemode.Capture, gamemode.CTF, gamemode.Protect, gamemode.Hold, gamemode.Collect:
		gs.SelectedWeapon = weapon.Pistol

	default:
		println("unhandled gamemode:", mode)
		panic("fix this!")
	}
}

func (gs *GameState) SelectWeapon(selectedWeapon weapon.Weapon) (weapon.Weapon, bool) {
	if gs.State != playerstate.Alive {
		return selectedWeapon, false
	}

	if selectedWeapon >= weapon.Saw && selectedWeapon <= weapon.Pistol {
		gs.SelectedWeapon = selectedWeapon
	} else {
		gs.SelectedWeapon = weapon.Pistol
	}

	return gs.SelectedWeapon, true
}

// Resets a client's game state.
func (gs *GameState) Reset() {
	if gs.State != playerstate.Spectator {
		gs.State = playerstate.Dead
	}
	gs.MaxHealth = 100
	gs.Tokens = 0

	gs.LastSpawn = 0
	gs.LifeSequence = 0
	gs.LastShot = time.Time{}
	gs.LastDeath = time.Time{}

	gs.Frags = 0
	gs.Deaths = 0
	gs.Teamkills = 0
	gs.ShotDamage = 0
	gs.Damage = 0
	gs.Flags = 0
}
