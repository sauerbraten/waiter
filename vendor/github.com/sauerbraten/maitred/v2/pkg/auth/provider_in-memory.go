package auth

import (
	"errors"

	"github.com/sauerbraten/maitred/v2/pkg/protocol"
	"github.com/sauerbraten/waiter/pkg/protocol/role"
)

// request holds the data we need to remember between
// generating a challenge and checking the response.
type request struct {
	id       uint32
	user     *User
	solution string
}

type InMemoryProvider struct {
	ids             *protocol.IDCycle
	usersByName     map[string]*User
	pendingRequests map[uint32]*request
}

func NewInMemoryProvider(users []*User) *InMemoryProvider {
	p := &InMemoryProvider{
		ids:             new(protocol.IDCycle),
		usersByName:     map[string]*User{},
		pendingRequests: map[uint32]*request{},
	}
	for _, u := range users {
		p.usersByName[u.Name] = u
	}
	return p
}

func (p *InMemoryProvider) GenerateChallenge(name string, callback func(uint32, string, error)) {
	u, ok := p.usersByName[name]
	if !ok {
		callback(0, "", errors.New("auth: user not found"))
		return
	}

	req := &request{
		id:   p.ids.Next(),
		user: u,
	}

	chal, sol, err := GenerateChallenge(u.PublicKey)
	if err == nil {
		req.solution = sol
		p.pendingRequests[req.id] = req
	}
	callback(req.id, chal, err)
}

func (p *InMemoryProvider) ConfirmAnswer(reqID uint32, answ string, callback func(role.ID, error)) {
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
