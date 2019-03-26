package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"

	"github.com/sauerbraten/maitred/pkg/auth"

	sauth "github.com/sauerbraten/waiter/pkg/auth"
	"github.com/sauerbraten/waiter/pkg/protocol/role"
)

func main() {
	if len(os.Args) < 4 {
		fmt.Println("Usage: genauth <name> <domain> <role>")
		os.Exit(1)
		return
	}

	name, domain, rol := os.Args[1], os.Args[2], role.Parse(os.Args[3])

	if rol != role.Master && rol != role.Admin {
		fmt.Println("role must be 'master' or 'admin'")
		os.Exit(2)
		return
	}

	priv, pub, err := auth.GenerateKeyPair()
	if err != nil {
		fmt.Println(err)
		os.Exit(3)
		return
	}

	u := &sauth.User{
		Name:      name,
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
