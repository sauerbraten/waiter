package packet

import (
	"log"

	"github.com/sauerbraten/waiter/internal/client/privilege"
	"github.com/sauerbraten/waiter/internal/protocol/definitions/armour"
	"github.com/sauerbraten/waiter/internal/protocol/definitions/disconnectreason"
	"github.com/sauerbraten/waiter/internal/protocol/definitions/gamemode"
	"github.com/sauerbraten/waiter/internal/protocol/definitions/mastermode"
	"github.com/sauerbraten/waiter/internal/protocol/definitions/nmc"
	"github.com/sauerbraten/waiter/internal/protocol/definitions/sound"
	"github.com/sauerbraten/waiter/internal/protocol/definitions/weapon"
	"github.com/sauerbraten/waiter/internal/utils"
)

type Packet struct {
	buf []byte
	pos int
}

var Empty = New()

func New(args ...interface{}) *Packet {
	p := &Packet{}
	p.Put(args...)
	return p
}

func (p *Packet) SubPacket(length int) *Packet {
	return &Packet{
		buf: p.buf[p.pos : p.pos+utils.Min(length, p.Len()-1)],
		pos: 0,
	}
}

func (p *Packet) Len() int { return len(p.buf) }

func (p *Packet) HasRemaining() bool { return p.pos < p.Len() }

func (p *Packet) Seek(skip int) { p.pos = utils.Min(p.pos+skip, p.Len()-1) }

func (p *Packet) Bytes() []byte { return p.buf[p.pos:] }

func (p *Packet) Clear() {
	p.buf = p.buf[:0]
	p.pos = 0
}

// Appends all arguments to the packet.
func (p *Packet) Put(args ...interface{}) {
	for _, arg := range args {
		switch v := arg.(type) {
		case int32:
			p.putInt32(v)

		case []int32:
			for _, w := range v {
				p.putInt32(w)
			}

		case int:
			p.Put(int32(v))

		case uint:
			p.Put(int32(v))

		case byte:
			p.Put(int32(v))

		case armour.Armour:
			p.Put(int32(v))

		case gamemode.GameMode:
			p.Put(int32(v))

		case mastermode.MasterMode:
			p.Put(int32(v))

		case nmc.NetMessCode:
			p.Put(int32(v))

		case privilege.Privilege:
			p.Put(int32(v))

		case sound.Sound:
			p.Put(int32(v))

		case weapon.Weapon:
			p.Put(int32(v))

		case disconnectreason.DisconnectReason:
			p.Put(int32(v))

		case []byte:
			p.putBytes(v)

		case bool:
			if v {
				p.Put(1)
			} else {
				p.Put(0)
			}

		case string:
			p.putString(v)

		case Packet:
			p.putBytes(v.buf)

		case *Packet:
			p.putBytes((*v).buf)

		default:
			log.Printf("unhandled type %T of arg %v\n", v, v)
		}
	}
}

// Appends a []byte to the end of the packet.
func (p *Packet) putBytes(b []byte) {
	p.buf = append(p.buf, b...)
}

// Encodes an int32 and appends it to the packet.
func (p *Packet) putInt32(i int32) {
	if i < 128 && i > -127 {
		p.buf = append(p.buf, byte(i))
	} else if i < 0x8000 && i >= -0x8000 {
		p.buf = append(p.buf, 0x80, byte(i), byte(i>>8))
	} else {
		p.buf = append(p.buf, 0x81, byte(i), byte(i>>8), byte(i>>16), byte(i>>24))
	}
}

// Appends a string to the packet.
func (p *Packet) putString(s string) {
	for _, c := range s {
		p.Put(int32(c))
	}
	p.Put(0)
}

// Returns the first byte in the Packet.
func (p *Packet) GetByte() byte {
	b := p.buf[p.pos]
	p.pos++
	return b
}

// Decodes an int32 and increases the position index accordingly. Returns the number of bytes read to decode the int.
func (p *Packet) getInt32() (int32, int) {
	b := p.GetByte()

	switch b {
	default:
		return int32(int8(b)), 1
	case 0x80:
		return int32(int16(int32(p.GetByte()) + (int32(p.GetByte()) << 8))), 3
	case 0x81:
		return int32(int32(p.GetByte()) + (int32(p.GetByte()) << 8) + (int32(p.GetByte()) << 16) + (int32(p.GetByte()) << 24)), 5
	}
}

// Decodes an int32 and increases the position index accordingly.
func (p *Packet) GetInt32() int32 {
	value, _ := p.getInt32()
	return value
}

// Decodes an int32 without advancing the position.
func (p *Packet) PeekInt32() int32 {
	value, n := p.getInt32()
	p.pos -= n
	return value
}

// Decodes an int32 using the different compression meant for uint32s and increases the position index accordingly.
// func (p *Packet) getUint32() int32 {
// 	i := int32(p.getByte())
// 	if i >= 0x80 {
// 		i += int32(p.getByte()<<7) - 0x80
// 		if i >= (1 << 14) {
// 			i += int32(p.getByte()<<14) - (1 << 14)
// 		}
// 		if i >= (1 << 21) {
// 			i += int32(p.getByte()<<21) - (1 << 21)
// 		}
// 		if i >= (1 << 28) {
// 			i |= -(1 << 28)
// 		}
// 	}
//
// 	return i
// }

// Reads a string from the packet and increases the position index accordingly.
func (p *Packet) GetString() string {
	s := ""

	codepoint := uint8(p.GetInt32())
	for codepoint != 0x0 {
		s += string(cubeToUni[codepoint])
		codepoint = uint8(p.GetInt32())
	}

	return s
}
