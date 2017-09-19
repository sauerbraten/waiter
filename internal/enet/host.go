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

ENetEvent serviceHost(ENetHost *host, int timeout) {
	ENetEvent event;

	// Wait for an event (up to timeout milliseconds)
	int e = 0;

	do {
		e = enet_host_check_events(host, &event);
		if (e <= 0) {
			e = enet_host_service(host, &event, timeout);
		}
	} while (e < 0);

	return event;
}
*/
import "C"

import "errors"

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

func (h *Host) Service(timeout int) Event {
	var cEvent C.ENetEvent = C.serviceHost(h.enetHost, C.int(timeout))
	return eventFromCEvent(interface{}(&cEvent))
}

func (h *Host) Flush() {
	C.enet_host_flush(h.enetHost)
}
