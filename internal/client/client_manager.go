package client

import (
	"sync"

	"github.com/sauerbraten/waiter/internal/enet"
	"github.com/sauerbraten/waiter/internal/protocol/packet"
)

type ClientManager struct {
	sync.RWMutex
	clients []*Client
}

// Links an ENet peer to a client object. If no unused client object can be found, a new one is created and added to the global set of clients.
func (cm *ClientManager) Add(peer *enet.Peer) *Client {
	cm.Lock()
	defer cm.Unlock()

	// re-use unused client object with low cn
	for _, c := range cm.clients {
		if !c.InUse {
			c.InUse = true
			return c
		}
	}

	c := newClient(int32(len(cm.clients)), peer, cm)

	cm.clients = append(cm.clients, c)

	return c
}

func (cm *ClientManager) GetClientByCN(cn int32) *Client {
	return cm.clients[cn]
}

func (cm *ClientManager) ForEach(do func(client *Client)) {
	cm.RLock()

	for _, c := range cm.clients {
		do(c)
	}

	cm.RUnlock()
}

// Sends a packet to all clients currently in use.
func (cm *ClientManager) Broadcast(flags enet.PacketFlag, channel uint8, p *packet.Packet) {
	for _, c := range cm.clients {
		if !c.InUse {
			continue
		}
		c.Send(flags, channel, p)
	}
}

func (cm *ClientManager) NumClients() int {
	return len(cm.clients)
}

// Returns the number of connected clients.
func (cm *ClientManager) NumberOfClientsConnected() (n int) {
	for _, c := range cm.clients {
		if !c.InUse {
			continue
		}
		n++
	}
	return
}
