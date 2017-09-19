FROM golang:alpine as build

RUN apk add --update --no-cache git g++ make autoconf automake libtool && \
	wget -q -O - http://enet.bespin.org/download/enet-1.3.13.tar.gz | tar -xz && \
	cd enet-1.3.13 && autoreconf -vfi && ./configure && make && make install

WORKDIR /go/src/github.com/sauerbraten/waiter
COPY . .

WORKDIR /go/src/github.com/sauerbraten/waiter/cmd/waiter
RUN go-wrapper download
RUN go-wrapper install -ldflags "-linkmode external -extldflags -static"


FROM gcr.io/distroless/base

COPY --from=build /go/bin/waiter /
COPY ./config.json ./bans.json /

EXPOSE 28785-28786/udp

CMD ["/waiter"]