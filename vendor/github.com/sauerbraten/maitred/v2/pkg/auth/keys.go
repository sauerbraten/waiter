package auth

import (
	"crypto/elliptic"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"math/big"
)

type PrivateKey []byte

func ParsePrivateKey(s string) (PrivateKey, error) {
	return hex.DecodeString(s)
}

func (k PrivateKey) String() string {
	return hex.EncodeToString(k)
}

type PublicKey struct {
	x *big.Int
	y *big.Int
}

func ParsePublicKey(s string) (PublicKey, error) {
	x, y, err := parsePoint(s)
	return PublicKey{
		x: x,
		y: y,
	}, err
}

func (k PublicKey) String() string {
	return encodePoint(k.x, k.y)
}

func (k PublicKey) MarshalJSON() ([]byte, error) {
	return json.Marshal(k.String())
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
