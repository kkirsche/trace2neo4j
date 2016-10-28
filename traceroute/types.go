package traceroute

import "net"

type Probe struct {
	srcPort int
	ttl     int
}

type ICMPResponse struct {
	Probe
	fromAddr *net.IP
	fromName string
	rtt      uint32
}

type TCPResponse struct {
	Probe
	rtt uint32
}
