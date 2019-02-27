package masterserver

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/sauerbraten/chef/pkg/ips"

	"github.com/sauerbraten/waiter/pkg/bans"
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

	conn             *net.TCPConn
	lastRegistration time.Time

	requestChallengeCallbacks map[uint32]func(challenge string)
	confirmAnswerCallbacks    map[uint32]func(sucess bool)
}

// New connects to the specified master server. Bans received from the master server are added to the given ban manager.
func New(addr string, listenPort int, bans *bans.BanManager) (*MasterServer, <-chan string, error) {
	raddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return nil, nil, fmt.Errorf("error resolving master server address (%s): %v", addr, err)
	}

	ms := &MasterServer{
		raddr:      raddr,
		listenPort: listenPort,
		bans:       bans,

		lastRegistration: time.Now().Add(-2 * time.Hour),

		requestChallengeCallbacks: map[uint32]func(string){},
		confirmAnswerCallbacks:    map[uint32]func(sucess bool){},
	}

	err = ms.connect()
	if err != nil {
		return nil, nil, err
	}

	sc := bufio.NewScanner(ms.conn)
	inc := make(chan string)

	go func() {
		for sc.Scan() {
			inc <- sc.Text()
		}
		if err := sc.Err(); err != nil {
			log.Println(err)
		} else {
			log.Println("EOF from master server")
			ms.reconnect(io.EOF)
		}
	}()

	return ms, inc, nil
}

func (ms *MasterServer) connect() error {
	conn, err := net.DialTCP("tcp", nil, ms.raddr)
	if err != nil {
		return fmt.Errorf("error connecting to master server: %v", err)
	}

	ms.conn = conn
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
	log.Println("registering at master server")
	err := ms.Request("%s %d", registerServer, ms.listenPort)
	if err != nil {
		log.Println("registering at master server failed:", err)
		return
	}
	ms.lastRegistration = time.Now()
}

func (ms *MasterServer) RequestAuthChallenge(requestID uint32, name string, callback func(challenge string)) error {
	err := ms.Request("%s %d %s", requestAuthChallenge, requestID, name)
	if err == nil {
		ms.requestChallengeCallbacks[requestID] = callback
	}
	return err
}

func (ms *MasterServer) ConfirmAuthAnswer(requestID uint32, answer string, callback func(bool)) error {
	err := ms.Request("%s %d %s", confirmAuthAnswer, requestID, answer)
	if err == nil {
		ms.confirmAnswerCallbacks[requestID] = callback
	}
	return err
}

func (ms *MasterServer) Request(format string, args ...interface{}) error {
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
	msgParts := strings.Split(msg, " ")
	cmd := msgParts[0]
	args := msgParts[1:]

	switch cmd {
	case registrationSuccessful:
		log.Println("master server registration succeeded")

	case registrationFailed:
		log.Println("master server registration failed:", strings.Join(args, " "))

	case clearBans:
		ms.bans.ClearGlobalBans()

	case addBan:
		ms.handleAddGlobalBan(args)

	case authChallenge:
		ms.handleAuthChallenge(args)

	case authFailed:
		ms.handleAuthResult(false, authFailed, args)

	case authSuccesful:
		ms.handleAuthResult(true, authSuccesful, msgParts[1:])

	default:
		log.Println("received from master:", msg)
	}
}

func (ms *MasterServer) handleAddGlobalBan(args []string) {
	if len(args) != 1 {
		log.Printf("malformed '%s' message from master server: '%s'\n", addBan, strings.Join(args, " "))
		return
	}

	ipString := args[0]
	network := ips.GetSubnet(ipString)

	ms.bans.AddBan(network, "banned by master server", time.Time{}, true)
}

func (ms *MasterServer) handleAuthChallenge(args []string) {
	if len(args) != 2 {
		log.Printf("malformed '%s' message from master server: '%s'\n", authChallenge, strings.Join(args, " "))
		return
	}

	_requestID, err := strconv.ParseUint(args[0], 10, 32)
	if err != nil {
		log.Printf("malformed request ID '%s' in '%s' message from master server: '%s'\n", args[0], authChallenge, strings.Join(args, " "))
		return
	}
	requestID := uint32(_requestID)

	challenge := args[1]

	if callback, ok := ms.requestChallengeCallbacks[requestID]; ok {
		callback(challenge)
	} else {
		log.Println("unsolicited auth challenge from master server")
	}
}

func (ms *MasterServer) handleAuthResult(sucess bool, cmd string, args []string) {
	if len(args) != 1 {
		log.Printf("malformed '%s' message from master server: '%s'\n", cmd, strings.Join(args, " "))
		return
	}

	_requestID, err := strconv.ParseUint(args[0], 10, 32)
	if err != nil {
		log.Printf("malformed request ID '%s' in '%s' message from master server: '%s'\n", args[0], cmd, strings.Join(args, " "))
		return
	}
	requestID := uint32(_requestID)

	if callback, ok := ms.confirmAnswerCallbacks[requestID]; ok {
		callback(sucess)
	} else {
		log.Println("unsolicited auth confirmation from master server")
	}
}
