package bans

import (
	"encoding/json"
	"log"
	"net"
	"os"
	"time"
)

type BanManager struct {
	bans map[string]Ban
}

func (bm *BanManager) addBan(ban Ban) {
	// never overwrite global bans with non-global bans
	if existing, ok := bm.bans[ban.Network.String()]; ok && existing.IsGlobal && !ban.IsGlobal {
		return
	}

	log.Println("added ban:", ban)

	bm.bans[ban.Network.String()] = ban
}
func (bm *BanManager) AddBan(network net.IPNet, reason string, expiryDate time.Time, isGlobal bool) {
	bm.addBan(Ban{
		Network:    network,
		Reason:     reason,
		ExpiryDate: expiryDate,
		IsGlobal:   isGlobal,
	})
}

func FromFile(fileName string) (*BanManager, error) {
	file, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}

	var bansFromFile []Ban
	dec := json.NewDecoder(file)

	err = dec.Decode(&bansFromFile)
	if err != nil {
		return nil, err
	}

	bm := &BanManager{
		bans: map[string]Ban{},
	}

	for _, ban := range bansFromFile {
		bm.addBan(ban)
	}

	return bm, nil
}

func (bm *BanManager) ClearGlobalBans() {
	for cidr, ban := range bm.bans {
		if ban.IsGlobal {
			bm.ClearBan(cidr)
		}
	}
}

func (bm *BanManager) ClearBan(cidr string) {
	delete(bm.bans, cidr)
}

func (bm *BanManager) GetBan(ip net.IP) (ban Ban, ok bool) {
	for cidr, ban := range bm.bans {
		if ban.Network.Contains(ip) {
			// check if the ban already expired
			if !ban.IsGlobal && ban.ExpiryDate.Before(time.Now()) {
				bm.ClearBan(cidr)
				break
			} else {
				return ban, true
			}
		}
	}

	return
}
