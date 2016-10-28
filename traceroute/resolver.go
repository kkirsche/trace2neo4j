package traceroute

import "net"

func Resolver(input chan interface{}) (chan interface{}, error) {
	out := make(chan interface{})
	go func() {
		defer close(out)

		for val := range input {
			switch val.(type) {
			case ICMPResponse:
				resp := val.(ICMPResponse)
				names, err := net.LookupAddr(resp.fromAddr.String())
				if err != nil {
					resp.fromName = resp.fromAddr.String()
				} else {
					resp.fromName = names[0]
				}
				out <- resp
			default:
				out <- val
			}
		}
	}()

	return out, nil
}
