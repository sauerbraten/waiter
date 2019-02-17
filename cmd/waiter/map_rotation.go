package main

import (
	"github.com/sauerbraten/waiter/internal/utils"
)

type MapRotation struct {
	Deathmatch []string `json:"deathmatch"`
	CTF        []string `json:"ctf"`
	Capture    []string `json:"capture"`
}

func (mr *MapRotation) NextMap(mode GameMode, currentMap string) string {
	nextMap := func(pool []string) string {
		for i, m := range pool {
			if m == currentMap {
				return pool[(i+1)%len(pool)]
			}
		}

		// current map wasn't found in map rotation, return random map in rotation
		return pool[utils.RNG.Intn(len(pool))]
	}

	switch mode.(type) {
	case CTFMode:
		return nextMap(mr.CTF)
	case CaptureMode:
		return nextMap(mr.Capture)
	default:
		return nextMap(mr.Deathmatch)
	}
}
