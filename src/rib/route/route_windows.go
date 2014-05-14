package route

// See http://golang.org/pkg/go/build/

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"os/exec"
	"strconv"
	"strings"

	"rib/util"
)

const (
	S_NEW    = 0
	S_ERASED = 1
	S_NONE   = 2
)

type Route struct {
	Net           net.IP
	PrefixLen     int
	NextHop       net.IP
	InterfaceAddr net.IP
	Metric        int
	Status        int
}

func (r1 Route) Equal(r2 Route) bool {
	return r1.PrefixLen == r2.PrefixLen &&
		r1.Metric == r2.Metric &&
		r1.Net.Equal(r2.Net) &&
		r1.NextHop.Equal(r2.NextHop) &&
		r1.InterfaceAddr.Equal(r2.InterfaceAddr)
}

var (
	routeAdd = make(chan Route)
	routeDel = make(chan Route)
)

func parseRoute(cols []string, route *Route) error {
	//log.Printf("parseRoute: cols=[%v]", cols)

	n := len(cols)

	/*
		route.Net = net.ParseIP(cols[2])
		if route.Net != nil && !util.IpIsIPv4(route.Net) {
			log.Printf("IPv6: [%v]", route.Net)
			// IPv6
			return nil
		}
	*/

	route.Net = net.ParseIP(cols[0])
	if route.Net == nil {
		return nil
	}

	//log.Printf("parseRoute: dest=%v", dest)

	if util.IpIsIPv4(route.Net) {
		// IPv4

		mask := net.ParseIP(cols[1])
		if mask == nil {
			return nil
		}

		m := mask.To4()
		ipmask := net.IPv4Mask(m[0], m[1], m[2], m[3])

		route.PrefixLen, _ = ipmask.Size()

		route.NextHop = net.ParseIP(cols[2])
		if route.NextHop == nil {
			// may be a non-address string "On-link"
			route.NextHop = net.IPv4zero
		}

		var metricCol int

		switch n {
		case 6:
			route.InterfaceAddr = net.ParseIP(cols[4])
			metricCol = 5
		case 5:
			route.InterfaceAddr = net.ParseIP(cols[3])
			metricCol = 4
		case 4:
			metricCol = 3
		default:
			return fmt.Errorf("parse error")
		}

		//log.Printf("cols=%d: %v", len(cols), cols)

		var err error
		route.Metric, err = strconv.Atoi(cols[metricCol])
		if err != nil {
			route.Metric = -1
		}

		//log.Printf("parse: [%v] => [%v]", cols, route)

		return nil
	}

	return fmt.Errorf("parse IPv6")
}

func removeErased(routeTable *[]Route) {
	for i := 0; i < len(*routeTable); {
		if (*routeTable)[i].Status == S_ERASED {
			last := len(*routeTable) - 1
			(*routeTable)[i], *routeTable = (*routeTable)[last], (*routeTable)[:last]
			continue
		}
		i++
	}
}

func sendUpdates(routeTable []Route) {
	log.Printf("table size=%d", len(routeTable))

	for _, r := range routeTable {
		switch r.Status {
		case S_ERASED:
			routeDel <- r
		case S_NEW:
			routeAdd <- r
		}
	}
}

func markToDelete(routeTable []Route) {
	for i := range routeTable {
		routeTable[i].Status = S_ERASED
	}
}

func refreshRoute(routeTable *[]Route, route Route) {
	for i, r := range *routeTable {
		if route.Equal(r) {
			(*routeTable)[i].Status = S_NONE
			return
		}
	}
	*routeTable = append(*routeTable, route)
}

func scanLines(input io.ReadCloser) error {

	routeTable := []Route{}

	scanningIPv4 := false

	scanner := bufio.NewScanner(input)
	for scanner.Scan() {
		line := scanner.Text()
		//log.Printf("scanLines: line=[%v]\n", line)

		if line == "IPv4 Route Table" || line == "Tabela de rotas IPv4" {
			scanningIPv4 = true
			markToDelete(routeTable)
			continue
		}

		if line == "IPv6 Route Table" || line == "Tabela de rotas IPv6" {
			scanningIPv4 = false
			continue
		}

		if line == "Persistent Routes:" || line == "Rotas persistentes:" {
			if !scanningIPv4 {
				sendUpdates(routeTable)
				removeErased(&routeTable)
			}
			continue
		}

		cols := strings.Fields(line)

		n := len(cols)
		//log.Printf("cols=%d [%v]", n, cols)
		if n < 4 {
			continue
		}

		route := Route{Status: S_NEW}

		if err := parseRoute(cols, &route); err != nil {
			log.Printf("parse error: %v", err)
			continue
		}

		if route.Net == nil {
			continue
		}

		//log.Printf("route: [%v]", route)

		refreshRoute(&routeTable, route)
	}

	return scanner.Err()
}

func scanRoutes() {
	//
	// Another option: netsh interface ipv4 show route
	//
	cmd := exec.Command("netstat", "-nvr", "5")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Printf("scanRoutes: cmd: %v", err)
		return
	}
	if err := cmd.Start(); err != nil {
		log.Printf("scanRoutes: start: %v", err)
		return
	}

	if err := scanLines(stdout); err != nil {
		log.Printf("scanRoutes: scan lines: %v", err)
	}

	log.Printf("scanLines: unexpected end")

	close(routeAdd)

	if err := cmd.Wait(); err != nil {
		log.Printf("scanRoutes: wait: %v", err)
	}
}

func Routes() {
	log.Printf("compile-time operating system: windows")

	go scanRoutes()

	log.Printf("Routes: scanning route table")

	for {
		select {
		case r, ok := <-routeAdd:
			if !ok {
				break
			}
			log.Printf("route add: %v", r)
		case r := <-routeDel:
			log.Printf("route del: %v", r)
		}
	}

	log.Printf("Routes: quit")
}
