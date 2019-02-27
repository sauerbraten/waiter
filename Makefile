all:
	go build ./cmd/waiter
	go build ./cmd/genauth

tools:
	go build ./cmd/cenc
	go build ./cmd/cdec

clean:
	rm -f ./waiter ./genauth ./cenc ./cdec