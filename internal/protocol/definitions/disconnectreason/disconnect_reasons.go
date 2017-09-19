package disconnectreason

type DisconnectReason uint32

const (
	None DisconnectReason = iota
	EOP
	LocalMode // LOCAL
	Kick
	MessageError // MSGERR
	Banned       // IPBAN
	PrivateMode  // PRIVATE
	MaxClients
	Timeout
	Overflow
	Password
	//DISCNUM
)

var String []string = []string{
	"",
	"end of packet",
	"server is in local mode",
	"kicked/banned",
	"message error",
	"ip is banned",
	"server is in private mode",
	"server full",
	"connection timed out",
	"overflow",
	"invalid password",
}
