package main

import (
	"errors"
	"sync"
	"time"

	"github.com/sauerbraten/waiter/internal/definitions/nmc"
	"github.com/sauerbraten/waiter/internal/protocol/enet"
	"github.com/sauerbraten/waiter/internal/protocol/packet"
)

type sendFunc func(channel uint8, flags enet.PacketFlag, payload []byte)

// Relay handles subscribing and unsubcribing to topics.
type Relay struct {
	μ sync.Mutex

	incPositionsNotifs chan uint32              // channel on which clients notify the broker about new packets
	incPositions       map[uint32]<-chan []byte // clients' update channels by topic
	positions          map[uint32][]byte

	incClientPacketsNotifs chan uint32
	incClientPackets       map[uint32]<-chan []byte
	clientPackets          map[uint32][]byte

	send map[uint32]sendFunc
}

func NewRelay() *Relay {
	return &Relay{
		incPositionsNotifs: make(chan uint32),
		incPositions:       map[uint32]<-chan []byte{},
		positions:          map[uint32][]byte{},

		incClientPacketsNotifs: make(chan uint32),
		incClientPackets:       map[uint32]<-chan []byte{},
		clientPackets:          map[uint32][]byte{},

		send: map[uint32]sendFunc{},
	}
}

func (r *Relay) loop() {
	t := time.Tick(33 * time.Millisecond)
	for {
		select {
		case <-t:
			if len(r.positions) > 0 {
				// publish positions
				r.flush(
					r.positions,
					func(cn uint32, pkt []byte) []byte {
						return nil
					},
					0,
					enet.PACKET_FLAG_NONE,
				)

				// clear positions
				for cn := range r.positions {
					delete(r.positions, cn)
				}
			}

			if len(r.clientPackets) > 0 {
				// publish client packets
				r.flush(
					r.clientPackets,
					func(cn uint32, pkt []byte) []byte {
						p := packet.Encode(nmc.Client, cn)
						p.PutUint(uint32(len(pkt)))
						return p
					},
					1,
					enet.PACKET_FLAG_RELIABLE,
				)

				// clear client packets
				for cn := range r.clientPackets {
					delete(r.clientPackets, cn)
				}
			}

		case cn := <-r.incPositionsNotifs:
			r.receive(cn, r.incPositions, func(pos []byte) {
				r.positions[cn] = pos
			})

		case cn := <-r.incClientPacketsNotifs:
			r.receive(cn, r.incClientPackets, func(pkt []byte) {
				r.clientPackets[cn] = append(r.clientPackets[cn], pkt...)
			})
		}
	}
}

// Subscribe returns a new channel on which to receive updates on a certain topic.
// Subscribe makes sure the topic exists by creating it if neccessary. When a new
// topic was created, a corresponding publisher is returned, otherwise newPublisher
// is nil.
func (r *Relay) AddClient(cn uint32, sf sendFunc) (positions *Publisher, packets *Publisher) {
	r.μ.Lock()
	defer r.μ.Unlock()

	if _, ok := r.send[cn]; ok {
		// client is already serviced
		return nil, nil
	}

	r.send[cn] = sf

	positions, posCh := newPublisher(cn, r.incPositionsNotifs)
	r.incPositions[cn] = posCh

	packets, pktCh := newPublisher(cn, r.incClientPacketsNotifs)
	r.incClientPackets[cn] = pktCh

	return
}

// Unsubscribe removes the specified channel from the topic, meaning there will be no more messages sent to updates.
// Unsubscribe will close updates.
func (r *Relay) RemoveClient(cn uint32) error {
	r.μ.Lock()
	defer r.μ.Unlock()

	if _, ok := r.send[cn]; !ok {
		return errors.New("no such client")
	}

	delete(r.incPositions, cn)
	delete(r.positions, cn)
	delete(r.incClientPackets, cn)
	delete(r.clientPackets, cn)
	delete(r.send, cn)

	return nil
}

func (r *Relay) BroadcastAfterPosition(cn uint32, p []byte) {
	r.μ.Lock()
	defer r.μ.Unlock()

	if pos := r.positions[cn]; pos != nil {
		for _cn, send := range r.send {
			if _cn == cn {
				continue
			}
			send(0, enet.PACKET_FLAG_NO_ALLOCATE, pos)
		}
		delete(r.positions, cn)
	}

	for _cn, send := range r.send {
		if _cn == cn {
			continue
		}
		send(0, enet.PACKET_FLAG_NO_ALLOCATE, p)
	}
}

func (r *Relay) receive(cn uint32, from map[uint32]<-chan []byte, process func(upd []byte)) {
	r.μ.Lock()
	defer r.μ.Unlock()

	ch, ok := from[cn]
	if !ok {
		// ignore clients that were already removed
		return
	}

	p, ok := <-ch
	if ok {
		process(p)
	}
}

func (r *Relay) flush(packets map[uint32][]byte, prefix func(uint32, []byte) []byte, channel uint8, flags enet.PacketFlag) {
	r.μ.Lock()
	defer r.μ.Unlock()

	if len(packets) == 0 || len(r.send) < 2 {
		return
	}

	order := make([]uint32, 0, len(r.send))
	lengths := map[uint32]int{}
	combined := make([]byte, 0, 2*len(packets)*20)

	for cn := range r.send {
		order = append(order, cn)
		pkt := packets[cn]
		if pkt == nil {
			continue
		}
		prfx := prefix(cn, pkt)
		lengths[cn] = len(prfx) + len(pkt)
		combined = append(append(combined, prfx...), pkt...)
	}

	if len(combined) == 0 {
		return
	}

	combined = append(combined, combined...)

	offset := 0
	for _, cn := range order {
		l := lengths[cn]
		offset += l
		p := combined[offset : (len(combined)/2)-l+offset]
		r.send[cn](channel, flags, p)
	}
}

// Publisher provides methods to send updates to all subscribers of a certain topic.
type Publisher struct {
	cn          uint32
	notifyRelay chan<- uint32
	updates     chan<- []byte
}

func newPublisher(cn uint32, notifyRelay chan<- uint32) (*Publisher, <-chan []byte) {
	updates := make(chan []byte)

	p := &Publisher{
		cn:          cn,
		notifyRelay: notifyRelay,
		updates:     updates,
	}

	return p, updates
}

// Publish notifies p's broker that there is an update on p's topic and blocks until the broker received the notification.
// Publish then blocks until the broker received the update. Calling Publish() after Close() returns immediately. Use p's
// Stop channel to know when the broker stopped listening.
func (p *Publisher) Publish(args ...interface{}) {
	p.notifyRelay <- p.cn
	p.updates <- packet.Encode(args...)
}

// Close tells the broker there will be no more updates coming from p. Calling Publish() after Close() returns immediately.
// Calling Close() makes the broker unsubscribe all subscribers and telling them updates on the topic have ended.
func (p *Publisher) Close() {
	close(p.updates)
}
