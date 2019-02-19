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

	"github.com/sauerbraten/waiter/internal/definitions/nmc"
)

type PeerState uint

type Peer struct {
	Address *net.UDPAddr
	Network net.IPNet // for bans
	State   PeerState
	cPeer   *C.ENetPeer
}

func peerFromCPeer(cPeer *C.ENetPeer) *Peer {
	if cPeer == nil {
		return nil
	}

	// peer exists already
	if p, ok := peers[cPeer]; ok {
		return p
	}

	ipBytes := uint32(cPeer.address.host)
	ip := net.IPv4(byte((ipBytes<<24)>>24), byte((ipBytes<<16)>>24), byte((ipBytes<<8)>>24), byte(ipBytes>>24))

	p := &Peer{
		Address: &net.UDPAddr{
			IP:   ip,
			Port: int(cPeer.address.port),
		},
		Network: net.IPNet{
			IP:   ip,
			Mask: ip.DefaultMask(),
		},
		State: PeerState(cPeer.state),
		cPeer: cPeer,
	}

	peers[cPeer] = p

	return p
}

func (p *Peer) Disconnect(reason uint32) {
	delete(peers, p.cPeer)
	C.enet_peer_disconnect(p.cPeer, C.enet_uint32(reason))
}

func (p *Peer) Send(channel uint8, flags PacketFlag, payload []byte) {
	if len(payload) == 0 {
		return
	}

	flags = flags & ^PACKET_FLAG_NO_ALLOCATE // always allocate (safer with CGO usage below)

	switch nmc.ID(payload[0]) {
	case nmc.Position,
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
		nmc.Client:
	// do nothing
	default:
		log.Println("sending", payload, "to", p.Address.String())
	}

	packet := C.enet_packet_create(unsafe.Pointer(&payload[0]), C.size_t(len(payload)), C.enet_uint32(flags))
	C.enet_peer_send(p.cPeer, C.enet_uint8(channel), packet)
}
