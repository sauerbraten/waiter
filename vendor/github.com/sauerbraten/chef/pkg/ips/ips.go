package ips

import (
	"net"
	"regexp"
	"strconv"
	"strings"
)

// Matches when there is at least the first octet and the first dot. Octets one to three need to end with a dot. Also matches CIDR notations of ranges.
// Examples:
// 123.
// 109.103.
// 11.233.109.201
// 154.93.0.0/16
var partialOrFullIpRangeRegex *regexp.Regexp = regexp.MustCompile(`^(25[0-5]|2[0-4][0-9]|[01]?[0-9]?[0-9])\.((25[0-5]|2[0-4][0-9]|[01]?[0-9]?[0-9])\.)?((25[0-5]|2[0-4][0-9]|[01]?[0-9]?[0-9])\.)?(25[0-5]|2[0-4][0-9]|[01]?[0-9]?[0-9])?(\/(3[0-1]|[12]?[0-9]))?$`)

// IsPartialOrFullCIDR returns true if s is an IP or a range in CIDR notation.
func IsPartialOrFullCIDR(s string) bool {
	return partialOrFullIpRangeRegex.MatchString(s)
}

// Matches when a full and vaild CIDR notation of a range is given.
// Example:
// 154.93.0.0/16
var ipRangeRegex *regexp.Regexp = regexp.MustCompile(`^(25[0-5]|2[0-4][0-9]|[01]?[0-9]?[0-9])\.(25[0-5]|2[0-4][0-9]|[01]?[0-9]?[0-9])\.(25[0-5]|2[0-4][0-9]|[01]?[0-9]?[0-9])\.(25[0-5]|2[0-4][0-9]|[01]?[0-9]?[0-9])\/(3[0-1]|[12]?[0-9])$`)

// IsRangeAsCIDR returns true if s is an IP or a range in CIDR notation.
func IsRangeAsCIDR(s string) bool {
	return ipRangeRegex.MatchString(s)
}

// IP2Int returns the int representation of the IP. Uses int64 to prevent negative values (easier range checks in DB). Assumes 4-byte IPv4. Inverse function of Int2IP.
func IP2Int(ip net.IP) (intIp int64) {
	for index, octet := range ip.To4() {
		intIp += int64(octet) << uint((3-index)*8)
	}

	return
}

// Int2IP returns the net.IP representation of the integer. Inverse function of IP2Int.
func Int2IP(intIp int64) net.IP {
	abcd := [4]byte{}

	for index, _ := range abcd {
		abcd[index] = byte(intIp >> uint((3-index)*8))
	}

	return net.IPv4(abcd[0], abcd[1], abcd[2], abcd[3])
}

// GetSubnet parses all of the following examples into valid IP ranges by padding the IP:
//     123.
//     184.29.39.193/16
//     12.304/8
//     29.43.223./13
// A CIDR notation prefix size is optional, a fitting prefix size will be chosen in case it's omitted or the specified prefix size is > 24.
func GetSubnet(cidr string) (ipNet *net.IPNet) {
	parts := strings.Split(cidr, "/")

	ipString := parts[0]
	prefixSize := 0

	// check prefix size

	if len(parts) == 2 {
		var err error
		prefixSize, err = strconv.Atoi(parts[1])
		if err != nil {
			prefixSize = 0
		} else if prefixSize < 0 || prefixSize > 24 {
			prefixSize = 0
		}
	}

	// pad IP & choose a prefix size if not specified

	if !strings.HasSuffix(ipString, ".") && strings.Count(ipString, ".") < 3 {
		ipString += "."
	}

	switch strings.Count(ipString, ".") {
	case 1:
		ipString += "0.0.0"

		if prefixSize == 0 {
			prefixSize = 8
		}

	case 2:
		ipString += "0.0"

		if prefixSize == 0 {
			prefixSize = 16
		}

	case 3:
		if strings.HasSuffix(ipString, ".") {
			ipString += "0"
		}

		if prefixSize == 0 {
			prefixSize = 24
		}
	}

	// TODO: maybe handle error?
	_, ipNet, _ = net.ParseCIDR(ipString + "/" + strconv.Itoa(prefixSize))

	return
}

// GetDecimalBoundaries returns the lowest and the highest IP inside the IPNet as integers (for easy range checking).
func GetDecimalBoundaries(ipNet *net.IPNet) (lowest, highest int64) {
	var notMask int64

	for index, ipOctet := range ipNet.IP.To4() {
		maskOctet := ipNet.Mask[index]
		notMask += int64(^maskOctet) << uint((3-index)*8)

		octet := ipOctet & maskOctet
		lowest += int64(octet) << uint((3-index)*8)
	}

	highest = lowest + notMask

	return
}
