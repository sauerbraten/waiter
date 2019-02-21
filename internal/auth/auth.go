// package auth implements server-side functionality for Sauerbraten's player authentication mechanism.
//
// The mechanism relies on the associativity of scalar multiplication on elliptic curves: private keys
// are random (big) scalars, and the corresponding public key is created by multiplying the curve base point
// with the private key. (This means the public key is another point on the curve.)
// To check for posession of the private key belonging to a public key known to the server, the base point is
// multiplied with another random, big scalar (the "secret") and the resulting point is sent to the user as
// "challenge". The user multiplies the challenge curve point with his private key (a scalar), and sends the
// X coordinate of the resulting point back to the server.
// The server instead multiplies the user's public key with the secret scalar. Since pub = base * priv,
// pub * secret = (base * priv) * secret = (base * secret) * priv = challenge * priv. Because of the curve's
// symmetry, there are exactly two points on the curve at any given X. For simplicity (and maybe performance),
// the server is satisfied when the user responds with the correct X.
package auth

import (
	"crypto/elliptic"
	"crypto/rand"
	"errors"
	"log"
	"math/big"
	mrand "math/rand"
	"time"

	"github.com/sauerbraten/waiter/internal/definitions/role"
)

func init() {
	mrand.Seed(time.Now().UnixNano())
}

type CallbackWithRole func(rol role.ID)
type Callback func()

// request holds the data we need to remember between
// generating a challenge and checking the response.
type request struct {
	id       uint32
	domain   string
	name     string
	cn       uint32
	solution string

	onSuccess Callback
	onFailure Callback
}

type Manager struct {
	users   map[UserIdentifier]*User
	pending map[uint32]*request
}

func NewManager(users []*User) *Manager {
	m := &Manager{
		users:   map[UserIdentifier]*User{},
		pending: map[uint32]*request{},
	}
	for _, u := range users {
		m.users[u.UserIdentifier] = u
	}
	return m
}

func (m *Manager) RegisterAuthRequest(cn uint32, domain, name string, onSuccess, onFailure Callback) (requestID uint32) {
	requestID = mrand.Uint32()
	req := &request{
		id:     requestID,
		domain: domain,
		name:   name,
		cn:     cn,

		onSuccess: onSuccess,
		onFailure: onFailure,
	}
	m.pending[requestID] = req
	return
}

func (m *Manager) LookupGlobalAuthRequest(requestID uint32) (Callback, Callback, bool) {
	defer m.ClearAuthRequest(requestID)
	req, ok := m.pending[requestID]
	return req.onSuccess, req.onFailure, ok
}

func (m *Manager) ClearAuthRequest(requestID uint32) { delete(m.pending, requestID) }

func (m *Manager) GenerateChallenge(cn uint32, domain, name string, onSuccess CallbackWithRole, onFailure Callback) (challenge string, requestID uint32, err error) {
	log.Println("generating challenge for", name, domain)
	u, ok := m.users[UserIdentifier{Name: name, Domain: domain}]
	if !ok {
		return "", 0, errors.New("auth: user '" + name + "' not found in domain '" + domain + "'")
	}

	onSuccessWithRole := func() { onSuccess(u.Role) }

	requestID = m.RegisterAuthRequest(cn, domain, name, onSuccessWithRole, onFailure)

	challenge, solution, err := generateChallenge(u.PublicKey)
	if err != nil {
		m.ClearAuthRequest(requestID)
		return "", 0, err
	}

	m.pending[requestID].solution = solution

	return challenge, requestID, nil
}

func (m *Manager) CheckAnswer(requestID, cn uint32, domain string, answer string) {
	defer m.ClearAuthRequest(requestID)
	req, ok := m.pending[requestID]
	if !ok {
		return
	}
	successful := requestID == req.id && domain == req.domain && cn == req.cn && answer == req.solution
	if successful {
		req.onSuccess()
	} else {
		req.onFailure()
	}
}

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

func generateChallenge(pub publicKey) (challenge, solution string, err error) {
	secret, x, y, err := elliptic.GenerateKey(p192, rand.Reader)

	// what we send to the client
	challenge = encodePoint(x, y)

	// what the client should return if she applies her private key to the challenge
	solX, _ := p192.ScalarMult(pub.x, pub.y, secret)
	solution = solX.Text(16)

	return
}
