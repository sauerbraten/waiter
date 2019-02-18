package main

import (
	"github.com/sauerbraten/waiter/internal/utils"
)

type MapRotation struct {
	Deathmatch []string `json:"deathmatch"`
	CTF        []string `json:"ctf"`
	Capture    []string `json:"capture"`

	queue []string
}

func (mr *MapRotation) NextMap(mode GameMode, currentMap string) string {
	if mode.ID() == s.Mode().ID() {
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

	switch mode.(type) {
	case CTFMode:
		return nextMap(mr.CTF)
	case CaptureMode:
		return nextMap(mr.Capture)
	default:
		return nextMap(mr.Deathmatch)
	}
}

func (mr *MapRotation) inPool(mode GameMode, mapp string) bool {
	_inPool := func(pool []string) bool {
		for _, m := range pool {
			if m == mapp {
				return true
			}
		}
		return false
	}

	switch mode.(type) {
	case CTFMode:
		return _inPool(mr.CTF)
	case CaptureMode:
		return _inPool(mr.Capture)
	default:
		return _inPool(mr.Deathmatch)
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
	if !mr.inPool(s.Mode(), mapp) {
		return mapp + " is not in the map pool for " + s.Mode().ID().String() + "!"
	}
	mr.queue = append(mr.queue, mapp)
	return ""
}
