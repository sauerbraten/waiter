package enet

/*
#cgo LDFLAGS: -lenet
#include <enet/enet.h>
*/
import "C"

import (
	"unsafe"
)

type PacketFlag uint32

const (
	PACKET_FLAG_NONE                PacketFlag = 0
	PACKET_FLAG_RELIABLE                       = (1 << 0)
	PACKET_FLAG_UNSEQUENCED                    = (1 << 1)
	PACKET_FLAG_NO_ALLOCATE                    = (1 << 2)
	PACKET_FLAG_UNRELIABLE_FRAGMENT            = (1 << 3)
	PACKET_FLAG_SENT                           = (1 << 8)
)

type Packet struct {
	Flags uint32 // bitwise-or of ENetPacketFlag constants
	Data  []byte // allocated data for packet
}

func packetFromCPacket(cPacket *C.ENetPacket) Packet {
	if cPacket == nil {
		return Packet{}
	}

	return Packet{
		Flags: uint32(cPacket.flags),
		Data:  C.GoBytes(unsafe.Pointer(cPacket.data), C.int(cPacket.dataLength)),
	}
}
