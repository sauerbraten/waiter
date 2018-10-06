package main

import (
	"fmt"
	"log"

	"github.com/sauerbraten/waiter/cubecode"
	"github.com/sauerbraten/waiter/internal/client/privilege"
	"github.com/sauerbraten/waiter/internal/definitions/disconnectreason"
	"github.com/sauerbraten/waiter/internal/definitions/nmc"
	"github.com/sauerbraten/waiter/internal/protocol/enet"
	"github.com/sauerbraten/waiter/internal/protocol/packet"
)

func (s *Server) handleAuthRequest(client *Client, domain string, p *cubecode.Packet) {
	name, ok := p.GetString()
	if !ok {
		log.Println("could not read name from auth try packet:", p)
		return
	}

	challenge, requestID, err := s.Auth.GenerateChallenge(client.CN, domain, name)
	if err != nil {
		log.Println(err)
		return
	}

	client.Peer.Send(1, enet.PACKET_FLAG_RELIABLE, packet.Encode(nmc.AuthChallenge, domain, requestID, challenge))
}

func (s *Server) handleGlobalAuthRequest(client *Client, p *cubecode.Packet) {
	name, ok := p.GetString()
	if !ok {
		log.Println("could not read name from auth try packet:", p)
		return
	}

	requestID := s.Auth.RegisterAuthRequest(client.CN, "", name, privilege.Auth)

	callback := func(sessionID int32) func(string) {
		return func(challenge string) {
			if client == nil || client.SessionID != sessionID {
				return
			}
			client.Peer.Send(1, enet.PACKET_FLAG_RELIABLE, packet.Encode(nmc.AuthChallenge, "", requestID, challenge))
		}
	}(client.SessionID)

	err := ms.RequestAuthChallenge(requestID, name, callback)
	if err != nil {
		s.Auth.ClearAuthRequest(requestID)
		client.Peer.Send(1, enet.PACKET_FLAG_RELIABLE, packet.Encode(nmc.ServerMessage, "not connected to authentication server"))
		return
	}
}

func (s *Server) handleAuthAnswer(client *Client, domain string, p *cubecode.Packet) {
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
	sucess, name, prvlg := s.Auth.CheckAnswer(uint32(requestID), client.CN, domain, answer)
	if !sucess {
		if client.AuthRequiredBecause > disconnectreason.None {
			s.Clients.Disconnect(client, client.AuthRequiredBecause)
		}
		return
	}
	log.Println("sucessful auth by", client.CN)
	client.Privilege = prvlg
	client.AuthRequiredBecause = disconnectreason.None
	pup := s.Clients.PrivilegedUsersPacket()
	if pup != nil {
		s.Clients.Broadcast(nil, 1, enet.PACKET_FLAG_RELIABLE, pup)
	}
	switch client.Privilege {
	case privilege.None:
		s.Clients.Broadcast(nil, 1, enet.PACKET_FLAG_RELIABLE, nmc.ServerMessage, fmt.Sprintf("%s claimed %s as '\fs\f5%s\fr'", client.Name, client.Privilege, name))
	}
}

func (s *Server) handleGlobalAuthAnswer(client *Client, p *cubecode.Packet) {
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

	name, ok := s.Auth.LookupAuthName(requestID)
	if !ok {
		log.Println("no pending request with ID", requestID)
	}

	callback := func(sessionID int32) func(bool) {
		return func(sucess bool) {
			if client == nil || client.SessionID != sessionID || !sucess {
				return
			}
			s.setPrivilege(client, privilege.Auth, name)
		}
	}(client.SessionID)

	err := ms.ConfirmAuthAnswer(requestID, answer, callback)
	if err != nil {
		s.Auth.ClearAuthRequest(requestID)
		client.Peer.Send(1, enet.PACKET_FLAG_RELIABLE, packet.Encode(nmc.ServerMessage, "not connected to authentication server"))
		return
	}
}

func (s *Server) setPrivilege(client *Client, prvlg privilege.Privilege, name string) {
	client.Privilege = prvlg
	client.AuthRequiredBecause = disconnectreason.None
	pup := s.Clients.PrivilegedUsersPacket()
	if pup != nil {
		s.Clients.Broadcast(nil, 1, enet.PACKET_FLAG_RELIABLE, pup)
	}
	switch client.Privilege {
	case privilege.None:
		s.Clients.Broadcast(nil, 1, enet.PACKET_FLAG_RELIABLE, nmc.ServerMessage, fmt.Sprintf("%s claimed %s as '\fs\f5%s\fr'", client.Name, client.Privilege, name))
	}
}
