package auth

import (
	"errors"
	"math"

	"github.com/sauerbraten/maitred/pkg/auth"

	"github.com/sauerbraten/waiter/pkg/definitions/role"
)

type IDCycle uint32

func (c *IDCycle) NextID() uint32 {
	n := uint32(*c)
	*c++
	if *c == math.MaxUint32 {
		*c = 0
	}
	return n
}

// request holds the data we need to remember between
// generating a challenge and checking the response.
type request struct {
	id       uint32
	user     *User
	solution string
}

type inMemoryProvider struct {
	*IDCycle
	usersByName     map[string]*User
	pendingRequests map[uint32]*request
}

func NewInMemoryProvider(users []*User) Provider {
	p := &inMemoryProvider{
		IDCycle:         new(IDCycle),
		usersByName:     map[string]*User{},
		pendingRequests: map[uint32]*request{},
	}
	for _, u := range users {
		p.usersByName[u.Name] = u
	}
	return p
}

func (p *inMemoryProvider) GenerateChallenge(name string, callback func(uint32, string, error)) {
	u, ok := p.usersByName[name]
	if !ok {
		callback(0, "", errors.New("auth: user not found"))
		return
	}

	req := &request{
		id:   p.NextID(),
		user: u,
	}

	chal, sol, err := auth.GenerateChallenge(u.PublicKey)
	if err == nil {
		req.solution = sol
		p.pendingRequests[req.id] = req
	}
	callback(req.id, chal, err)
}

func (p *inMemoryProvider) ConfirmAnswer(reqID uint32, answ string, callback func(role.ID, error)) {
	req, ok := p.pendingRequests[reqID]
	if !ok {
		callback(role.None, errors.New("auth: request not found"))
		return
	}
	if answ != req.solution {
		callback(role.None, errors.New("auth: wrong answer"))
		return
	}
	callback(req.user.Role, nil)
}
