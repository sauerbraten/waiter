package pickup

import (
	"github.com/sauerbraten/waiter/pkg/protocol/entity"
	"github.com/sauerbraten/waiter/pkg/protocol/sound"
)

type PickUp struct {
	Sound  sound.ID
	Amount int32
}

var PickUps map[int32]PickUp = map[int32]PickUp{
	entity.PU_SHOTGUN:         PickUp{sound.PickUpAmmo, 10},
	entity.PU_MINIGUN:         PickUp{sound.PickUpAmmo, 20},
	entity.PU_ROCKETLAUNCHER:  PickUp{sound.PickUpAmmo, 5},
	entity.PU_RIFLE:           PickUp{sound.PickUpAmmo, 5},
	entity.PU_GRENADELAUNCHER: PickUp{sound.PickUpAmmo, 10},
	entity.PU_PISTOL:          PickUp{sound.PickUpAmmo, 30},
	entity.PU_HEALTH:          PickUp{sound.PickUpHealth, 25},
	entity.PU_BOOST:           PickUp{sound.PickUpHealth, 10},
	entity.PU_GREENARMOUR:     PickUp{sound.PickUpArmour, 100},
	entity.PU_YELLOWARMOUR:    PickUp{sound.PickUpArmour, 200},
	entity.PU_QUAD:            PickUp{sound.PickUpQuaddamage, 20000},
}
