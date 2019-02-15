package bans

import (
	"encoding/json"
	"fmt"
	"net"
	"time"
)

type Ban struct {
	Network    *net.IPNet
	Reason     string
	ExpiryDate time.Time
	Global     bool
}

// UnmarshalJSON implements json.Unmarshaler for Ban
func (b *Ban) UnmarshalJSON(jsonBytes []byte) error {
	ban := struct {
		Network    string `json:"network"`
		Reason     string `json:"reason"`
		ExpiryDate int64  `json:"expiry_date"`
	}{}
	err := json.Unmarshal(jsonBytes, &ban)
	if err != nil {
		return err
	}

	_, b.Network, err = net.ParseCIDR(ban.Network)
	if err != nil {
		return err
	}
	b.Reason = ban.Reason
	b.ExpiryDate = time.Unix(ban.ExpiryDate, 0)

	return nil
}

func (b *Ban) String() string {
	if b.ExpiryDate.IsZero() {
		return fmt.Sprintf("%v is banned indefinitely (%v)", b.Network.String(), b.Reason)
	}
	return fmt.Sprintf("%v is banned until %v (%v)", b.Network.String(), b.ExpiryDate, b.Reason)
}
