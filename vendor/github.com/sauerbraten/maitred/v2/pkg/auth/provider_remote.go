package auth

import (
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/sauerbraten/maitred/v2/pkg/protocol"
	"github.com/sauerbraten/waiter/pkg/protocol/role"
)

type RemoteProvider struct {
	// for communication with master
	inc <-chan string
	out chan<- string

	rol                       role.ID // all successful auths will get this role in the ConfirmAnswer callback
	ids                       *protocol.IDCycle
	lastActivity              map[uint32]time.Time
	requestChallengeCallbacks map[uint32]func(uint32, string, error)
	confirmAnswerCallbacks    map[uint32]func(role.ID, error)
}

func NewRemoteProvider(inc <-chan string, out chan<- string, rol role.ID) *RemoteProvider {
	rp := &RemoteProvider{
		inc: inc,
		out: out,

		rol:                       rol,
		ids:                       new(protocol.IDCycle),
		lastActivity:              map[uint32]time.Time{},
		requestChallengeCallbacks: map[uint32]func(uint32, string, error){},
		confirmAnswerCallbacks:    map[uint32]func(role.ID, error){},
	}
	go rp.run()
	return rp
}

func (p *RemoteProvider) run() {
	for {
		select {
		case msg := <-p.inc:
			p.handle(msg)
		case <-time.Tick(10 * time.Second):
			timedOut := []uint32{}
			for reqID, lastActive := range p.lastActivity {
				if time.Since(lastActive) > 30*time.Second {
					timedOut = append(timedOut, reqID)
				}
			}
			for _, reqID := range timedOut {
				if callback, ok := p.requestChallengeCallbacks[reqID]; ok {
					callback(reqID, "", errors.New("timed out waiting for challenge"))
				}
				delete(p.requestChallengeCallbacks, reqID)
				if callback, ok := p.confirmAnswerCallbacks[reqID]; ok {
					callback(role.None, errors.New("timed out waiting for confirmation"))
				}
				delete(p.confirmAnswerCallbacks, reqID)
				delete(p.lastActivity, reqID)
			}
		}
	}
}

func (p *RemoteProvider) handle(msg string) {
	cmd := strings.Split(msg, " ")[0]
	args := msg[len(cmd):]

	switch cmd {
	case protocol.ChalAuth:
		p.handleChalAuth(args)

	case protocol.SuccAuth:
		p.handleSuccAuth(args)

	case protocol.FailAuth:
		p.handleFailAuth(args)

	default:
		log.Println("unhandled message from master:", msg)
	}
}

func (p *RemoteProvider) GenerateChallenge(name string, callback func(reqID uint32, chal string, err error)) {
	reqID := p.ids.Next()
	p.out <- fmt.Sprintf("%s %d %s", protocol.ReqAuth, reqID, name)
	p.requestChallengeCallbacks[reqID] = callback
	p.lastActivity[reqID] = time.Now()
}

func (p *RemoteProvider) ConfirmAnswer(reqID uint32, answ string, callback func(role.ID, error)) {
	p.out <- fmt.Sprintf("%s %d %s", protocol.ConfAuth, reqID, answ)
	p.confirmAnswerCallbacks[reqID] = callback
	p.lastActivity[reqID] = time.Now()
}

func (p *RemoteProvider) handleChalAuth(args string) {
	var reqID uint32
	var chal string
	_, err := fmt.Sscanf(args, "%d %s", &reqID, &chal)
	if err != nil {
		log.Printf("malformed %s message from remote provider: '%s': %v", protocol.ChalAuth, args, err)
		return
	}

	defer delete(p.requestChallengeCallbacks, reqID)

	if callback, ok := p.requestChallengeCallbacks[reqID]; ok {
		callback(reqID, chal, nil)
	} else {
		log.Printf("unsolicited %s message from remote provider: '%s'", protocol.ChalAuth, args)
	}
}

func (p *RemoteProvider) handleSuccAuth(args string) {
	var reqID uint32
	_, err := fmt.Sscanf(args, "%d", &reqID)
	if err != nil {
		log.Printf("malformed %s message from remote provider: '%s': %v", protocol.SuccAuth, args, err)
		return
	}

	defer delete(p.confirmAnswerCallbacks, reqID)

	if callback, ok := p.confirmAnswerCallbacks[reqID]; ok {
		callback(p.rol, nil)
	} else {
		log.Printf("unsolicited %s message from remote provider: '%s'", protocol.SuccAuth, args)
	}
}

func (p *RemoteProvider) handleFailAuth(args string) {
	var reqID uint32
	_, err := fmt.Sscanf(args, "%d", &reqID)
	if err != nil {
		log.Printf("malformed %s message from remote provider: '%s': %v", protocol.FailAuth, args, err)
		return
	}

	defer delete(p.confirmAnswerCallbacks, reqID)

	if callback, ok := p.confirmAnswerCallbacks[reqID]; ok {
		callback(role.None, errors.New("remote auth provider signalled failure"))
	} else {
		if _, ok := p.requestChallengeCallbacks[reqID]; ok {
			delete(p.requestChallengeCallbacks, reqID)
		} else {
			log.Printf("unsolicited %s message from remote provider: '%s'", protocol.FailAuth, args)
		}
	}
}
