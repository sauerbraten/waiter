package packet

import (
	"log"
	"net"
	"time"

	"github.com/sauerbraten/waiter/pkg/definitions/armour"
	"github.com/sauerbraten/waiter/pkg/definitions/disconnectreason"
	"github.com/sauerbraten/waiter/pkg/definitions/gamemode"
	"github.com/sauerbraten/waiter/pkg/definitions/mastermode"
	"github.com/sauerbraten/waiter/pkg/definitions/nmc"
	"github.com/sauerbraten/waiter/pkg/definitions/playerstate"
	"github.com/sauerbraten/waiter/pkg/definitions/role"
	"github.com/sauerbraten/waiter/pkg/definitions/sound"
	"github.com/sauerbraten/waiter/pkg/definitions/weapon"
	"github.com/sauerbraten/waiter/internal/geom"
	"github.com/sauerbraten/waiter/pkg/protocol"
)

func Encode(args ...interface{}) protocol.Packet {
	if len(args) == 0 {
		return nil
	}

	p := make(protocol.Packet, 0, len(args))

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

		case playerstate.ID:
			p.PutInt(int32(v))

		case armour.ID:
			p.PutInt(int32(v))

		case gamemode.ID:
			p.PutInt(int32(v))

		case mastermode.ID:
			p.PutInt(int32(v))

		case nmc.ID:
			p.PutInt(int32(v))

		case role.ID:
			p.PutInt(int32(v))

		case sound.ID:
			p.PutInt(int32(v))

		case weapon.ID:
			p.PutInt(int32(v))

		case disconnectreason.ID:
			p.PutInt(int32(v))

		case []byte:
			p = append(p, v...)

		case protocol.Packet:
			p = append(p, v...)

		case net.IP:
			p = append(p, v...)

		case time.Duration:
			p.PutInt(int32(v / time.Second))

		case bool:
			if v {
				p = append(p, 1)
			} else {
				p = append(p, 0)
			}

		case string:
			p.PutString(v)

		case *geom.Vector:
			p.PutInt(int32(v.X()))
			p.PutInt(int32(v.Y()))
			p.PutInt(int32(v.Z()))

		default:
			log.Printf("unhandled type %T of arg %v\n", v, v)
		}
	}

	return p
}
