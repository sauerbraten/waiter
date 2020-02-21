.PHONY: all server tools clean

all: server tools

server:
	go build ./cmd/waiter

tools:
	go build ./cmd/cenc
	go build ./cmd/cdec
	go build ./cmd/genauth

clean:
	rm -f ./waiter ./cenc ./cdec ./genauth