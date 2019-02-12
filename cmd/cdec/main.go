package main

import (
	"encoding/hex"
	"fmt"
	"os"
	"strings"

	"github.com/sauerbraten/waiter/pkg/protocol"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("nothing to decode")
		fmt.Println("cdec \"<b|i|s>...\" <byte as hex>...")
		os.Exit(1)
		return
	}

	buf, err := hex.DecodeString(strings.Join(os.Args[2:], ""))
	if err != nil {
		fmt.Println(err)
		return
	}

	p := protocol.Packet(buf)
	pos := 0

	for pos < len(os.Args[1]) && len(p) > 0 {
		switch os.Args[1][pos] {
		case 'b':
			b, ok := p.GetByte()
			if !ok {
				fmt.Println("input too short")
			}
			fmt.Printf("%x ", b)
		case 'i':
			i, ok := p.GetInt()
			if !ok {
				fmt.Println("input too short")
			}
			fmt.Printf("%d ", i)
		case 'x':
			s, ok := p.GetString()
			if !ok {
				fmt.Println("input too short")
			}
			fmt.Printf("%s ", s)
		}
	}

	fmt.Println()
}
