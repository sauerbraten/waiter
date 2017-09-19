package client

import (
	"github.com/sauerbraten/waiter/internal/client/playerstate"
	"github.com/sauerbraten/waiter/internal/protocol/definitions/armour"
	"github.com/sauerbraten/waiter/internal/protocol/definitions/gamemode"
	"github.com/sauerbraten/waiter/internal/protocol/definitions/weapon"
	"github.com/sauerbraten/waiter/internal/protocol/packet"
)

// The game state of a client.
type GameState struct {
	// position of player
	Position *packet.Packet

	// fields that change at spawn
	State          uint
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
	LastShot     int32
	LastDeath    int32

	// fields that change at intermission
	Frags      int32
	Deaths     int32
	Teamkills  int32
	ShotDamage int32
	Damage     int32
	Flags      int32
}

func newGameState() *GameState {
	return &GameState{
		Position: packet.New(),
	}
}

func (gs *GameState) ToWire() *packet.Packet {
	return packet.New(
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

	switch mode {
	case gamemode.Effic, gamemode.EfficTeam, gamemode.EfficCTF, gamemode.EfficProtect, gamemode.EfficHold, gamemode.EfficCollect:
		gs.Health = 100
		gs.MaxHealth = 100
		gs.Armour = 100
		gs.ArmourType = armour.Green
		gs.SelectedWeapon = weapon.Minigun
		gs.Ammo = weapon.SpawnAmmo[gamemode.Effic]

	case gamemode.Insta, gamemode.InstaTeam, gamemode.InstaCTF, gamemode.InstaProtect, gamemode.InstaHold, gamemode.InstaCollect:
		gs.Health = 1
		gs.MaxHealth = 1
		gs.Armour = 0
		gs.ArmourType = armour.Blue
		gs.SelectedWeapon = weapon.Rifle
		gs.Ammo = weapon.SpawnAmmo[gamemode.Insta]

	case gamemode.CTF, gamemode.Protect, gamemode.Hold, gamemode.Collect:
		gs.Health = 100
		gs.Armour = 50
		gs.ArmourType = armour.Blue
		gs.Ammo = weapon.SpawnAmmo[gamemode.FFA]

	case gamemode.FFA, gamemode.Teamplay, gamemode.RegenCapture:
		gs.Armour = 25
		gs.ArmourType = armour.Blue
		gs.SelectedWeapon = weapon.Pistol
		gs.Ammo = weapon.SpawnAmmo[gamemode.FFA]

		// TODO: tactics, tactics team and capture (random weapons) still missing
	}

}

func (gs *GameState) SelectWeapon(selectedWeapon weapon.Weapon) {
	if gs.State != playerstate.Alive {
		return
	}

	if selectedWeapon >= weapon.Saw && selectedWeapon <= weapon.Pistol {
		gs.SelectedWeapon = selectedWeapon
	} else {
		gs.SelectedWeapon = weapon.Pistol
	}
}

// Resets a client's game state.
func (gs *GameState) Reset() {
	gs.Position.Clear()

	if gs.State != playerstate.Spectator {
		gs.State = playerstate.Dead
	}
	gs.MaxHealth = 0
	gs.Tokens = 0

	gs.LastSpawn = 0
	gs.LifeSequence = 0
	gs.LastShot = 0
	gs.LastDeath = 0

	gs.Frags = 0
	gs.Deaths = 0
	gs.Teamkills = 0
	gs.ShotDamage = 0
	gs.Damage = 0
	gs.Flags = 0
}
