package weapon

import (
	"math/rand"

	"github.com/sauerbraten/waiter/internal/definitions/gamemode"
	"github.com/sauerbraten/waiter/internal/definitions/sound"
)

type ID int32

const (
	Saw ID = iota
	Shotgun
	Minigun
	RocketLauncher
	Rifle
	GrenadeLauncher
	Pistol
	numWeapons int32 = iota
)

var WeaponsWithAmmo = []ID{
	Shotgun,
	Minigun,
	RocketLauncher,
	Rifle,
	GrenadeLauncher,
	Pistol,
}

type Weapon struct {
	ID              ID
	Sound           sound.ID
	ReloadTime      int32
	Damage          int32
	Spread          int32
	ProjectileSpeed int32
	Recoil          int32
	Range           float64
	Rays            int32
	HitPush         int32
	ExplosionRadius float64
	TimeToLive      int32
	AmmoPickUpSize  int32
}

var byID = map[ID]Weapon{
	Saw:             Weapon{Saw, sound.Saw, 250, 50, 0, 0, 0, 14, 1, 80, 0.0, 0, 0},
	Shotgun:         Weapon{Shotgun, sound.Shotgun, 1400, 10, 400, 0, 20, 1024, 20, 80, 0.0, 0, 10},
	Minigun:         Weapon{Minigun, sound.Minigun, 100, 30, 100, 0, 7, 1024, 1, 80, 0.0, 0, 20},
	RocketLauncher:  Weapon{RocketLauncher, sound.RocketLaunch, 800, 120, 0, 320, 10, 1024, 1, 160, 40.0, 0, 5},
	Rifle:           Weapon{Rifle, sound.Rifle, 1500, 100, 0, 0, 30, 2048, 1, 80, 0.0, 0, 5},
	GrenadeLauncher: Weapon{GrenadeLauncher, sound.GrenadeLaunch, 600, 90, 0, 200, 10, 1024, 1, 250, 45.0, 1500, 10},
	Pistol:          Weapon{Pistol, sound.Pistol, 500, 35, 50, 0, 7, 1024, 1, 80, 0.0, 0, 30},
}

func ByID(id ID) Weapon {
	if id < Saw || id > Pistol {
		return byID[Pistol]
	}
	return byID[id]
}

func randomID() ID {
	return ID(rand.Int31n(numWeapons-1) + 1) // -1 +1 to exclude chainsaw (= 0)
}

func SpawnAmmo(mode gamemode.ID) (map[ID]int32, Weapon) {
	switch mode {
	case gamemode.Insta, gamemode.InstaTeam, gamemode.InstaCTF, gamemode.InstaProtect, gamemode.InstaHold, gamemode.InstaCollect:
		return map[ID]int32{
			Rifle: 100,
		}, byID[Rifle]

	case gamemode.Tactics, gamemode.TacticsTeam, gamemode.Capture:
		spawnWeapon1 := randomID()
		spawnWeapon2 := randomID()
		for spawnWeapon2 == spawnWeapon1 {
			spawnWeapon2 = randomID()
		}
		var factor int32 = 2
		if mode == gamemode.Capture {
			factor = 1
		}
		ammo := map[ID]int32{
			spawnWeapon1: byID[spawnWeapon1].AmmoPickUpSize * factor,
			spawnWeapon2: byID[spawnWeapon2].AmmoPickUpSize * factor,
		}
		if mode == gamemode.Capture {
			ammo[GrenadeLauncher]++
		}
		return ammo, byID[spawnWeapon1]

	case gamemode.Effic, gamemode.EfficTeam, gamemode.EfficCTF, gamemode.EfficProtect, gamemode.EfficHold, gamemode.EfficCollect:
		// two of each weapons ammo pick up size, except just one for minigun
		return map[ID]int32{
			Shotgun:         20,
			Minigun:         20,
			RocketLauncher:  10,
			Rifle:           10,
			GrenadeLauncher: 20,
		}, byID[Minigun]

	case gamemode.FFA, gamemode.Teamplay, gamemode.RegenCapture, gamemode.CTF, gamemode.Protect, gamemode.Hold, gamemode.Collect:
		return map[ID]int32{
			Shotgun:         0,
			Minigun:         0,
			RocketLauncher:  0,
			Rifle:           0,
			GrenadeLauncher: 1,
			Pistol:          40,
		}, byID[Pistol]

	default:
		println("unhandled gamemode in SpawnAmmo:", mode)
		panic("fix this!")
	}
}

// Flattens m into a slice by putting values in the order specified by keys.
func FlattenAmmo(m map[ID]int32) (values []int32) {
	values = make([]int32, len(WeaponsWithAmmo))

	for index, id := range WeaponsWithAmmo {
		values[index] = m[id]
	}

	return
}

const (
	ExplosionDistanceScale   = 1.5
	ExplosionSelfDamageScale = 0.5
)
