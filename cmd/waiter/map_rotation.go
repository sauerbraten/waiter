package main

import (
	"github.com/sauerbraten/waiter/pkg/definitions/gamemode"
	"github.com/sauerbraten/waiter/internal/utils"
)

type MapRotation struct {
	Deathmatch []string `json:"deathmatch"`
	CTF        []string `json:"ctf"`
	Capture    []string `json:"capture"`

	queue []string
}

func (mr *MapRotation) NextMap(mode GameMode, currentMap string) string {
	if mode.ID() == s.GameMode.ID() {
		if len(mr.queue) > 0 {
			mapp := mr.queue[0]
			mr.queue = mr.queue[1:]
			return mapp
		}
		mr.queue = mr.queue[:0]
	}

	nextMap := func(pool []string) string {
		for i, m := range pool {
			if m == currentMap {
				return pool[(i+1)%len(pool)]
			}
		}

		// current map wasn't found in map rotation, return random map in rotation
		return pool[utils.RNG.Intn(len(pool))]
	}

	if gamemode.IsCTF(mode.ID()) {
		return nextMap(mr.CTF)
	} else if gamemode.IsCapture(mode.ID()) {
		return nextMap(mr.Capture)
	} else {
		return nextMap(mr.Deathmatch)
	}
}

func (mr *MapRotation) InPool(mode GameMode, mapp string) bool {
	inPool := func(pool []string) bool {
		for _, m := range pool {
			if m == mapp {
				return true
			}
		}
		return false
	}

	if gamemode.IsCTF(mode.ID()) {
		return inPool(mr.CTF)
	} else if gamemode.IsCapture(mode.ID()) {
		return inPool(mr.Capture)
	} else {
		return inPool(mr.Deathmatch)
	}
}

func (mr *MapRotation) inQueue(mapp string) bool {
	for _, m := range mr.queue {
		if m == mapp {
			return true
		}
	}
	return false
}

func (mr *MapRotation) queueMap(mapp string) (err string) {
	if mr.inQueue(mapp) {
		return mapp + " is already queued!"
	}
	if !mr.InPool(s.GameMode, mapp) {
		return mapp + " is not in the map pool for " + s.GameMode.ID().String() + "!"
	}
	mr.queue = append(mr.queue, mapp)
	return ""
}
