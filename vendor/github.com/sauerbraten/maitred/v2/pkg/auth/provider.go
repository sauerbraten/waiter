package auth

import (
	"github.com/sauerbraten/waiter/pkg/protocol/role"
)

type Provider interface {
	GenerateChallenge(name string, callback func(reqID uint32, chal string, err error))
	ConfirmAnswer(reqID uint32, answ string, callback func(rol role.ID, err error))
}
