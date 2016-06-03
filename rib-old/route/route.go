package route

import (
	"net"
)

type Route struct {
	Net           net.IP
	PrefixLen     int
	NextHop       net.IP
	InterfaceAddr net.IP
	Metric        int
	Status        int
}
