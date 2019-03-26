package rng

import (
	"math/rand"
	"time"
)

var RNG = rand.New(rand.NewSource(time.Now().UnixNano()))
