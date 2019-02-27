all:
	go build ./cmd/waiter

tools:
	go build ./cmd/cenc
	go build ./cmd/cdec
	go build ./cmd/genauth

clean:
	rm -f ./waiter ./genauth ./cenc ./cdec