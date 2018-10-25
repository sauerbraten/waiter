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
	numWeapons
)

func Random() ID {
	return ID(rand.Int31n(int32(numWeapons)))
}

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
	Sound           sound.Sound
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
}

var ByID = map[ID]Weapon{
	Saw:             Weapon{Saw, sound.Saw, 250, 50, 0, 0, 0, 14, 1, 80, 0.0, 0},
	Shotgun:         Weapon{Shotgun, sound.Shotgun, 1400, 10, 400, 0, 20, 1024, 20, 80, 0.0, 0},
	Minigun:         Weapon{Minigun, sound.Minigun, 100, 30, 100, 0, 7, 1024, 1, 80, 0.0, 0},
	RocketLauncher:  Weapon{RocketLauncher, sound.RocketLaunch, 800, 120, 0, 320, 10, 1024, 1, 160, 40.0, 0},
	Rifle:           Weapon{Rifle, sound.Rifle, 1500, 100, 0, 0, 30, 2048, 1, 80, 0.0, 0},
	GrenadeLauncher: Weapon{GrenadeLauncher, sound.GrenadeLaunch, 600, 90, 0, 200, 10, 1024, 1, 250, 45.0, 1500},
	Pistol:          Weapon{Pistol, sound.Pistol, 500, 35, 50, 0, 7, 1024, 1, 80, 0.0, 0},
}

func SpawnAmmo(mode gamemode.ID) map[ID]int32 {
	switch mode {
	case gamemode.Insta, gamemode.InstaTeam, gamemode.InstaCTF, gamemode.InstaProtect, gamemode.InstaHold, gamemode.InstaCollect:
		return map[ID]int32{
			Shotgun:         0,
			Minigun:         0,
			RocketLauncher:  0,
			Rifle:           100,
			GrenadeLauncher: 0,
			Pistol:          0,
		}

	case gamemode.Tactics, gamemode.TacticsTeam:
		ammo := map[ID]int32{
			Pistol:          40,
			GrenadeLauncher: 1,
			// TODO
		}
		return ammo

	case gamemode.Effic, gamemode.EfficTeam, gamemode.EfficCTF, gamemode.EfficProtect, gamemode.EfficHold, gamemode.EfficCollect:
		return map[ID]int32{
			Shotgun:         20,
			Minigun:         20,
			RocketLauncher:  10,
			Rifle:           10,
			GrenadeLauncher: 20,
			Pistol:          0,
		}

	case gamemode.FFA, gamemode.Teamplay, gamemode.Capture, gamemode.RegenCapture, gamemode.CTF, gamemode.Protect, gamemode.Hold, gamemode.Collect:
		return map[ID]int32{
			Shotgun:         0,
			Minigun:         0,
			RocketLauncher:  0,
			Rifle:           0,
			GrenadeLauncher: 1,
			Pistol:          40,
		}

	default:
		println("unhandled gamemode in SpawnAmmo:", mode)
		panic("fix this!")
	}
}

// Flattens m into a slice by putting values in the order specified by keys.
func FlattenAmmo(m map[ID]int32) (values []int32) {
	values = make([]int32, len(m))

	for index, id := range WeaponsWithAmmo {
		if ammo, ok := m[id]; ok {
			values[index] = ammo
		}
	}

	return
}

const (
	ExplosionDistanceScale   = 1.5
	ExplosionSelfDamageScale = 0.5
)
