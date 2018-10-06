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
)

type PeerState uint

const (
	PEER_STATE_DISCONNECTED             PeerState = C.ENET_PEER_STATE_DISCONNECTED
	PEER_STATE_CONNECTING               PeerState = C.ENET_PEER_STATE_CONNECTING
	PEER_STATE_ACKNOWLEDGING_CONNECT    PeerState = C.ENET_PEER_STATE_ACKNOWLEDGING_CONNECT
	PEER_STATE_CONNECTION_PENDING       PeerState = C.ENET_PEER_STATE_CONNECTION_PENDING
	PEER_STATE_CONNECTION_SUCCEEDED     PeerState = C.ENET_PEER_STATE_CONNECTION_SUCCEEDED
	PEER_STATE_CONNECTED                PeerState = C.ENET_PEER_STATE_CONNECTED
	PEER_STATE_DISCONNECT_LATER         PeerState = C.ENET_PEER_STATE_DISCONNECT_LATER
	PEER_STATE_DISCONNECTING            PeerState = C.ENET_PEER_STATE_DISCONNECTING
	PEER_STATE_ACKNOWLEDGING_DISCONNECT PeerState = C.ENET_PEER_STATE_ACKNOWLEDGING_DISCONNECT
	PEER_STATE_ZOMBIE                   PeerState = C.ENET_PEER_STATE_ZOMBIE
)

type Peer struct {
	Address net.UDPAddr
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
		Address: net.UDPAddr{
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

	switch payload[0] {
	case 4, 31, 32, 33, 85:
	// do nothing
	default:
		log.Println("sending", payload)
	}

	packet := C.enet_packet_create(unsafe.Pointer(&payload[0]), C.size_t(len(payload)), C.enet_uint32(flags))
	C.enet_peer_send(p.cPeer, C.enet_uint8(channel), packet)
}
