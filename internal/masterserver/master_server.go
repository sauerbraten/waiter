package masterserver

import (
	"bufio"
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

	onReconnect func() // executed when the game server reconnects to the remote master server
}

// New connects to the specified master server. Bans received from the master server are added to the given ban manager.
func NewMaster(addr string, listenPort int, bans *bans.BanManager, authRole role.ID, onReconnect func()) (*MasterServer, <-chan string, error) {
	raddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return nil, nil, fmt.Errorf("master (%s): error resolving server address (%s): %v", raddr, addr, err)
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

		onReconnect: onReconnect,
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
		return fmt.Errorf("master (%s): error connecting to master server: %v", ms.raddr, err)
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
			log.Printf("master (%s): EOF while scanning input", ms.raddr)
			if !ms.pingFailed {
				ms.reconnect(io.EOF)
			}
		}
	}()

	go func() {
		for msg := range ms.authOut {
			err := ms.Send(msg)
			if err != nil {
				log.Printf("master (%s): remote auth: %v", ms.raddr, err)
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
		time.Sleep(time.Duration(try) * 30 * time.Second)
		log.Printf("master (%s): trying to reconnect (attempt %d)", ms.raddr, try)

		err = ms.connect()
		try++
	}

	if err == nil {
		log.Printf("master (%s): reconnected successfully", ms.raddr)
		ms.onReconnect()
	} else {
		log.Printf("master (%s): could not reconnect: %v", ms.raddr, err)
	}
}

func (ms *MasterServer) Register() {
	if ms.pingFailed {
		return
	}
	log.Printf("master (%s): registering", ms.raddr)
	err := ms.Send("%s %d", registerServer, ms.listenPort)
	if err != nil {
		log.Printf("master (%s): registration failed: %v", ms.raddr, err)
		return
	}
}

func (ms *MasterServer) Send(format string, args ...interface{}) error {
	if ms.conn == nil {
		return fmt.Errorf("master (%s): not connected", ms.raddr)
	}

	err := ms.conn.SetWriteDeadline(time.Now().Add(10 * time.Millisecond))
	if err != nil {
		log.Printf("master (%s): write failed: %v", ms.raddr, err)
		return err
	}
	_, err = ms.conn.Write([]byte(fmt.Sprintf(format+"\n", args...)))
	if err != nil {
		log.Printf("master (%s): write failed: %v", ms.raddr, err)
	}
	return err
}

func (ms *MasterServer) Handle(msg string) {
	cmd := strings.Split(msg, " ")[0]
	args := strings.TrimSpace(msg[len(cmd):])

	switch cmd {
	case registrationSuccessful:
		log.Printf("master (%s): registration succeeded", ms.raddr)

	case registrationFailed:
		log.Printf("master (%s): registration failed: %v", ms.raddr, args)
		if args == "failed pinging server" {
			log.Printf("master (%s): disabling reconnecting", ms.raddr)
			ms.pingFailed = true // stop trying
		}

	case clearBans:
		ms.bans.ClearGlobalBans()

	case addBan:
		ms.handleAddGlobalBan(args)

	case authChallenge, authFailed, authSuccesful:
		ms.authInc <- msg

	default:
		log.Printf("master (%s): received and not handled: %v", ms.raddr, msg)
	}
}

func (ms *MasterServer) handleAddGlobalBan(args string) {
	var ip string
	_, err := fmt.Sscanf(args, "%s", &ip)
	if err != nil {
		log.Printf("master (%s): malformed %s message from game server: '%s': %v", ms.raddr, addBan, args, err)
		return
	}

	network := ips.GetSubnet(ip)

	ms.bans.AddBan(network, fmt.Sprintf("banned by master server (%s)", ms.raddr), time.Time{}, true)
}
