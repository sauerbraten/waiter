package protocol

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"sync"
	"time"
)

type Conn struct {
	*net.TCPConn
	_disconnect sync.Once

	incoming chan string
	outgoing chan string

	onDisconnect func(error)
}

func NewConn(onDisconnect func(error)) *Conn {
	if onDisconnect == nil {
		onDisconnect = func(error) {}
	}

	return &Conn{
		_disconnect: sync.Once{},

		incoming: make(chan string),
		outgoing: make(chan string),

		onDisconnect: onDisconnect,
	}
}

func (c *Conn) Start(tcpConn *net.TCPConn) {
	tcpConn.SetKeepAlive(true)
	tcpConn.SetKeepAlivePeriod(2 * time.Minute)
	c.TCPConn = tcpConn
	go c.ingest()
	go c.drain()
}

func (c *Conn) Incoming() <-chan string { return c.incoming }

func (c *Conn) disconnect(err error) {
	c._disconnect.Do(func() {
		c.TCPConn.Close()
		c.onDisconnect(err)
	})
}

func (c *Conn) ingest() {
	sc := bufio.NewScanner(c.TCPConn)

	for sc.Scan() {
		c.incoming <- sc.Text()
	}

	err := sc.Err()
	if err == nil {
		err = io.EOF
	}

	c.disconnect(err)
}

func (c *Conn) drain() {
	var err error

	for msg := range c.outgoing {
		err = c.TCPConn.SetWriteDeadline(time.Now().Add(10 * time.Millisecond))
		if err != nil {
			err = fmt.Errorf("failed to set write deadline: %v", err)
			break
		}

		_, err = c.TCPConn.Write([]byte(msg + "\n"))
		if err != nil {
			err = fmt.Errorf("failed to send '%s': %v", msg, err)
			break
		}
	}

	c.disconnect(err)
}

func (c *Conn) Send(format string, args ...interface{}) {
	c.outgoing <- fmt.Sprintf(format, args...)
}
