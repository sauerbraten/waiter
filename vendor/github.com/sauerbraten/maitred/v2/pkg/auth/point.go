package auth

import (
	"errors"
	"math/big"
)

// from ecjacobian::print() in shared/crypto.cpp
func encodePoint(x, y *big.Int) (s string) {
	if y.Bit(0) == 1 {
		s += "-"
	} else {
		s += "+"
	}
	s += x.Text(16)
	return
}

func parsePoint(s string) (x, y *big.Int, err error) {
	if len(s) < 1 {
		return nil, nil, errors.New("auth: could not parse curve point: too short")
	}

	var ok bool
	x, ok = new(big.Int).SetString(s[1:], 16)
	if !ok {
		return nil, nil, errors.New("auth: could not set X coordinate of curve point")
	}

	// the next steps find y using the formula y^2 = x^3 - 3*x + B
	// x^3
	xxx := new(big.Int).Mul(x, x)
	xxx.Mul(xxx, x)
	// 3*x
	threeX := new(big.Int).Add(x, x)
	threeX.Add(threeX, x)
	// x^3 - 3*x + B
	yy := new(big.Int).Sub(xxx, threeX)
	yy.Add(yy, p192.B)

	// find a square root
	y = new(big.Int).ModSqrt(yy, p192.P)

	if s[0] == '-' && y.Bit(0) == 0 {
		y.Neg(y)
	}

	return
}
