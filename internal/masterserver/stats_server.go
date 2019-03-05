package masterserver

import (
	"fmt"
	"log"
	"strings"

	"github.com/sauerbraten/maitred/pkg/protocol"

	"github.com/sauerbraten/waiter/pkg/definitions/role"
)

type StatsServer struct {
	*MasterServer
	onSuccess func(reqID uint32)
	onFailure func(reqID uint32, reason string)
}

func NewStatsMaster(addr string, listenPort int, onSuccess func(uint32), onFailure func(uint32, string), onReconnect func()) (*StatsServer, <-chan string, error) {
	ms, inc, err := NewMaster(addr, listenPort, nil, role.None, onReconnect)
	if err != nil {
		return nil, nil, err
	}

	return &StatsServer{
		MasterServer: ms,
		onSuccess:    onSuccess,
		onFailure:    onFailure,
	}, inc, nil
}

func (s *StatsServer) Handle(msg string) {
	cmd := strings.Split(msg, " ")[0]
	args := strings.TrimSpace(msg[len(cmd):])

	switch cmd {
	case protocol.FailStats:
		s.handleFailStats(args)

	case protocol.SuccStats:
		s.handleSuccStats(args)

	default:
		s.MasterServer.Handle(msg)
	}
}

func (s *StatsServer) handleFailStats(args string) {
	r := strings.NewReader(strings.TrimSpace(args))

	var reqID uint32
	_, err := fmt.Fscanf(r, "%d", &reqID)
	if err != nil {
		log.Printf("malformed %s message from stats server: '%s': %v", protocol.FailStats, args, err)
		return
	}
	reason := args[len(args)-r.Len():] // unread portion of args
	reason = strings.TrimSpace(reason)
	s.onFailure(reqID, reason)
}

func (s *StatsServer) handleSuccStats(args string) {
	var reqID uint32
	_, err := fmt.Sscanf(args, "%d", &reqID)
	if err != nil {
		log.Printf("malformed %s message from stats server: '%s': %v", protocol.SuccStats, args, err)
		return
	}
	s.onSuccess(reqID)
}
