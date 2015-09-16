package iface

import (
	"log"
	"net"
	"os"
	//"strings"
	"fmt"
	"syscall"
	"unsafe"

	"rib-old/util"
)

type adapterInfo struct {
	head *syscall.IpAdapterInfo
}

// From: https://code.google.com/p/go/source/browse/src/pkg/net/interface_windows.go
/*
func bytePtrToString(p *uint8) string {
	a := (*[1000]uint8)(unsafe.Pointer(p))
	i := 0
	for a[i] != 0 {
		i++
	}
	return string(a[:i])
}
*/

// From: https://code.google.com/p/go/source/browse/src/pkg/net/interface_windows.go
func getAdapterList() (*syscall.IpAdapterInfo, error) {
	b := make([]byte, 1000)
	l := uint32(len(b))
	a := (*syscall.IpAdapterInfo)(unsafe.Pointer(&b[0]))
	// TODO(mikio): GetAdaptersInfo returns IP_ADAPTER_INFO that
	// contains IPv4 address list only. We should use another API
	// for fetching IPv6 stuff from the kernel.
	err := syscall.GetAdaptersInfo(a, &l)
	if err == syscall.ERROR_BUFFER_OVERFLOW {
		b = make([]byte, l)
		a = (*syscall.IpAdapterInfo)(unsafe.Pointer(&b[0]))
		err = syscall.GetAdaptersInfo(a, &l)
	}
	if err != nil {
		return nil, os.NewSyscallError("GetAdaptersInfo", err)
	}
	return a, nil
}

func toString(p []byte) string {
	for i, b := range p {
		if b == 0 {
			return string(p[:i])
		}
	}
	return string(p)
}

func parseIP(p [16]byte) net.IP {
	//str := bytePtrToString(&p[0])
	str := toString(p[:])
	//log.Printf("parseIP: [%v] len=%d", str, len(str))
	return net.ParseIP(str)
}

// getMask: get net.IPNet from net.IPAddr
func getMask(info *adapterInfo, index int, addr net.IPAddr) (net.IPNet, error) {

	ipNet := net.IPNet{}

	if info.head == nil {
		var err error
		info.head, err = getAdapterList()
		if err != nil {
			return ipNet, err
		}
	}

	v4 := util.IpIsIPv4(addr.IP)

	for ai := info.head; ai != nil; ai = ai.Next {
		if index != int(ai.Index) {
			continue
		}

		for ipl := &ai.IpAddressList; ipl != nil; ipl = ipl.Next {

			ip := parseIP(ipl.IpAddress.String)
			if ip == nil {
				return ipNet, fmt.Errorf("getMask: parse error: [%v]", ip)
			}

			if !ip.Equal(addr.IP) {
				continue
			}

			mask := parseIP(ipl.IpMask.String)
			if mask == nil {
				return ipNet, fmt.Errorf("getMask: parse error: [%v]", mask)
			}

			ipNet.IP = addr.IP

			if v4 {
				m := mask.To4() // convert mask into 4-byte
				ipNet.Mask = net.IPv4Mask(m[0], m[1], m[2], m[3])
			} else {
				// IPv6 mask
				ipNet.Mask = net.IPMask{
					mask[0], mask[1], mask[2], mask[3],
					mask[4], mask[5], mask[6], mask[7],
					mask[8], mask[9], mask[10], mask[11],
					mask[12], mask[13], mask[14], mask[15],
				}
			}

			return ipNet, nil
		}
	}

	return ipNet, fmt.Errorf("getMask: not found: [%v]", addr)
}

/*
	GetInterfaceAddrs() is work-around for:
	http://code.google.com/p/go/issues/detail?id=5395
	Otherwise it could be replaced with net.Interface.Addrs()
*/
func GetInterfaceAddrs(i net.Interface) ([]net.Addr, error) {

	addrs, err := i.Addrs()
	if err != nil {
		return addrs, err
	}

	result := []net.Addr{}

	info := adapterInfo{}

	for _, a := range addrs {
		switch ad := a.(type) {
		case *net.IPNet:
			// linux, bsd, darwin, etc...
			result = append(result, a)
		case *net.IPAddr:
			// windows: missing netmask
			//log.Printf("GetInterfaceAddrs: net.IPAddr: %v: does not provide netmask", ad)
			ipNet, err := getMask(&info, i.Index, *ad)
			if err != nil {
				log.Printf("GetInterfaceAddrs: net.IPAddr: %v: error: %v", ad, err)
				result = append(result, a)
				continue
			}
			result = append(result, &ipNet)
		default:
			// does this happen?
			log.Printf("GetInterfaceAddrs: unknown type: %v: does not provide netmask", ad)
			result = append(result, a)
		}
	}

	return result, nil
}
