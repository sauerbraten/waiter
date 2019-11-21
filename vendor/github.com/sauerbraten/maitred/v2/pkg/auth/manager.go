package auth

import (
	"fmt"

	"github.com/sauerbraten/waiter/pkg/protocol/role"
)

type callbacks struct {
	onSuccess func(role.ID)
	onFailure func(error)
}

type Manager struct {
	providersByDomain  map[string]Provider
	callbacksByRequest map[uint32]callbacks
}

func NewManager(providers map[string]Provider) *Manager {
	return &Manager{
		providersByDomain:  providers,
		callbacksByRequest: map[uint32]callbacks{},
	}
}

func (m *Manager) TryAuthentication(domain, name string, onChal func(reqID uint32, chal string), onSuccess func(role.ID), onFailure func(error)) {
	p, ok := m.providersByDomain[domain]
	if !ok {
		onFailure(fmt.Errorf("auth: no provider for domain '%s'", domain))
		return
	}

	p.GenerateChallenge(name, func(reqID uint32, chal string, err error) {
		if err != nil {
			onFailure(err)
			return
		}
		m.callbacksByRequest[reqID] = callbacks{
			onSuccess: onSuccess,
			onFailure: onFailure,
		}
		onChal(reqID, chal)
	})

	return
}

func (m *Manager) CheckAnswer(reqID uint32, domain string, answ string) (err error) {
	defer delete(m.callbacksByRequest, reqID)

	p, ok := m.providersByDomain[domain]
	if !ok {
		err = fmt.Errorf("auth: no provider for domain '%s'", domain)
		return
	}

	callbacks, ok := m.callbacksByRequest[reqID]
	if !ok {
		err = fmt.Errorf("auth: unkown request '%d'", reqID)
		return
	}

	p.ConfirmAnswer(reqID, answ, func(rol role.ID, err error) {
		if err != nil {
			go callbacks.onFailure(err)
			return
		}
		go callbacks.onSuccess(rol)
	})

	return
}
