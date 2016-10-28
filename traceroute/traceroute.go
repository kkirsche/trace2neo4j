package traceroute

import (
	"fmt"
	"net"
)

// validateAddrType ensures that we chose a valid address family type
func validateAddrType(af string) bool {
	for _, f := range []string{"ip", "ip4", "ip6"} {
		if af == f {
			return true
		}
	}
	return false
}

// getSourceIPAddress is used to get a source IP address to use in our traceroute packets
func getSourceIPAddress(af, addr string) (*net.IP, error) {
	// We have a target IP Address
	if addr != "" {
		if !validateAddrType(af) {
			return nil, fmt.Errorf("Invalid address type %s. Please use ip, ip4, or ip6", af)
		}

		a, err := net.ResolveIPAddr(af, addr)
		if err != nil {
			return nil, err
		}
		return &a.IP, nil
	}

	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return nil, err
	}

	for _, a := range addrs {
		ipnet, ok := a.(*net.IPNet)
		if ok && !ipnet.IP.IsLoopback() {
			if (ipnet.IP.To4() != nil && af == "ip4") || (ipnet.IP.To4() == nil && af == "ip6") {
				return &ipnet.IP, nil
			}
		}
	}

	return nil, fmt.Errorf("Could not find a source address with the %s address family", af)
}

func resolveIPAddr(af, addr string) (*net.IP, error) {
	a, err := net.ResolveIPAddr(af, addr)
	if err != nil {
		return nil, err
	}
	return &a.IP, nil
}
