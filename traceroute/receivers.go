package traceroute

import (
	"fmt"
	"net"
	"time"

	"golang.org/x/net/icmp"

	"github.com/Sirupsen/logrus"
)

const (
	icmpHeaderSize   int = 8
	minTCPHeaderSize int = 20
	maxTCPHeaderSize int = 60
	minIP4HeaderSize int = 20
	minIP6HeaderSize int = 40
	maxIP4HeaderSize int = 60
)

// TCPReceiver Feeds on TCP RST messages we receive from the end host; we use lots of parameters to check if the incoming packet
// is actually a response to our probe. We create TCPResponse structs and emit them on the output channel
func TCPReceiver(done <-chan struct{}, af, targetAddr string, srcAddr *net.IP,
	probePortStart, probePortEnd, targetPort, maxTTL int) (chan interface{}, error) {
	logrus.Infoln("Starting TCP Receiver...")

	conn, err := net.ListenPacket(net.JoinHostPort(af, "tcp"), srcAddr.String())
	if err != nil {
		return nil, err
	}

	// The out channel will be written to when we receive a response
	out := make(chan interface{})
	recv := make(chan *TCPResponse)
	go func() {
		ipHeaderSize := 0
		if af == "ip4" {
			ipHeaderSize = minIP4HeaderSize
		} else if af == "ip6" {
			ipHeaderSize = minIP6HeaderSize
		}

		ipAndTCPHeader := ipHeaderSize + maxTCPHeaderSize
		packet := make([]byte, ipAndTCPHeader)
		for {
			readBytes, from, err := conn.ReadFrom(packet)
			if err != nil {
				// this will most likely occur if the parent has closed the socket
				break
			}

			if readBytes < ipAndTCPHeader {
				continue
			}

			// Let's make sure this packet was meant for this program
			tcpHdr := parseTCPHeader(packet[ipHeaderSize:readBytes])
			if int(tcpHdr.Source) != targetPort {
				// This packet wasn't meant for us. Let's keep going
				continue
			}

			// is this a RST or ACK packet?
			if tcpHdr.Flags&RST != RST && tcpHdr.Flags&ACK != ACK {
				// Not a RST or ACK. Let's keep waiting
				continue
			}

			// is this packet from our target destination?
			if from.String() != targetAddr {
				// Nope... :( let's move on
				continue
			}

			logrus.Infof("Received a TCP response message of length %d: %x", readBytes, packet[:readBytes])

			// lets extract the original TTL and timestamp from the ACK number
			ackNum := tcpHdr.AckNum - 1 // this gives us an unsigned int32
			ttl := int(ackNum >> 24)

			if ttl > maxTTL {
				continue
			}

			ts := ackNum & 0x00ffffff
			now := uint32(time.Now().UnixNano()/(1000*1000)) & 0x00ffffff

			// received timestamp is higher than local time; it is possible
			// that ts == now, since our clock resolution is coarse
			if ts > now {
				continue
			}

			recv <- &TCPResponse{
				Probe: Probe{
					srcPort: int(tcpHdr.Destination),
					ttl:     ttl,
				},
				rtt: now - ts,
			}
		}

	}()

	go func() {
		defer conn.Close()
		defer close(out)

		for {
			select {
			case response := <-recv:
				out <- response
			case <-done:
				logrus.Infoln("TCP Receiver exiting...")
				return
			}
		}
	}()

	return out, nil
}

// ICMPReceiver runs on its own collecting ICMP responses until its explicitly told to stop
func ICMPReceiver(done <-chan struct{}, af string, srcAddr net.IP) (chan interface{}, error) {
	var (
		minInnerIPHeaderSize int
		// icmpMsgType          byte
		listenNet string
	)

	switch af {
	case "ip4":
		minInnerIPHeaderSize = minIP4HeaderSize // the size of the original IPv4 header that was on the TCP packet sent out
		// icmpMsgType = 11                        // time to live exceeded
		listenNet = "ip4:1" // IPv4 ICMP proto number
	case "ip6":
		minInnerIPHeaderSize = minIP6HeaderSize // the size of the original IPv4 header that was on the TCP packet sent out
		// icmpMsgType = 3                         // time to live exceeded
		listenNet = "ip6:58" // IPv6 ICMP proto number
	default:
		return nil, fmt.Errorf("ICMPReceiver: Unsupported network %s", af)
	}

	conn, err := icmp.ListenPacket(listenNet, srcAddr.String())
	if err != nil {
		return nil, err
	}

	logrus.Infoln("Starting ICMP Receiver...")
	out := make(chan interface{})
	recv := make(chan *ICMPResponse)
	go func() {
		packet := make([]byte, icmpHeaderSize+maxIP4HeaderSize+maxTCPHeaderSize)
		for {
			readBytes, from, err := conn.ReadFrom(packet)
			if err != nil {
				// parent probably closed the socket
				break
			}

			// extract the 8 bytes of the original TCP header
			if readBytes < icmpHeaderSize+minInnerIPHeaderSize+minTCPHeaderSize {
				continue
			}

			// not ttl exceeded
			// if packet[0] != icmpMsgType || packet[1] != 0 {
			// 	continue
			// }

			logrus.Infof("Received ICMP response message of length %d: %x", readBytes, packet[:readBytes])
			tcpHeader := parseTCPHeader(packet[icmpHeaderSize+minInnerIPHeaderSize : readBytes])

			// extract ttl bits
			ttl := int(tcpHeader.SeqNum) >> 24

			// extract the timestamp
			ts := tcpHeader.SeqNum & 0x00ffffff
			// scale the time
			now := uint32(time.Now().UnixNano()/(1000*1000)) & 0x00ffffff
			fromIP := net.ParseIP(from.String())
			recv <- &ICMPResponse{
				Probe: Probe{
					srcPort: int(tcpHeader.Source),
					ttl:     ttl,
				},
				fromAddr: &fromIP,
				rtt:      now - ts,
			}
		}
	}()

	go func() {
		defer conn.Close()
		defer close(out)
		for {
			select {
			case response := <-recv:
				out <- response
			case <-done:
				logrus.Infoln("ICMP Receiver exiting...")
				return
			}
		}
	}()

	return out, nil
}
