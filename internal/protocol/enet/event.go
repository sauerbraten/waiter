package enet

/*
#cgo LDFLAGS: -lenet
#include <enet/enet.h>
*/
import "C"

type EventType uint

const (
	EVENT_TYPE_NONE       EventType = C.ENET_EVENT_TYPE_NONE
	EVENT_TYPE_CONNECT    EventType = C.ENET_EVENT_TYPE_CONNECT
	EVENT_TYPE_DISCONNECT EventType = C.ENET_EVENT_TYPE_DISCONNECT
	EVENT_TYPE_RECEIVE    EventType = C.ENET_EVENT_TYPE_RECEIVE
)

type Event struct {
	Type      EventType
	Peer      *Peer
	ChannelID uint8
	Data      uint32
	Packet    *Packet
}

func eventFromCEvent(cEvent *C.ENetEvent) Event {
	e := Event{
		Type:      EventType(cEvent._type),
		Peer:      peerFromCPeer(cEvent.peer),
		ChannelID: uint8(cEvent.channelID),
		Data:      uint32(cEvent.data),
	}

	if e.Type == EVENT_TYPE_RECEIVE {
		e.Packet = packetFromCPacket(cEvent.packet)
		C.enet_packet_destroy(cEvent.packet)
	}

	return e

}
