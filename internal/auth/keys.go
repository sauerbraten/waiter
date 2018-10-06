package auth

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
)

type privateKey []byte

type publicKey struct {
	x *big.Int
	y *big.Int
}

// proxy for JSON (un)marshalling
type _publicKey struct {
	X string `json:"x"`
	Y string `json:"y"`
}

func (k publicKey) MarshalJSON() ([]byte, error) {
	proxy := _publicKey{
		X: k.x.Text(16),
		Y: k.y.Text(16),
	}
	return json.Marshal(proxy)
}

func (k *publicKey) UnmarshalJSON(data []byte) error {
	proxy := _publicKey{}
	err := json.Unmarshal(data, &proxy)
	if err != nil {
		return err
	}
	var ok bool
	k.x, ok = new(big.Int).SetString(proxy.X, 16)
	if !ok {
		return fmt.Errorf("auth: could not unmarshal public key: invalid format for X coordinate")
	}
	k.y, ok = new(big.Int).SetString(proxy.Y, 16)
	if !ok {
		return fmt.Errorf("auth: could not unmarshal public key: invalid format for Y coordinate")
	}
	return nil
}

func GenerateKeyPair() (priv privateKey, pub publicKey, err error) {
	priv = make([]byte, 24)
	_, err = rand.Read(priv)
	if err != nil {
		err = fmt.Errorf("auth: not enough entropy to create a private key: %v", err)
		return
	}
	pub.x, pub.y = p192.ScalarBaseMult(priv)
	return
}
