package maprotation

import "github.com/sauerbraten/waiter/internal/utils"

// temporary set of maps used in development phase
var mr = []string{
	"hashi",
	"ot",
	"turbine",
	"shiva",
	"complex",
}

func NextMap(currentMap string) string {
	for i, m := range mr {
		if m == currentMap {
			return mr[(i+1)%len(mr)]
		}
	}

	// current map wasn't found in map rotation, return random map in rotation
	return mr[utils.RNG.Intn(len(mr))]
}
