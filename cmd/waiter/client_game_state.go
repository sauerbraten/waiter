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
	GunReloadEnd   time.Time
	Ammo           map[weapon.ID]int32 // weapon → ammo
	Tokens         int32               // skulls

	LastSpawn    time.Time
	LifeSequence int32
	LastShot     time.Time
	LastDeath    time.Time

	// fields that change at intermission
	Frags      int
	Deaths     int
	Teamkills  int
	ShotDamage int32
	Damage     int32
	Flags      int
}

func NewGameState() *GameState {
	gs := &GameState{}
	gs.Reset()
	return gs
}

func (gs *GameState) ToWire() []byte {
	return packet.Encode(
		gs.LifeSequence, gs.Health, gs.MaxHealth, gs.Armour, gs.ArmourType, gs.SelectedWeapon.ID,
		weapon.FlattenAmmo(gs.Ammo),
	)
}

// Sets GameState properties to the initial values depending on the mode.
func (gs *GameState) Spawn(mode gamemode.GameMode) {
	gs.QuadTimeLeft = 0
	gs.GunReloadEnd = time.Time{}
	gs.Tokens = 0
	gs.ArmourType, gs.Armour = armour.SpawnArmour(mode)
	gs.Ammo = weapon.SpawnAmmo(mode)
	gs.LastSpawn = time.Now()
	gs.LifeSequence = (gs.LifeSequence + 1) % 128

	gs.Health = gs.MaxHealth

	switch mode {
	case gamemode.Insta, gamemode.InstaTeam, gamemode.InstaCTF, gamemode.InstaProtect, gamemode.InstaHold, gamemode.InstaCollect:
		gs.Health, gs.MaxHealth = 1, 1
		gs.SelectedWeapon = weapon.ByID[weapon.Rifle]

	case gamemode.RegenCapture:
		gs.SelectedWeapon = weapon.ByID[weapon.Random()]

	case gamemode.Effic, gamemode.EfficTeam, gamemode.EfficCTF, gamemode.EfficProtect, gamemode.EfficHold, gamemode.EfficCollect:
		gs.SelectedWeapon = weapon.ByID[weapon.Minigun]

	case gamemode.FFA, gamemode.Teamplay, gamemode.Capture, gamemode.CTF, gamemode.Protect, gamemode.Hold, gamemode.Collect:
		gs.SelectedWeapon = weapon.ByID[weapon.Pistol]

	default:
		println("unhandled gamemode:", mode)
		panic("fix this!")
	}
}

func (gs *GameState) SelectWeapon(id weapon.ID) (weapon.Weapon, bool) {
	if gs.State != playerstate.Alive {
		return weapon.ByID[weapon.Pistol], false
	}

	wpn, ok := weapon.ByID[id]
	if ok {
		gs.SelectedWeapon = wpn
	} else {
		gs.SelectedWeapon = weapon.ByID[weapon.Pistol]
	}

	return gs.SelectedWeapon, true
}

func (gs *GameState) applyDamage(damage int32) {
	// TODO: account for armour
	damageToArmour := damage * armour.Absorption(gs.ArmourType) / 100
	if damageToArmour > gs.Armour {
		damageToArmour = gs.Armour
	}
	gs.Armour -= damageToArmour
	damage -= damageToArmour
	gs.Health -= damage
}

// Resets a client's game state.
func (gs *GameState) Reset() {
	if gs.State != playerstate.Spectator {
		gs.State = playerstate.Dead
	}
	gs.MaxHealth = 100

	gs.LifeSequence = 0
	gs.LastDeath = time.Time{}

	gs.Frags = 0
	gs.Deaths = 0
	gs.Teamkills = 0
	gs.ShotDamage = 0
	gs.Damage = 0
	gs.Flags = 0

	gs.Respawn()
}

func (gs *GameState) Respawn() {
	gs.LastSpawn = time.Time{}
	gs.LastShot = time.Time{}
	gs.Tokens = 0
}
