package bans

import (
	"encoding/json"
	"fmt"
	"net"
	"time"
)

type Ban struct {
	Network    net.IPNet
	Reason     string
	ExpiryDate time.Time
	IsGlobal   bool
}

// UnmarshalJSON implements json.Unmarshaler for Ban
func (b *Ban) UnmarshalJSON(jsonBytes []byte) error {
	var temp map[string]interface{}

	err := json.Unmarshal(jsonBytes, &temp)
	if err != nil {
		return err
	}

	_, network, err := net.ParseCIDR(temp["network"].(string))
	if err != nil {
		return err
	}

	b.Network = *network
	b.Reason = temp["reason"].(string)
	b.ExpiryDate = time.Unix(int64(temp["expiry_date"].(float64)), 0)
	b.IsGlobal = temp["is_global"].(bool)

	return nil
}

func (b Ban) String() string {
	format := "%v is locally banned until %v for %v"
	if b.IsGlobal {
		format = "%v is globally banned until %v for %v"
	}

	return fmt.Sprintf(format, b.Network.String(), b.ExpiryDate, b.Reason)
}
