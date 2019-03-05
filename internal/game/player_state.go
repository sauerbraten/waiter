package game

import (
	"time"

	"github.com/sauerbraten/waiter/internal/net/packet"
	"github.com/sauerbraten/waiter/pkg/definitions/armour"
	"github.com/sauerbraten/waiter/pkg/definitions/playerstate"
	"github.com/sauerbraten/waiter/pkg/definitions/weapon"
)

type PlayerState struct {
	State playerstate.ID

	// fields that reset at spawn
	LastSpawnAttempt time.Time
	QuadTimeLeft     int32 // in milliseconds
	LastShot         time.Time
	GunReloadEnd     time.Time
	// reset at spawn to value depending on mode
	Health         int32
	Armour         int32
	ArmourType     armour.ID
	SelectedWeapon weapon.Weapon
	Ammo           map[weapon.ID]int32 // weapon → ammo

	// reset at map change
	LifeSequence    int32
	LastDeath       time.Time
	MaxHealth       int32
	Frags           int
	Deaths          int
	Teamkills       int
	DamagePotential int32
	Damage          int32
	Flags           int
}

func NewPlayerState() PlayerState {
	ps := PlayerState{}
	ps.Reset()
	return ps
}

func (ps *PlayerState) ToWire() []byte {
	return packet.Encode(
		ps.LifeSequence,
		ps.Health,
		ps.MaxHealth,
		ps.Armour,
		ps.ArmourType,
		ps.SelectedWeapon.ID,
		weapon.FlattenAmmo(ps.Ammo),
	)
}

func (ps *PlayerState) Spawn() {
	ps.LifeSequence = (ps.LifeSequence + 1) % 128

	ps.LastSpawnAttempt = time.Now()
	ps.QuadTimeLeft = 0
	ps.LastShot = time.Time{}
	ps.GunReloadEnd = time.Time{}
}

func (ps *PlayerState) SelectWeapon(id weapon.ID) (weapon.Weapon, bool) {
	if ps.State != playerstate.Alive {
		return weapon.ByID(weapon.Pistol), false
	}
	ps.SelectedWeapon = weapon.ByID(id)
	return ps.SelectedWeapon, true
}

func (ps *PlayerState) applyDamage(damage int32) {
	damageToArmour := damage * armour.Absorption(ps.ArmourType) / 100
	if damageToArmour > ps.Armour {
		damageToArmour = ps.Armour
	}
	ps.Armour -= damageToArmour
	damage -= damageToArmour
	ps.Health -= damage
}

func (ps *PlayerState) Die() {
	if ps.State != playerstate.Alive {
		return
	}
	ps.State = playerstate.Dead
	ps.Deaths++
	ps.LastDeath = time.Now()
}

// Resets a client's game state.
func (ps *PlayerState) Reset() {
	if ps.State != playerstate.Spectator {
		ps.State = playerstate.Dead
	}
	ps.MaxHealth = 100

	ps.LifeSequence = 0
	ps.LastDeath = time.Time{}

	ps.Frags = 0
	ps.Deaths = 0
	ps.Teamkills = 0
	ps.DamagePotential = 0
	ps.Damage = 0
	ps.Flags = 0
}
