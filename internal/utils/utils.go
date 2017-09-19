package utils

import (
	"math/rand"
	"time"
)

var RNG = rand.New(rand.NewSource(time.Now().UnixNano()))

func Min(i, j int) int {
	if i < j {
		return i
	}
	return j
}

func Max(i, j int32) int32 {
	if i > j {
		return i
	}
	return j
}
