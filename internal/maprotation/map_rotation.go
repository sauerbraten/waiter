package maprotation

import (
	"github.com/sauerbraten/waiter/internal/definitions/gamemode"
	"github.com/sauerbraten/waiter/internal/utils"
)

// temporary set of maps used in development phase
var mr = map[gamemode.GameMode][]string{
	gamemode.Effic: []string{
		"hashi",
		"turbine",
		"ot",
		"memento",
		"kffa",
	},
	gamemode.EfficCTF: []string{
		"reissen",
		"forge",
		"haste",
		"dust2",
		"redemption",
	},
	gamemode.Capture: []string{
		"nmp8",
		"nmp9",
		"nmp4",
		"nevil_c",
		"serenity",
	},
}

func NextMap(mode gamemode.GameMode, currentMap string) string {
	for i, m := range mr[mode] {
		if m == currentMap {
			return mr[mode][(i+1)%len(mr)]
		}
	}

	// current map wasn't found in map rotation, return random map in rotation
	return mr[mode][utils.RNG.Intn(len(mr))]
}
