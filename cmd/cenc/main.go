package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/sauerbraten/waiter/pkg/protocol"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("nothing to encode")
		fmt.Println("cenc <i|int|s|string> <input>")
		os.Exit(1)
		return
	}

	p := make(protocol.Packet, 0, 1024)

	switch os.Args[1] {
	case "i", "int":
		v, err := strconv.ParseInt(os.Args[2], 10, 32)
		if err != nil {
			fmt.Println("could not parse integer:", err)
			os.Exit(1)
			return
		}
		p.PutInt(int32(v))
	case "s", "string":
		p.PutString(os.Args[2])
	}

	fmt.Println(p)
}
