package auth

import (
	"crypto/elliptic"
	"crypto/rand"
	"encoding/json"
	"errors"
	"math/big"
)

type PrivateKey []byte

type PublicKey struct {
	x *big.Int
	y *big.Int
}

func (k PublicKey) MarshalJSON() ([]byte, error) {
	return json.Marshal(FormatPublicKey(k))
}

func (k *PublicKey) UnmarshalJSON(data []byte) error {
	var proxy string
	err := json.Unmarshal(data, &proxy)
	if err != nil {
		return err
	}
	pub, err := ParsePublicKey(proxy)
	k.x, k.y = pub.x, pub.y
	return err
}

func GenerateKeyPair() (priv PrivateKey, pub PublicKey, err error) {
	priv, pub.x, pub.y, err = elliptic.GenerateKey(p192, rand.Reader)
	return
}

func FormatPublicKey(pub PublicKey) (s string) { return encodePoint(pub.x, pub.y) }

func ParsePublicKey(s string) (PublicKey, error) {
	if len(s) < 1 {
		return PublicKey{}, errors.New("auth: could not parse public key: too short")
	}

	neg := s[0] == '-'

	var (
		pub = PublicKey{
			x: new(big.Int),
			y: new(big.Int),
		}
		xxx    = new(big.Int)
		threeX = new(big.Int)
		y2     = new(big.Int)
	)

	_, ok := pub.x.SetString(s[1:], 16)
	if !ok {
		return PublicKey{}, errors.New("auth: could not set X coordinate of public key")
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
