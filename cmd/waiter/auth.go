package main

import (
	"fmt"
	"log"

	"github.com/sauerbraten/waiter/pkg/protocol"
	"github.com/sauerbraten/waiter/pkg/protocol/cubecode"

	"github.com/sauerbraten/waiter/internal/client/privilege"
	"github.com/sauerbraten/waiter/internal/definitions/disconnectreason"
	"github.com/sauerbraten/waiter/internal/definitions/nmc"
	"github.com/sauerbraten/waiter/internal/net/enet"
	"github.com/sauerbraten/waiter/internal/net/packet"
)

func (s *Server) handleAuthRequest(client *Client, domain string, name string) {
	challenge, requestID, err := s.Auth.GenerateChallenge(client.CN, domain, name)
	if err != nil {
		log.Println(err)
		return
	}

	client.Peer.Send(1, enet.PACKET_FLAG_RELIABLE, packet.Encode(nmc.AuthChallenge, domain, requestID, challenge))
}

func (s *Server) handleGlobalAuthRequest(client *Client, name string) {
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
	sucess, name, prvlg := s.Auth.CheckAnswer(uint32(requestID), client.CN, domain, answer)
	if !sucess {
		if client.AuthRequiredBecause > disconnectreason.None {
			s.Clients.Disconnect(client, client.AuthRequiredBecause)
		}
		return
	}
	log.Printf("successful auth by %s (%d) as %s [%s]\n", client.Name, client.CN, name, domain)
	s.setAuthPrivilege(client, prvlg, domain, name)
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

	name, ok := s.Auth.LookupAuthName(requestID)
	if !ok {
		log.Println("no pending request with ID", requestID)
	}

	callback := func(sessionID int32) func(bool) {
		return func(sucess bool) {
			if client == nil || client.SessionID != sessionID || !sucess {
				return
			}
			log.Printf("successful gauth by %s (%d) as %s\n", client.Name, client.CN, name)
			s.setAuthPrivilege(client, privilege.Auth, "", name)
		}
	}(client.SessionID)

	err := ms.ConfirmAuthAnswer(requestID, answer, callback)
	if err != nil {
		s.Auth.ClearAuthRequest(requestID)
		client.Peer.Send(1, enet.PACKET_FLAG_RELIABLE, packet.Encode(nmc.ServerMessage, cubecode.Error("not connected to authentication server")))
		return
	}
}

func (s *Server) setAuthPrivilege(client *Client, prvlg privilege.ID, domain, name string) {
	s.setPrivilege(client, prvlg)
	msg := fmt.Sprintf("%s claimed %s privileges as %s", s.Clients.UniqueName(client), client.Privilege, cubecode.Magenta(name))
	if domain != "" {
		msg = fmt.Sprintf("%s claimed %s privileges as %s [%s]", s.Clients.UniqueName(client), client.Privilege, cubecode.Magenta(name), cubecode.Green(domain))
	}
	s.Clients.Broadcast(nil, 1, enet.PACKET_FLAG_RELIABLE, nmc.ServerMessage, msg)
}

func (s *Server) setPrivilege(client *Client, prvlg privilege.ID) {
	client.Privilege = prvlg
	if prvlg > privilege.None {
		client.AuthRequiredBecause = disconnectreason.None
	}
	pup, _ := s.Clients.PrivilegedUsersPacket()
	s.Clients.Broadcast(nil, 1, enet.PACKET_FLAG_RELIABLE, pup)
}
