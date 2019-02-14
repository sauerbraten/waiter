package maprotation

import (
	"github.com/sauerbraten/waiter/internal/definitions/gamemode"
	"github.com/sauerbraten/waiter/internal/utils"
)

// temporary set of maps used in development phase
var (
	dmMaps = []string{
		"hashi",
		"turbine",
		"ot",
		"memento",
		"kffa",
	}
	ctfMaps = []string{
		"reissen",
		"forge",
		"haste",
		"dust2",
		"redemption",
	}
	captureMaps = []string{
		"nmp8",
		"nmp9",
		"nmp4",
		"nevil_c",
		"serenity",
	}
	mr = map[gamemode.ID][]string{
		gamemode.Insta:       dmMaps,
		gamemode.InstaTeam:   dmMaps,
		gamemode.Effic:       dmMaps,
		gamemode.EfficTeam:   dmMaps,
		gamemode.Tactics:     dmMaps,
		gamemode.TacticsTeam: dmMaps,
		gamemode.InstaCTF:    ctfMaps,
		gamemode.EfficCTF:    ctfMaps,
		gamemode.Capture:     captureMaps,
	}
)

func NextMap(mode gamemode.ID, currentMap string) string {
	for i, m := range mr[mode] {
		if m == currentMap {
			return mr[mode][(i+1)%len(mr[mode])]
		}
	}

	// current map wasn't found in map rotation, return random map in rotation
	return mr[mode][utils.RNG.Intn(len(mr[mode]))]
}
