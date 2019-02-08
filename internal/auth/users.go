package auth

import (
	"encoding/json"
	"fmt"

	"github.com/sauerbraten/waiter/internal/client/privilege"
)

type UserIdentifier struct {
	Name   string `json:"name"`
	Domain string `json:"domain"`
}

type User struct {
	UserIdentifier
	PublicKey publicKey    `json:"public_key"`
	Privilege privilege.ID `json:"-"`
}

func (u *User) MarshalJSON() ([]byte, error) {
	proxy := struct {
		User
		Privilege string `json:"privilege"`
	}{
		User:      *u,
		Privilege: u.Privilege.String(),
	}
	return json.Marshal(proxy)
}

func (u *User) UnmarshalJSON(data []byte) error {
	proxy := &struct {
		UserIdentifier
		PublicKey publicKey `json:"public_key"`
		Privilege string    `json:"privilege"`
	}{}
	err := json.Unmarshal(data, proxy)
	if err != nil {
		return err
	}
	u.UserIdentifier = proxy.UserIdentifier
	u.PublicKey = proxy.PublicKey
	u.Privilege = privilege.Parse(proxy.Privilege)
	if u.Privilege == -1 {
		return fmt.Errorf("invalid value for 'privilege'")
	}
	return nil
}
