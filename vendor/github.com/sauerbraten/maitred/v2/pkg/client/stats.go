package client

import (
	"fmt"
	"log"
	"strings"

	"github.com/sauerbraten/maitred/v2/pkg/protocol"
)

func ExtendWithStats(c *Client, onSuccess func(uint32), onFailure func(uint32, string)) {
	c.RegisterExtension(protocol.FailStats, func(args string) {
		r := strings.NewReader(strings.TrimSpace(args))

		var reqID uint32
		_, err := fmt.Fscanf(r, "%d", &reqID)
		if err != nil {
			log.Printf("malformed %s message from stats server: '%s': %v", protocol.FailStats, args, err)
			return
		}

		reason := args[len(args)-r.Len():] // unread portion of args
		reason = strings.TrimSpace(reason)

		onFailure(reqID, reason)
	})

	c.RegisterExtension(protocol.SuccStats, func(args string) {
		var reqID uint32
		_, err := fmt.Sscanf(args, "%d", &reqID)
		if err != nil {
			log.Printf("malformed %s message from stats server: '%s': %v", protocol.SuccStats, args, err)
			return
		}

		onSuccess(reqID)
	})
}
