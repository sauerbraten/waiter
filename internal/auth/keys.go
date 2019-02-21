package auth

import (
	"crypto/elliptic"
	"crypto/rand"
	"encoding/json"
	"errors"
	"math/big"
)

type privateKey []byte

type publicKey struct {
	x *big.Int
	y *big.Int
}

func (k publicKey) MarshalJSON() ([]byte, error) {
	return json.Marshal(formatPublicKey(k))
}

func (k *publicKey) UnmarshalJSON(data []byte) error {
	var proxy string
	err := json.Unmarshal(data, &proxy)
	if err != nil {
		return err
	}
	pub, err := parsePublicKey(proxy)
	k.x, k.y = pub.x, pub.y
	return err
}

func GenerateKeyPair() (priv privateKey, pub publicKey, err error) {
	priv, pub.x, pub.y, err = elliptic.GenerateKey(p192, rand.Reader)
	return
}

func formatPublicKey(pub publicKey) (s string) { return encodePoint(pub.x, pub.y) }

func parsePublicKey(s string) (publicKey, error) {
	if len(s) < 1 {
		return publicKey{}, errors.New("auth: could not parse public key: too short")
	}

	neg := s[0] == '-'

	var (
		pub = publicKey{
			x: new(big.Int),
			y: new(big.Int),
		}
		xxx    = new(big.Int)
		threeX = new(big.Int)
		y2     = new(big.Int)
	)

	_, ok := pub.x.SetString(s[1:], 16)
	if !ok {
		return publicKey{}, errors.New("auth: could not set X coordinate of public key")
	}

	// the next steps find y using the formula y^2 = x^3 - 3*x + B
	x := pub.x
	xxx.Mul(x, x).Mul(xxx, x)           // x^3
	threeX.Add(x, x).Add(threeX, x)     // 3*x
	y2.Sub(xxx, threeX).Add(y2, p192.B) // x^3 - 3*x + B
	pub.y.ModSqrt(y2, p192.P)           // find a square root

	if neg && pub.y.Bit(0) == 1 {
		pub.y.Neg(pub.y)
	}

	return pub, nil
}
