package client

import (
	"fmt"
	"net"

	"github.com/sauerbraten/maitred/v2/pkg/protocol"
)

type conn struct {
	addr string
	*protocol.Conn
	onConnect func()
}

func newConn(addr string, onConnect func(), onDisconnect func(error)) *conn {
	pConn := protocol.NewConn(onDisconnect)

	return &conn{
		addr:      addr,
		Conn:      pConn,
		onConnect: onConnect,
	}
}

func (c *conn) connect() error {
	raddr, err := net.ResolveTCPAddr("tcp", c.addr)
	if err != nil {
		return fmt.Errorf("resolving %s: %w", c.addr, err)
	}

	conn, err := net.DialTCP("tcp", nil, raddr)
	if err != nil {
		return err
	}

	c.Conn.Start(conn)
	c.onConnect()
	return nil
}
