package main

import (
	"fmt"
	"log"

	"github.com/sauerbraten/waiter/internal/definitions/nmc"
	"github.com/sauerbraten/waiter/internal/definitions/role"
	"github.com/sauerbraten/waiter/pkg/auth"
	"github.com/sauerbraten/waiter/pkg/protocol"
	"github.com/sauerbraten/waiter/pkg/protocol/cubecode"
)

func (s *Server) handleAuthRequest(client *Client, domain string, name string, onSuccess auth.CallbackWithRole, onFailure auth.Callback) {
	challenge, requestID, err := s.Auth.GenerateChallenge(client.CN, domain, name, onSuccess, onFailure)
	if err != nil {
		log.Println(err)
		return
	}

	client.Send(nmc.AuthChallenge, domain, requestID, challenge)
}

func (s *Server) handleGlobalAuthRequest(client *Client, name string, onSuccess, onFailure auth.Callback) {
	requestID := s.Auth.RegisterAuthRequest(client.CN, "", name, onSuccess, onFailure)

	callback := func(sessionID int32) func(string) {
		return func(challenge string) {
			if client == nil || client.SessionID != sessionID {
				return
			}
			client.Send(nmc.AuthChallenge, "", requestID, challenge)
		}
	}(client.SessionID)

	err := ms.RequestAuthChallenge(requestID, name, callback)
	if err != nil {
		s.Auth.ClearAuthRequest(requestID)
		client.Send(nmc.ServerMessage, "not connected to authentication server")
		return
	}
}

func (s *Server) handleAuthAnswer(client *Client, domain string, p *protocol.Packet) {
	requestID, ok := p.GetInt()
	if !ok {
		log.Println("could not read request ID from auth answer packet:", p)
		return
	}
	answer, ok := p.GetString()
	if !ok {
		log.Println("could not read answer from auth answer packet:", p)
		return
	}
	s.Auth.CheckAnswer(uint32(requestID), client.CN, domain, answer)
}

func (s *Server) handleGlobalAuthAnswer(client *Client, p *protocol.Packet) {
	_requestID, ok := p.GetInt()
	if !ok {
		log.Println("could not read request ID from auth answer packet:", p)
		return
	}
	requestID := uint32(_requestID)
	answer, ok := p.GetString()
	if !ok {
		log.Println("could not read answer from auth answer packet:", p)
		return
	}

	onSuccess, onFailure, ok := s.Auth.LookupGlobalAuthRequest(requestID)
	if !ok {
		log.Println("no pending request with ID", requestID)
	}

	callback := func(sessionID int32) func(bool) {
		return func(success bool) {
			if client == nil || client.SessionID != sessionID {
				return
			}
			if success {
				onSuccess()
			} else {
				onFailure()
			}
		}
	}(client.SessionID)

	err := ms.ConfirmAuthAnswer(requestID, answer, callback)
	if err != nil {
		client.Send(nmc.ServerMessage, cubecode.Error("not connected to authentication server"))
		return
	}
}

func (s *Server) setAuthRole(client *Client, prvlg role.ID, domain, name string) {
	msg := fmt.Sprintf("%s claimed %s privileges as '%s'", s.Clients.UniqueName(client), prvlg, cubecode.Magenta(name))
	if domain != "" {
		msg = fmt.Sprintf("%s claimed %s privileges as '%s' [%s]", s.Clients.UniqueName(client), prvlg, cubecode.Magenta(name), cubecode.Green(domain))
	}
	s.Clients.Broadcast(nil, nmc.ServerMessage, msg)
	log.Println(cubecode.SanitizeString(msg))

	s._setRole(client, prvlg)
}

func (s *Server) setRole(client *Client, targetCN uint32, rol role.ID) {
	target := s.Clients.GetClientByCN(targetCN)
	if target == nil {
		client.Send(nmc.ServerMessage, cubecode.Fail(fmt.Sprintf("no client with CN %d", targetCN)))
		return
	}
	if target.Role == rol {
		return
	}
	if client != target && client.Role <= target.Role || client == target && rol != role.None {
		client.Send(nmc.ServerMessage, cubecode.Fail("you can't do that"))
		return
	}

	var msg string
	if rol == role.None {
		if client == target {
			msg = fmt.Sprintf("%s relinquished %s privileges", s.Clients.UniqueName(client), target.Role)
		} else {
			msg = fmt.Sprintf("%s took away %s privileges from %s", s.Clients.UniqueName(client), target.Role, s.Clients.UniqueName(target))
		}
	} else {
		msg = fmt.Sprintf("%s gave %s privileges to %s", s.Clients.UniqueName(client), rol, s.Clients.UniqueName(target))
	}
	s.Clients.Broadcast(nil, nmc.ServerMessage, msg)
	log.Println(cubecode.SanitizeString(msg))

	s._setRole(target, rol)
}

func (s *Server) _setRole(client *Client, rol role.ID) {
	client.Role = rol
	pup, _ := s.Clients.PrivilegedUsersPacket()
	s.Clients.Broadcast(nil, pup)
}
