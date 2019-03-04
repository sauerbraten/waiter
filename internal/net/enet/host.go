package enet

/*
#cgo LDFLAGS: -lenet
#include <stdio.h>
#include <stdlib.h>
#include <enet/enet.h>


ENetHost * initServer(const char *addr, int port) {
	if (enet_initialize() != 0) {
		fprintf (stderr, "An error occurred while initializing ENet.\n");
		return NULL;
	}
	atexit(enet_deinitialize);

	ENetAddress address;

	// Bind the server to the provided address
	address.host = ENET_HOST_ANY;

	// Bind the server to the provided port
	address.port = port;

	ENetHost * server = enet_host_create(&address, 128, 2, 0, 0);
	if (server == NULL) {
		fprintf(stderr, "An error occurred while trying to create an ENet server host.\n");
		exit(EXIT_FAILURE);
	}

	return server;
}

ENetEvent serviceHost(ENetHost *host) {
	ENetEvent event;

	int e = 0;
	do {
		  e = enet_host_service(host, &event, 1); // 0 (= don't block) will hog an entire CPU core at 100%
	} while (e <= 0 || (event.type == ENET_EVENT_TYPE_RECEIVE && event.packet->dataLength == 0));

	return event;
}
*/
import "C"

import (
	"errors"
	"time"
)

func NewHost(laddr string, lport int) (h *Host, err error) {
	enetHost := C.initServer(C.CString(laddr), C.int(lport))
	if enetHost == nil {
		err = errors.New("an error occured initializing the ENet host in C")
		return
	}

	h = &Host{
		enetHost: enetHost,
		peers:    map[*C.ENetPeer]*Peer{},
	}

	return
}

type Host struct {
	enetHost *C.ENetHost
	peers    map[*C.ENetPeer]*Peer
}

func (h *Host) Service() <-chan Event {
	events := make(chan Event)
	go func() {
		for {
			cEvent := C.serviceHost(h.enetHost)
			events <- h.eventFromCEvent(&cEvent)
			time.Sleep(1 * time.Millisecond) // TODO: not sure why, but without this, the server crashes
		}
	}()
	return events
}
