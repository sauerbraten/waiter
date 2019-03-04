package enet

/*
#cgo LDFLAGS: -lenet
#include <enet/enet.h>

*/
import "C"

import (
	"log"
	"net"
	"unsafe"

	"github.com/sauerbraten/waiter/pkg/definitions/disconnectreason"
	"github.com/sauerbraten/waiter/pkg/definitions/nmc"
)

type PeerState uint

type Peer struct {
	Address *net.UDPAddr
	State   PeerState
	cPeer   *C.ENetPeer
}

func (h *Host) peerFromCPeer(cPeer *C.ENetPeer) *Peer {
	if cPeer == nil {
		return nil
	}

	// peer exists already
	if p, ok := h.peers[cPeer]; ok {
		return p
	}

	ipBytes := uint32(cPeer.address.host)
	ip := net.IPv4(byte((ipBytes<<24)>>24), byte((ipBytes<<16)>>24), byte((ipBytes<<8)>>24), byte(ipBytes>>24))

	p := &Peer{
		Address: &net.UDPAddr{
			IP:   ip,
			Port: int(cPeer.address.port),
		},
		State: PeerState(cPeer.state),
		cPeer: cPeer,
	}

	h.peers[cPeer] = p

	return p
}

func (h *Host) Disconnect(p *Peer, reason disconnectreason.ID) {
	C.enet_peer_disconnect(p.cPeer, C.enet_uint32(reason))
	delete(h.peers, p.cPeer)
}

func (p *Peer) Send(channel uint8, payload []byte) {
	if len(payload) == 0 {
		return
	}

	flags := ^uint32(PacketFlagNoAllocate) // always allocate (safer with CGO usage below)
	if channel == 1 {
		flags = flags & PacketFlagReliabe
	}

	switch nmc.ID(payload[0]) {
	case nmc.Position,
		nmc.Teleport,
		nmc.JumpPad,
		nmc.ServerInfo,
		nmc.Welcome,
		nmc.InitializeClient,
		nmc.Leave,
		nmc.Died,
		nmc.Damage,
		nmc.HitPush,
		nmc.ShotEffects,
		nmc.ExplodeEffects,
		nmc.SpawnState,
		nmc.SetTeam,
		nmc.MapChange,
		nmc.Pong,
		nmc.ClientPing,
		nmc.TimeLeft,
		nmc.ServerMessage,
		nmc.CurrentMaster,
		nmc.AuthChallenge,
		nmc.InitFlags,
		nmc.DropFlag,
		nmc.ReturnFlag,
		nmc.TakeFlag,
		nmc.ScoreFlag,
		nmc.ResetFlag,
		nmc.Spectator,
		nmc.Client:
	// do nothing
	default:
		log.Println("sending", payload, "to", p.Address.String())
	}

	packet := C.enet_packet_create(unsafe.Pointer(&payload[0]), C.size_t(len(payload)), C.enet_uint32(flags))
	C.enet_peer_send(p.cPeer, C.enet_uint8(channel), packet)
}
