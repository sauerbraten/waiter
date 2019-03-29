package maprot

import (
	"math/rand"
	"time"

	"github.com/sauerbraten/waiter/pkg/protocol/gamemode"
)

type Pools struct {
	Deathmatch []string `json:"deathmatch"`
	CTF        []string `json:"ctf"`
	Capture    []string `json:"capture"`
}

type Rotation struct {
	pools Pools
	queue []string
	rng   *rand.Rand
}

func NewRotation(pools Pools) *Rotation {
	return &Rotation{
		pools: pools,
		rng:   rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (r *Rotation) QueuedMaps() []string {
	q := make([]string, len(r.queue))
	copy(q, r.queue)
	return q
}

func (r *Rotation) ClearQueue() { r.queue = r.queue[:0] }

func (r *Rotation) NextMap(mode, currentMode gamemode.ID, currentMap string) string {
	if mode == currentMode {
		if len(r.queue) > 0 {
			mapp := r.queue[0]
			r.queue = r.queue[1:]
			return mapp
		}
		r.ClearQueue()
	}

	nextMap := func(pool []string) string {
		for i, m := range pool {
			if m == currentMap {
				return pool[(i+1)%len(pool)]
			}
		}

		// current map wasn't found in map rotation, return random map in rotation
		return pool[r.rng.Intn(len(pool))]
	}

	if gamemode.IsCTF(mode) {
		return nextMap(r.pools.CTF)
	} else if gamemode.IsCapture(mode) {
		return nextMap(r.pools.Capture)
	} else {
		return nextMap(r.pools.Deathmatch)
	}
}

func (r *Rotation) InPool(mode gamemode.ID, mapp string) bool {
	inPool := func(pool []string) bool {
		for _, m := range pool {
			if m == mapp {
				return true
			}
		}
		return false
	}

	if gamemode.IsCTF(mode) {
		return inPool(r.pools.CTF)
	} else if gamemode.IsCapture(mode) {
		return inPool(r.pools.Capture)
	} else {
		return inPool(r.pools.Deathmatch)
	}
}

func (r *Rotation) inQueue(mapp string) bool {
	for _, m := range r.queue {
		if m == mapp {
			return true
		}
	}
	return false
}

func (r *Rotation) QueueMap(currentMode gamemode.ID, mapp string) (err string) {
	if r.inQueue(mapp) {
		return mapp + " is already queued!"
	}
	if !r.InPool(currentMode, mapp) {
		return mapp + " is not in the map pool for " + currentMode.String() + "!"
	}
	r.queue = append(r.queue, mapp)
	return ""
}
