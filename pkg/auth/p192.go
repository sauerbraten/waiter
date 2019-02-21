package auth

import (
	"crypto/elliptic"
	"math/big"
)

var p192 *elliptic.CurveParams

func init() {
	p192 = &elliptic.CurveParams{Name: "P-192"}
	p192.P, _ = new(big.Int).SetString("6277101735386680763835789423207666416083908700390324961279", 10)
	p192.N, _ = new(big.Int).SetString("6277101735386680763835789423176059013767194773182842284081", 10)
	p192.B, _ = new(big.Int).SetString("64210519e59c80e70fa7e9ab72243049feb8deecc146b9b1", 16)
	p192.Gx, _ = new(big.Int).SetString("188da80eb03090f67cbf20eb43a18800f4ff0afd82ff1012", 16)
	p192.Gy, _ = new(big.Int).SetString("07192b95ffc8da78631011ed6b24cdd573f977a11e794811", 16)
	p192.BitSize = 192
}
