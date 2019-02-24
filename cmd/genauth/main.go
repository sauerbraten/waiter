package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"

	"github.com/sauerbraten/waiter/pkg/auth"
	"github.com/sauerbraten/waiter/pkg/definitions/role"
)

func main() {
	if len(os.Args) < 4 {
		fmt.Println("Usage: genauth <domain> <role> <name>")
		os.Exit(1)
		return
	}

	domain, rol, name := os.Args[1], role.Parse(os.Args[2]), os.Args[3]

	if rol != role.Master && rol != role.Admin {
		fmt.Println("privilege must be 'master' or 'admin'")
		os.Exit(2)
		return
	}

	priv, pub, err := auth.GenerateKeyPair()
	if err != nil {
		fmt.Println(err)
		os.Exit(3)
		return
	}

	u := &auth.User{
		UserIdentifier: auth.UserIdentifier{
			Name:   name,
			Domain: domain,
		},
		PublicKey: pub,
		Role:      rol,
	}

	fmt.Printf("add to user's auth.cfg:\nauthkey \"%s\" \"%s\" \"%s\"\n", name, hex.EncodeToString(priv), domain)
	fmt.Println("add to server's user.json:")

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "\t")
	err = enc.Encode(u)
	if err != nil {
		fmt.Println(err)
		os.Exit(4)
		return
	}
}
