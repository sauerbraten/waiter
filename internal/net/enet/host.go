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
	//enet_address_set_host(&address, addr);
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
		e = enet_host_service(host, &event, 2); // don't block
	} while (e <= 0 || (event.type == ENET_EVENT_TYPE_RECEIVE && event.packet->dataLength == 0));

	// TODO: investigate why we are receiving empty packets...

	return event;
}
*/
import "C"

import (
	"errors"
)

var peers map[*C.ENetPeer]*Peer = map[*C.ENetPeer]*Peer{}

func NewHost(laddr string, lport int) (h *Host, err error) {
	enetHost := C.initServer(C.CString(laddr), C.int(lport))
	if enetHost == nil {
		err = errors.New("an error occured running the C code")
		return
	}

	h = &Host{enetHost: enetHost}

	return
}

type Host struct {
	enetHost *C.ENetHost
}

func (h *Host) Service() Event {
	cEvent := C.serviceHost(h.enetHost)
	event := eventFromCEvent(&cEvent)
	return event
}

func (h *Host) Flush() {
	C.enet_host_flush(h.enetHost)
}
