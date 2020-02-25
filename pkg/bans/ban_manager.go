package bans

import (
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/sauerbraten/chef/pkg/ips"
	"github.com/sauerbraten/jsonfile"
	"github.com/sauerbraten/maitred/v2/pkg/protocol"
)

type BanManager struct {
	µ    sync.Mutex
	bans map[string]map[string]*Ban // domain -> cidr -> ban
}

func New(bans ...*Ban) *BanManager {
	bm := &BanManager{
		bans: map[string]map[string]*Ban{},
	}

	for _, ban := range bans {
		bm.addBan(ban)
	}

	return bm
}

func FromFile(fileName string) ([]*Ban, error) {
	var bansFromFile []*Ban
	err := jsonfile.ParseFile(fileName, &bansFromFile)
	if err != nil {
		return nil, err
	}

	return bansFromFile, nil
}

func (bm *BanManager) AddBan(network *net.IPNet, reason string, expiryDate time.Time, domain string) {
	bm.µ.Lock()
	defer bm.µ.Unlock()

	bm.addBan(&Ban{
		Network:    network,
		Reason:     reason,
		ExpiryDate: expiryDate,
		Domain:     domain,
	})
}

// not safe for concurrent use
func (bm *BanManager) addBan(ban *Ban) {
	// never overwrite global bans with non-global bans
	if existing, ok := bm.bans[ban.Domain][ban.Network.String()]; ok && existing.Domain == "" && ban.Domain != "" {
		return
	}

	bans, ok := bm.bans[ban.Domain]
	if !ok {
		bans = map[string]*Ban{}
		bm.bans[ban.Domain] = bans
	}
	bans[ban.Network.String()] = ban

	log.Println("added ban:", ban)
}

func (bm *BanManager) ClearBans(domain string) {
	bm.µ.Lock()
	defer bm.µ.Unlock()

	for cidr := range bm.bans[domain] {
		delete(bm.bans[domain], cidr)
	}
}

func (bm *BanManager) GetBan(ip net.IP) (ban *Ban, ok bool) {
	bm.µ.Lock()
	defer bm.µ.Unlock()

	for _, bans := range bm.bans {
		for cidr, ban := range bans {
			if ban.Network.Contains(ip) {
				// check if the ban already expired
				if !ban.ExpiryDate.IsZero() && ban.ExpiryDate.Before(time.Now()) {
					delete(bans, cidr)
				} else {
					return ban, true
				}
			}
		}
	}

	return nil, false
}

func (bm *BanManager) Handle(inc <-chan string) {
	for msg := range inc {
		cmd := strings.Split(msg, " ")[0]
		args := msg[len(cmd):]

		switch cmd {
		case protocol.ClearBans:
			bm.handleClearBans(args)

		case protocol.AddBan:
			bm.handleAddBan(args)

		default:
			log.Println("unhandled message in ban manager:", msg)
		}
	}
}

func (bm *BanManager) handleAddBan(args string) {
	var ip, domain string
	_, err := fmt.Sscanf(args, "%s %s", &ip, &domain)
	if err != nil {
		log.Printf("malformed %s message from master server: '%s': %v", protocol.AddBan, args, err)
		return
	}

	bm.AddBan(ips.GetSubnet(ip), fmt.Sprintf("gban from %s", domain), time.Time{}, domain)
}

func (bm *BanManager) handleClearBans(args string) {
	var domain string
	_, err := fmt.Sscanf(args, "%s", &domain)
	if err != nil {
		log.Printf("malformed %s message from master server: '%s': %v", protocol.ClearBans, args, err)
		return
	}

	bm.ClearBans(domain)
}
