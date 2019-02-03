package masterserver

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/sauerbraten/waiter/internal/bans"
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
	In  *bufio.Scanner
	Out *bufio.Writer

	Bans *bans.BanManager

	requestChallengeCallbacks map[uint32]func(challenge string)
	confirmAnswerCallbacks    map[uint32]func(sucess bool)
}

// New connects to the specified master server. Bans received from the master server are added to the given ban manager.
func New(addr string, listenPort int, bans *bans.BanManager) (ms *MasterServer, err error) {
	raddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("error resolving master server address (%s): %v", addr, err)
	}

	conn, err := net.DialTCP("tcp", nil, raddr)
	if err != nil {
		return nil, fmt.Errorf("error connecting to master server: %v", err)
	}

	ms = &MasterServer{
		In:   bufio.NewScanner(conn),
		Out:  bufio.NewWriter(conn),
		Bans: bans,

		requestChallengeCallbacks: map[uint32]func(string){},
		confirmAnswerCallbacks:    map[uint32]func(sucess bool){},
	}

	go ms.keepRegistered(listenPort)
	go ms.handleIncoming()

	return
}

func (ms *MasterServer) keepRegistered(listenPort int) {
	t := time.NewTicker(1 * time.Hour)
	for {
		log.Println("registering at master server")
		err := ms.Request("%s %d", registerServer, listenPort)
		if err != nil {
			log.Println("registering at master server failed:", err)
		}
		<-t.C
	}
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
	_, err := ms.Out.WriteString(fmt.Sprintf(format+"\n", args...))
	if err != nil {
		return err
	}

	err = ms.Out.Flush()

	return err
}

func (ms *MasterServer) handleIncoming() {
	for ms.In.Scan() {
		msg := ms.In.Text()
		msgParts := strings.Split(msg, " ")

		switch msgParts[0] {
		case registrationSuccessful:
			log.Println("master server registration succeeded")

		case registrationFailed:
			log.Println("master server registration failed:", strings.Join(msgParts[1:], " "))

		case clearBans:
			ms.Bans.ClearGlobalBans()

		case addBan:
			ms.handleAddGlobalBan(msgParts[1:])

		case authChallenge:
			ms.handleAuthChallenge(msgParts[1:])

		case authFailed:
			ms.handleAuthResult(false, authFailed, msgParts[1:])

		case authSuccesful:
			ms.handleAuthResult(true, authSuccesful, msgParts[1:])

		default:
			log.Println("received from master:", msg)
		}
	}

	if err := ms.In.Err(); err != nil {
		log.Println(err)
	}
}

func (ms *MasterServer) handleAddGlobalBan(args []string) {
	if len(args) != 1 {
		log.Printf("malformed '%s' message from master server: '%s'\n", addBan, strings.Join(args, " "))
		return
	}

	ipString := args[0]
	numDots := strings.Count(ipString, ".")
	for i := 0; i < 3-numDots; i++ {
		ipString += ".0"
	}

	ip := net.ParseIP(ipString)
	network := &net.IPNet{IP: ip, Mask: ip.DefaultMask()}

	ms.Bans.AddBan(network, "banned by master server", time.Now(), true)
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
