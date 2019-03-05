package masterserver

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"time"

	"github.com/sauerbraten/chef/pkg/ips"

	"github.com/sauerbraten/waiter/pkg/auth"
	"github.com/sauerbraten/waiter/pkg/bans"
	"github.com/sauerbraten/waiter/pkg/definitions/role"
)

// master server protocol constants
// exported constants can be sent to the master server
// unexported constants can only be received and are handled inside this package
const (
	registerServer         = "regserv"
	registrationSuccessful = "succreg"
	registrationFailed     = "failreg"

	addBan    = "addgban"
	clearBans = "cleargbans"

	requestAuthChallenge = "reqauth"
	authChallenge        = "chalauth"
	confirmAuthAnswer    = "confauth"
	authSuccesful        = "succauth"
	authFailed           = "failauth"
)

type MasterServer struct {
	raddr      *net.TCPAddr
	listenPort int
	bans       *bans.BanManager

	conn       *net.TCPConn
	inc        chan<- string
	pingFailed bool

	*auth.RemoteProvider
	authInc chan<- string
	authOut <-chan string
}

// New connects to the specified master server. Bans received from the master server are added to the given ban manager.
func NewMaster(addr string, listenPort int, bans *bans.BanManager, authRole role.ID) (*MasterServer, <-chan string, error) {
	raddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return nil, nil, fmt.Errorf("error resolving master server address (%s): %v", addr, err)
	}

	inc := make(chan string)
	authInc, authOut := make(chan string), make(chan string)

	ms := &MasterServer{
		raddr:      raddr,
		listenPort: listenPort,
		bans:       bans,

		inc: inc,

		RemoteProvider: auth.NewRemoteProvider(authInc, authOut, authRole),
		authInc:        authInc,
		authOut:        authOut,
	}

	err = ms.connect()
	if err != nil {
		return nil, nil, err
	}

	return ms, inc, nil
}

func (ms *MasterServer) connect() error {
	conn, err := net.DialTCP("tcp", nil, ms.raddr)
	if err != nil {
		return fmt.Errorf("error connecting to master server: %v", err)
	}

	ms.conn = conn

	sc := bufio.NewScanner(ms.conn)

	go func() {
		for sc.Scan() {
			ms.inc <- sc.Text()
		}
		if err := sc.Err(); err != nil {
			log.Println(err)
		} else {
			log.Println("EOF from master server", ms.raddr)
			ms.reconnect(io.EOF)
		}
	}()

	go func() {
		for msg := range ms.authOut {
			err := ms.Send(msg)
			if err != nil {
				log.Printf("remote auth (%s): %v", ms.raddr, err)
			}
		}
	}()

	ms.Register()

	return nil
}

func (ms *MasterServer) reconnect(err error) {
	ms.conn = nil

	try, maxTries := 1, 10
	for err != nil && try <= maxTries {
		time.Sleep(time.Duration(try) * time.Minute)
		log.Printf("trying to reconnect (attempt %d)", try)

		err = ms.connect()
		try++
	}

	if err == nil {
		log.Println("reconnected to master server")
	} else {
		log.Println("could not reconnect to master server:", err)
	}
}

func (ms *MasterServer) Register() {
	if ms.pingFailed {
		return
	}
	log.Println("registering at master server")
	err := ms.Send("%s %d", registerServer, ms.listenPort)
	if err != nil {
		log.Println("registering at master server failed:", err)
		return
	}
}

func (ms *MasterServer) Send(format string, args ...interface{}) error {
	if ms.conn == nil {
		return errors.New("not connected to master server")
	}

	_, err := ms.conn.Write([]byte(fmt.Sprintf(format+"\n", args...)))
	if err != nil {
		log.Println("write to master failed:", err)
	}
	return err
}

func (ms *MasterServer) Handle(msg string) {
	cmd := strings.Split(msg, " ")[0]
	args := strings.TrimSpace(msg[len(cmd):])

	switch cmd {
	case registrationSuccessful:
		log.Println("master server registration succeeded")

	case registrationFailed:
		log.Println("master server registration failed:", args)
		if args == "failed pinging server" {
			log.Println("disabling reconnecting")
			ms.pingFailed = true // stop trying
		}

	case clearBans:
		ms.bans.ClearGlobalBans()

	case addBan:
		ms.handleAddGlobalBan(args)

	case authChallenge, authFailed, authSuccesful:
		ms.authInc <- msg

	default:
		log.Println("received from master:", msg)
	}
}

func (ms *MasterServer) handleAddGlobalBan(args string) {
	var ip string
	_, err := fmt.Sscanf(args, "%s", &ip)
	if err != nil {
		log.Printf("malformed %s message from game server: '%s': %v", addBan, args, err)
		return
	}

	network := ips.GetSubnet(ip)

	ms.bans.AddBan(network, "banned by master server", time.Time{}, true)
}
