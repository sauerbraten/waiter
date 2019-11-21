package ips

import (
	"bufio"
	"log"
	"net"
	"strings"
)

// from https://www.iana.org/assignments/iana-ipv4-special-registry/iana-ipv4-special-registry.xhtml
//
// excluded for de-duplication (covered by 192.0.0.0/24 below):
// 192.0.0.0/29
// 192.0.0.170/32
// 192.0.0.171/32
const reservedAddressBlocks = `0.0.0.0/8
10.0.0.0/8
100.64.0.0/10
127.0.0.0/8
169.254.0.0/16
172.16.0.0/12
192.0.0.0/24
192.0.2.0/24
192.31.196.0/24
192.52.193.0/24
192.168.0.0/16
198.18.0.0/15
198.51.100.0/24
203.0.113.0/24
240.0.0.0/4
255.255.255.255/32`

var reservedNetworks []*net.IPNet

func init() {
	scanner := bufio.NewScanner(strings.NewReader(reservedAddressBlocks))

	for scanner.Scan() {
		_, reservedNet, err := net.ParseCIDR(scanner.Text())
		if err != nil {
			log.Println(err)
			continue
		}

		reservedNetworks = append(reservedNetworks, reservedNet)
	}

	if err := scanner.Err(); err != nil {
		log.Println(err)
	}
}

// IsInReservedBlock returns true if the IP belongs to a reserved IPv4 address block according to IANA (see
// https://www.iana.org/assignments/iana-ipv4-special-registry/iana-ipv4-special-registry.xhtml).
func IsInReservedBlock(ip net.IP) bool {
	for _, net := range reservedNetworks {
		if net.Contains(ip) {
			return true
		}
	}

	return false
}
