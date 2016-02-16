package sock

import (
	"net"
	"testing"
)

func TestJoin(t *testing.T) {
	ifname := "eth2"

	mcastSock, err1 := MulticastListener(2000, "eth2")
	if err1 != nil {
		t.Errorf("Unable to create multicast listener socket: %v", err1)
		return
	}

	if err := Join(mcastSock, net.IPv4(224, 0, 0, 9), ifname); err != nil {
		t.Errorf("Unable to join multicast group: %v", err1)
		Close(mcastSock)
		return
	}

	data := []byte{0}
	dst := net.UDPAddr{IP: net.IPv4(1, 0, 0, 1), Port: 3000}
	n, err2 := mcastSock.U.WriteToUDP(data, &dst)
	if err2 != nil {
		t.Errorf("Error sending to %v,%p: %v", dst.IP, dst.Port, err2)
		Close(mcastSock)
		return
	}

	if n != len(data) {
		t.Errorf("Partial send: %d of %d", n, len(data))
		Close(mcastSock)
		return
	}

	Close(mcastSock)

}
