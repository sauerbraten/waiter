package client

type Client interface {
	Start()
	Register(int)
	Send(string, ...interface{})
	Incoming() <-chan string
	Handle(string)
	Logf(string, ...interface{})
}
