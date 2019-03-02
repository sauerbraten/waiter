package stats

import (
	"log"
	"strings"

	"github.com/sauerbraten/waiter/internal/masterserver"
)

type Server struct {
	masterserver.MasterServer
}

func New(addr string, listenPort int) (*Server, <-chan string, error) {
	ms, inc, err := masterserver.New(addr, listenPort, nil)
	if err != nil {
		return nil, nil, err
	}

	return &Server{
		MasterServer: *ms,
	}, inc, nil
}

func (s *Server) Handle(msg string) {
	const (
		stats     = "stats"
		failStats = "failstats"
		succStats = "succstats"
	)

	cmd := strings.Split(msg, " ")[0]
	args := strings.TrimSpace(msg[len(cmd):])

	switch cmd {
	case failStats:
		log.Println("stats reporting failed:", args)

	case succStats:
		log.Println("stats reporting succeeded:", args)

	default:
		s.MasterServer.Handle(msg)
	}
}
