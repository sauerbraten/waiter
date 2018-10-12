package packet

import (
	"log"
	"net"

	"github.com/sauerbraten/waiter/cubecode"
	"github.com/sauerbraten/waiter/internal/client/privilege"
	"github.com/sauerbraten/waiter/internal/definitions/armour"
	"github.com/sauerbraten/waiter/internal/definitions/disconnectreason"
	"github.com/sauerbraten/waiter/internal/definitions/gamemode"
	"github.com/sauerbraten/waiter/internal/definitions/mastermode"
	"github.com/sauerbraten/waiter/internal/definitions/nmc"
	"github.com/sauerbraten/waiter/internal/definitions/sound"
	"github.com/sauerbraten/waiter/internal/definitions/weapon"
)

func Encode(args ...interface{}) cubecode.Packet {
	p := make(cubecode.Packet, 0, len(args))

	for _, arg := range args {
		switch v := arg.(type) {
		case int32:
			p.PutInt(v)

		case []int32:
			for _, w := range v {
				p.PutInt(w)
			}

		case int:
			p.PutInt(int32(v))

		case uint32:
			// you'll have to be explicit and call p.PutUint() if you
			// really want that!
			p.PutInt(int32(v))

		case float64:
			p.PutInt(int32(v))

		case byte:
			p = append(p, v)

		case armour.Armour:
			p.PutInt(int32(v))

		case gamemode.GameMode:
			p.PutInt(int32(v))

		case mastermode.MasterMode:
			p.PutInt(int32(v))

		case nmc.NetMessCode:
			p.PutInt(int32(v))

		case privilege.Privilege:
			p.PutInt(int32(v))

		case sound.Sound:
			p.PutInt(int32(v))

		case weapon.ID:
			p.PutInt(int32(v))

		case disconnectreason.DisconnectReason:
			p.PutInt(int32(v))

		case []byte:
			p = append(p, v...)

		case cubecode.Packet:
			p = append(p, v...)

		case net.IP:
			p = append(p, v...)

		case bool:
			if v {
				p = append(p, 1)
			} else {
				p = append(p, 0)
			}

		case string:
			p.PutString(v)

		default:
			log.Printf("unhandled type %T of arg %v\n", v, v)
		}
	}

	return p
}
