package auth

import (
	"crypto/elliptic"
	"crypto/rand"
)

func GenerateChallenge(pub PublicKey) (challenge, solution string, err error) {
	secret, x, y, err := elliptic.GenerateKey(p192, rand.Reader)

	// what we send to the client
	challenge = encodePoint(x, y)

	// what the client should return if she applies her private key to the challenge
	// (see Solve below)
	solX, _ := p192.ScalarMult(pub.x, pub.y, secret)
	solution = solX.Text(16)

	return
}

func Solve(challenge string, priv PrivateKey) (string, error) {
	x, y, err := parsePoint(challenge)
	if err != nil {
		return "", err
	}

	solX, _ := p192.ScalarMult(x, y, priv)
	return solX.Text(16), nil
}
