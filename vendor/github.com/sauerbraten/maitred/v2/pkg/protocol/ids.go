package protocol

import "math"

type IDCycle uint32

func (c *IDCycle) Next() uint32 {
	n := uint32(*c)
	*c++
	if *c == math.MaxUint32 {
		*c = 0
	}
	return n
}
