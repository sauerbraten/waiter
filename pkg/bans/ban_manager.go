package bans

import (
	"log"
	"net"
	"sync"
	"time"

	"github.com/sauerbraten/jsonfile"
)

type BanManager struct {
	µ    sync.Mutex
	bans map[string]*Ban
}

func New(bans ...*Ban) *BanManager {
	bm := &BanManager{
		bans: map[string]*Ban{},
	}

	for _, ban := range bans {
		bm.addBan(ban)
	}

	return bm
}

func FromFile(fileName string) (*BanManager, error) {
	var bansFromFile []*Ban
	err := jsonfile.ParseFile(fileName, &bansFromFile)
	if err != nil {
		return nil, err
	}

	return New(bansFromFile...), nil
}

func (bm *BanManager) AddBan(network *net.IPNet, reason string, expiryDate time.Time, global bool) {
	bm.µ.Lock()
	defer bm.µ.Unlock()

	bm.addBan(&Ban{
		Network:    network,
		Reason:     reason,
		ExpiryDate: expiryDate,
		Global:     global,
	})
}

// not safe for concurrent use
func (bm *BanManager) addBan(ban *Ban) {
	// never overwrite global bans with non-global bans
	if existing, ok := bm.bans[ban.Network.String()]; ok && existing.Global && !ban.Global {
		return
	}

	log.Println("added ban:", ban)

	bm.bans[ban.Network.String()] = ban
}

func (bm *BanManager) ClearGlobalBans() {
	bm.µ.Lock()
	defer bm.µ.Unlock()

	for cidr, ban := range bm.bans {
		if ban.Global {
			bm.clearBan(cidr)
		}
	}
}

func (bm *BanManager) ClearBan(cidr string) {
	bm.µ.Lock()
	defer bm.µ.Unlock()

	bm.clearBan(cidr)
}

// not safe for concurrent use
func (bm *BanManager) clearBan(cidr string) {
	delete(bm.bans, cidr)
}

func (bm *BanManager) GetBan(ip net.IP) (ban *Ban, ok bool) {
	bm.µ.Lock()
	defer bm.µ.Unlock()

	for cidr, b := range bm.bans {
		if b.Network.Contains(ip) {
			// check if the ban already expired
			if !b.ExpiryDate.IsZero() && b.ExpiryDate.Before(time.Now()) {
				bm.clearBan(cidr)
			} else {
				ban, ok = b, true
			}
		}
	}

	return
}
