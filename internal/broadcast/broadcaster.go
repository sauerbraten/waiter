package broadcast

import (
	"time"

	"github.com/sauerbraten/waiter/internal/client"
	"github.com/sauerbraten/waiter/internal/enet"
	"github.com/sauerbraten/waiter/internal/protocol/packet"
)

type broadcaster struct {
	clientManager      *client.ClientManager
	flags              enet.PacketFlag
	channel            uint8
	ticker             *time.Ticker
	getBroadcastPacket func(client *client.Client) *packet.Packet
	combinedPacket     *packet.Packet
}

func Forever(interval time.Duration, flags enet.PacketFlag, channel uint8, clientManager *client.ClientManager, getBroadcastPacket func(client *client.Client) *packet.Packet) {
	b := broadcaster{
		clientManager:      clientManager,
		flags:              flags,
		channel:            channel,
		ticker:             time.NewTicker(interval),
		getBroadcastPacket: getBroadcastPacket,
		combinedPacket:     packet.New(),
	}

	b.run()
}

func (b *broadcaster) run() {
	for range b.ticker.C {
		b.flush()
	}
}

func (b *broadcaster) flush() {
	lengths := map[*client.Client]int{}

	b.clientManager.RLock()
	defer b.clientManager.RUnlock()

	b.clientManager.ForEach(func(c *client.Client) {
		if !c.Joined {
			return
		}

		bp := b.getBroadcastPacket(c)

		if bp.Len() == 0 {
			return
		}

		lengths[c] = bp.Len()
		b.combinedPacket.Put(bp)

		bp.Clear()
	})

	if b.combinedPacket.Len() == 0 {
		return
	}

	// double packet
	b.combinedPacket.Put(b.combinedPacket)

	b.clientManager.ForEach(func(c *client.Client) {
		if !c.Joined {
			return
		}

		length := lengths[c]
		b.combinedPacket.Seek(length)

		if b.combinedPacket.Len() == length*2 {
			// only the client's own packages are in the master packet
			return
		}

		c.Send(b.flags, b.channel, b.combinedPacket.SubPacket((b.combinedPacket.Len()/2)-length))
	})

	b.combinedPacket.Clear()
}
