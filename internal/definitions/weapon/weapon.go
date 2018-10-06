package weapon

import (
	"math/rand"

	"github.com/sauerbraten/waiter/internal/definitions/gamemode"
)

type Weapon int32

const (
	Saw Weapon = iota
	Shotgun
	Minigun
	RocketLauncher
	Rifle
	GrenadeLauncher
	Pistol
	numWeapons
)

func Random() Weapon {
	return Weapon(rand.Int31n(int32(numWeapons)))
}

var WeaponsWithAmmo []Weapon = []Weapon{
	Shotgun,
	Minigun,
	RocketLauncher,
	Rifle,
	GrenadeLauncher,
	Pistol,
}

/*
type Weapon struct {
	Sound           sound.SoundNum
	ReloadTime      int32
	Damage          int32
	Spread          int32
	ProjectileSpeed int32
	Recoil          int32
	Range           int32
	Rays            int32
	HitPush         int32
	ExplosionRadius int32
	TimeToLive      int32
}

var Weapons map[WeaponNum]Weapon = map[WeaponNum]Weapon{
	Saw:             Weapon{sound.Saw, 250, 50, 0, 0, 0, 14, 1, 80, 0, 0},
	Shotgun:         Weapon{sound.Shotgun, 1400, 10, 400, 0, 20, 1024, 20, 80, 0, 0},
	Minigun:         Weapon{sound.Minigun, 100, 30, 100, 0, 7, 1024, 1, 80, 0, 0},
	RocketLauncher:  Weapon{sound.RocketLaunch, 800, 120, 0, 320, 10, 1024, 1, 160, 40, 0},
	Rifle:           Weapon{sound.Rifle, 1500, 100, 0, 0, 30, 2048, 1, 80, 0, 0},
	GrenadeLauncher: Weapon{sound.GrenadeLaunch, 600, 90, 0, 200, 10, 1024, 1, 250, 45, 1500},
	Pistol:          Weapon{sound.Pistol, 500, 35, 50, 0, 7, 1024, 1, 80, 0, 0},
}
*/

func SpawnAmmo(mode gamemode.GameMode) map[Weapon]int32 {
	switch mode {
	case gamemode.Insta, gamemode.InstaTeam, gamemode.InstaCTF, gamemode.InstaProtect, gamemode.InstaHold, gamemode.InstaCollect:
		return map[Weapon]int32{
			Shotgun:         0,
			Minigun:         0,
			RocketLauncher:  0,
			Rifle:           100,
			GrenadeLauncher: 0,
			Pistol:          0,
		}

	case gamemode.Tactics, gamemode.TacticsTeam:
		ammo := map[Weapon]int32{
			Pistol:          40,
			GrenadeLauncher: 1,
			// TODO
		}
		return ammo

	case gamemode.Effic, gamemode.EfficTeam, gamemode.EfficCTF, gamemode.EfficProtect, gamemode.EfficHold, gamemode.EfficCollect:
		return map[Weapon]int32{
			Shotgun:         20,
			Minigun:         20,
			RocketLauncher:  10,
			Rifle:           10,
			GrenadeLauncher: 20,
			Pistol:          0,
		}

	case gamemode.FFA, gamemode.Teamplay, gamemode.Capture, gamemode.RegenCapture, gamemode.CTF, gamemode.Protect, gamemode.Hold, gamemode.Collect:
		return map[Weapon]int32{
			Shotgun:         0,
			Minigun:         0,
			RocketLauncher:  0,
			Rifle:           0,
			GrenadeLauncher: 1,
			Pistol:          40,
		}

	default:
		println("unhandled gamemode:", mode)
		panic("fix this!")
	}
}

// Flattens m into a slice by putting values in the order specified by keys.
func FlattenAmmo(m map[Weapon]int32, keys []Weapon) (values []int32) {
	values = make([]int32, len(keys))

	for index, key := range keys {
		values[index] = m[key]
	}

	return
}
