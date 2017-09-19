package masterserver

import (
	"bufio"
	"log"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/sauerbraten/waiter/internal/bans"
)

// master server protocol constants
const (
	MASTER_REGISTERING_REQUEST  = "regserv"
	MASTER_REGISTERING_SUCCEDED = "succreg"
	MASTER_REGISTERING_FAILED   = "failreg"

	MASTER_ADD_GLOBAL_BAN    = "addgban"
	MASTER_CLEAR_GLOBAL_BANS = "cleargbans"

	MASTER_AUTH_REQUEST      = "reqauth"
	MASTER_AUTH_CONFIRMATION = "confauth"
)

type MasterServer struct {
	In   *bufio.Scanner
	Out  *bufio.Writer
	Bans *bans.BanManager
}

// New connects to the specified master server. Bans received from the master server are added to the given ban manager.
func New(addr string, bans *bans.BanManager) (ms *MasterServer, err error) {
	raddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		log.Println("could not resolve master server address ("+addr+"):", err)
		return
	}

	conn, err := net.DialTCP("tcp", nil, raddr)
	if err != nil {
		log.Println("failed to connect to master server:", err)
		return
	}

	ms = &MasterServer{
		In:   bufio.NewScanner(conn),
		Out:  bufio.NewWriter(conn),
		Bans: bans,
	}

	go ms.handleIncoming()

	return
}

func (ms *MasterServer) Register(listenPort int) error {
	return ms.request(MASTER_REGISTERING_REQUEST + " " + strconv.Itoa(listenPort))
}

func (ms *MasterServer) request(req string) error {
	_, err := ms.Out.WriteString(req + "\n")
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
		case MASTER_REGISTERING_SUCCEDED:
			log.Println("master server registration succeeded")
		case MASTER_REGISTERING_FAILED:
			log.Println("master server registration failed:", strings.Join(msgParts[1:], " "))
		case MASTER_CLEAR_GLOBAL_BANS:
			ms.Bans.ClearGlobalBans()
		case MASTER_ADD_GLOBAL_BAN:
			ms.handleAddGlobalBan(msgParts)
		default:
			log.Println("received from master:", msg)
		}
	}

	if err := ms.In.Err(); err != nil {
		log.Println(err)
	}
}

func (ms *MasterServer) handleAddGlobalBan(msgParts []string) {
	if len(msgParts) != 2 {
		log.Println("malformed 'addgbans' message from master server:", strings.Join(msgParts, " "))
		return
	}

	ipString := msgParts[1]
	numDots := strings.Count(ipString, ".")
	for i := 0; i < 3-numDots; i++ {
		ipString += ".0"
	}

	ip := net.ParseIP(ipString)
	network := net.IPNet{IP: ip, Mask: ip.DefaultMask()}

	ms.Bans.AddBan(network, "banned by master server", time.Now(), true)
}
