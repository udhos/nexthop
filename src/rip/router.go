package main

import (
	"fmt"
	"log"
	"net"
	"sort"
	"sync"
	"time"

	"golang.org/x/net/ipv4"

	"addr"
	"command"
	"fwd"
	"netorder"
	"sock"
)

type ripNet struct {
	addr    net.IPNet
	nexthop net.IP
	metric  int
}

type ripRoute struct {
	tag     uint16
	addr    net.IPNet
	nexthop net.IP
	metric  int

	routeChanged bool // for triggered updates
	installed    bool // for interaction with FIB

	creation          time.Time // timestamp
	timeout           time.Time // timer
	garbageCollection time.Time // timer

	// only for non-local routes
	srcExternal bool
	srcIfIndex  int
	srcIfName   string
	srcRouter   net.IP
}

func (route *ripRoute) String() string {
	return fmt.Sprintf("%v nexthop=%v metric=%d if=%s fib=%v", &route.addr, route.nexthop, route.metric, route.srcIfName, route.installed)
}

func (route *ripRoute) Family() int {
	if route.addr.IP.To4() != nil {
		return RIP_FAMILY_INET
	}
	if route.addr.IP.To16() != nil {
		return RIP_FAMILY_INET6
	}
	return RIP_FAMILY_UNSPEC
}

func newRipRoute(addr net.IPNet, nexthop net.IP, metric int, now time.Time, r *RipRouter) *ripRoute {
	newRoute := &ripRoute{addr: addr, nexthop: nexthop, metric: metric, creation: now, routeChanged: true}
	newRoute.resetTimer(now)
	r.trigUpdate(now) // since routeChanged=true, schedule triggered update
	return newRoute
}

func (r *ripRoute) resetTimer(now time.Time) {
	r.timeout = now.Add(RIP_ROUTE_TIMEOUT * time.Second) // start timeout timer
	r.garbageCollection = r.timeout.Add(RIP_ROUTE_GC * time.Second)
}

func (r *ripRoute) disable(now time.Time) {
	if r.isValid(now) {
		r.timeout = now.Add(-1 * time.Second)                     // forcedly expire timeout
		r.garbageCollection = now.Add(RIP_ROUTE_GC * time.Second) // start garbage collection timer
	}
	r.metric = RIP_METRIC_INFINITY
	if r.installed {
		// detect if we need to uninstall the route from FIB
		// because .disable() might be called repeatedly for the same route
		// but .uninstall() should be invoked only once per route
		r.uninstall()
	}
}

func (r *ripRoute) uninstall() {
	if !r.installed {
		log.Printf("ripRoute.uninstall: internal error: already uninstalled: %v", r)
	}
	r.installed = false
	log.Printf("ripRoute.uninstall: FIXME: remove route from FIB: %s", r)
	log.Printf("ripRoute.uninstall: route DOWN: %s", r)
}

func (r *ripRoute) install() {
	if r.installed {
		log.Printf("ripRoute.install: internal error: already installed: %s", r)
	}
	r.installed = true
	log.Printf("ripRoute.install: FIXME: send route to FIB: %s", r)
	log.Printf("ripRoute.install: route UP: %s", r)
}

/*
Upon expiration of the timeout, the route is no longer valid; however,
it is retained in the routing table for a short time so that neighbors
can be notified that the route has been dropped.
*/
func (r *ripRoute) isValid(now time.Time) bool {
	return r.timeout.After(now)
}

/*
Until the garbage-collection timer expires, the route is included in
all updates sent by this router.
Upon expiration of the garbage-collection timer, the route is finally
removed from the routing table.
*/
func (r *ripRoute) isGarbage(now time.Time) bool {
	if r.isValid(now) {
		return false // timeout timer is still running
	}

	return r.garbageCollection.Before(now)
}

type ripVrf struct {
	name   string
	nets   []*ripNet   // locally configured networks
	routes []*ripRoute // learnt networks
}

// Empty: VRF does not contain any data
func (v *ripVrf) Empty() bool {
	return len(v.nets) < 1
}

func (v *ripVrf) localRouteAdd(n *ripNet, r *RipRouter) {
	//log.Printf("ripVrf.localRouteAdd: vrf[%s]: %v", v.name, n)

	deleteList := []*ripRoute{}

	now := time.Now()

	for _, route := range v.routes {
		if !route.isValid(now) {
			continue // ignore invalid routes
		}
		if !addr.NetEqual(&n.addr, &route.addr) {
			continue // ignore routes for other prefixes
		}
		if n.metric > route.metric {
			return // found better metric -- refuse to change routing table
		}
		if n.metric < route.metric {
			// new route has better metric: delete existing routes
			deleteList = append(deleteList, route)
			continue
		}

		// new route has equal metric: keep existing routes

		if n.nexthop.Equal(route.nexthop) {
			log.Printf("ripVrf.localRouteAdd: internal error: duplicate prefix/nexthop/metric: vrf=[%s] route: %v", v.name, route)
			continue
		}
	}

	// delete existing routes
	for _, route := range deleteList {
		route.disable(now)
	}

	// add route
	newRoute := newRipRoute(n.addr, n.nexthop, n.metric, now, r)
	v.routeAdd(newRoute)
}

func (v *ripVrf) routeAdd(newRoute *ripRoute) {
	newRoute.install()
	v.routes = append(v.routes, newRoute)
}

func (v *ripVrf) localRouteDel(n *ripNet) {
	log.Printf("ripVrf.localRouteDel: vrf[%s]: %v", v.name, n)

	count := 0

	now := time.Now()

	for _, route := range v.routes {
		if route.srcExternal {
			continue // do not remove external routes
		}
		if !route.isValid(now) {
			continue // ignore expired routes
		}
		if !addr.NetEqual(&n.addr, &route.addr) {
			continue // ignore routes for other prefixes
		}
		if !n.nexthop.Equal(route.nexthop) {
			continue // ignore routes for other nexthops
		}
		if n.metric != route.metric {
			log.Printf("ripVrf.localRouteDel: internal error: wrong metric=%d: vrf=[%s]: %v", route.metric, v.name, route)
		}

		route.disable(now)
		count++
		if count > 1 {
			log.Printf("ripVrf.localRouteDel: internal error: removed multiple routes: count=%d: vrf=[%s]: %v", count, v.name, route)
		}
	}
}

func (v *ripVrf) nexthopGet(prefix *net.IPNet, nexthop net.IP) (int, *ripNet) {
	for i, n := range v.nets {
		if nexthop.Equal(n.nexthop) == addr.NetEqual(prefix, &n.addr) {
			return i, n
		}
	}
	return -1, nil
}

func (v *ripVrf) nexthopSet(prefix *net.IPNet, nexthop net.IP) *ripNet {
	_, n := v.nexthopGet(prefix, nexthop)
	if n == nil {
		n = v.netAdd(prefix)
		n.nexthop = nexthop
	}
	return n
}

func (v *ripVrf) netGet(prefix *net.IPNet) (int, *ripNet) {
	for i, n := range v.nets {
		if addr.NetEqual(prefix, &n.addr) {
			return i, n
		}
	}
	return -1, nil
}

func (v *ripVrf) netSet(prefix *net.IPNet) *ripNet {
	_, n := v.netGet(prefix)
	if n == nil {
		n = v.netAdd(prefix)
	}
	return n
}

func (v *ripVrf) netAdd(prefix *net.IPNet) *ripNet {

	/*
		for i, n := range v.nets {
			log.Printf("netAdd before: %d/%d vrf=%s net=%v", i, len(v.nets), v.name, n)
		}
	*/

	n := &ripNet{addr: *prefix, nexthop: net.IPv4zero, metric: 1}
	v.nets = append(v.nets, n) // add
	return n
}

func (v *ripVrf) netDel(index int) {

	/*
		for i, n := range v.nets {
			log.Printf("netDel before: %d/%d vrf=%s net=%v", i, len(v.nets), v.name, n)
		}
	*/

	last := len(v.nets) - 1
	v.nets[index] = v.nets[last] // overwrite position with last pointer
	v.nets[last] = nil           // free last pointer for garbage collection
	v.nets = v.nets[:last]       // shrink
}

func (v *ripVrf) NetAdd(prefix string, r *RipRouter) error {
	_, ipnet, err := net.ParseCIDR(prefix)
	if err != nil {
		return fmt.Errorf("ripVrf.NetAdd: parse error: addr=[%s]: %v", prefix, err)
	}
	if err1 := addr.CheckMask(ipnet); err1 != nil {
		return fmt.Errorf("ripVrf.NetAdd: bad mask: addr=[%s]: %v", prefix, err1)
	}
	_, n := v.netGet(ipnet)
	if n != nil {
		return fmt.Errorf("ripVrf.NetAdd: net exists: '%s'", prefix)
	}
	n = v.netAdd(ipnet)
	v.localRouteAdd(n, r)
	return nil
}

func (v *ripVrf) NetDel(prefix string) error {
	_, ipnet, err := net.ParseCIDR(prefix)
	if err != nil {
		return fmt.Errorf("ripVrf.NetDel: parse error: addr=[%s]: %v", prefix, err)
	}
	if err1 := addr.CheckMask(ipnet); err1 != nil {
		return fmt.Errorf("ripVrf.NetDel: bad mask: addr=[%s]: %v", prefix, err1)
	}
	i, n := v.netGet(ipnet)
	if n == nil {
		return fmt.Errorf("ripVrf.NetNet: not found: '%s'", prefix)
	}
	v.netDel(i)
	v.localRouteDel(n)
	return nil
}

func (v *ripVrf) NetNexthopAdd(prefix string, nexthop net.IP, r *RipRouter) error {
	_, ipnet, err := net.ParseCIDR(prefix)
	if err != nil {
		return fmt.Errorf("ripVrf.NetNexthopAdd: parse error: addr=[%s]: %v", prefix, err)
	}
	if err1 := addr.CheckMask(ipnet); err1 != nil {
		return fmt.Errorf("ripVrf.NetNexthopAdd: bad mask: addr=[%s]: %v", prefix, err1)
	}
	n := v.netSet(ipnet)
	n.nexthop = nexthop
	v.localRouteAdd(n, r)
	return nil
}

func (v *ripVrf) NetNexthopDel(prefix string, nexthop net.IP) error {
	_, ipnet, err := net.ParseCIDR(prefix)
	if err != nil {
		return fmt.Errorf("ripVrf.NetNexthopDel: parse error: addr=[%s]: %v", prefix, err)
	}
	if err1 := addr.CheckMask(ipnet); err1 != nil {
		return fmt.Errorf("ripVrf.NetNexthopDel: bad mask: addr=[%s]: %v", prefix, err1)
	}
	_, n := v.nexthopGet(ipnet, nexthop)
	if n == nil {
		return fmt.Errorf("ripVrf.NetNexthopDel: not found: prefix=%s nexthop=%v", prefix, nexthop)
	}
	n.nexthop = net.IPv4zero
	v.localRouteDel(n)
	return nil
}

func (v *ripVrf) NetMetricAdd(prefix string, nexthop net.IP, metric int, r *RipRouter) error {
	_, ipnet, err := net.ParseCIDR(prefix)
	if err != nil {
		return fmt.Errorf("ripVrf.NetMetricAdd: parse error: addr=[%s]: %v", prefix, err)
	}
	if err1 := addr.CheckMask(ipnet); err1 != nil {
		return fmt.Errorf("ripVrf.NetMetricAdd: bad mask: addr=[%s]: %v", prefix, err1)
	}
	n := v.nexthopSet(ipnet, nexthop)
	n.metric = metric
	v.localRouteAdd(n, r)
	return nil
}

func (v *ripVrf) NetMetricDel(prefix string, nexthop net.IP, metric int) error {
	_, ipnet, err := net.ParseCIDR(prefix)
	if err != nil {
		return fmt.Errorf("ripVrf.NetMetricDel: parse error: addr=[%s]: %v", prefix, err)
	}
	if err1 := addr.CheckMask(ipnet); err1 != nil {
		return fmt.Errorf("ripVrf.NetMetricDel: bad mask: addr=[%s]: %v", prefix, err1)
	}
	_, n := v.nexthopGet(ipnet, nexthop)
	if n == nil {
		return fmt.Errorf("ripVrf.NetMetricDel: not found: prefix=%s nexthop=%v", prefix, nexthop)
	}
	n.metric = 1
	v.localRouteDel(n)
	return nil
}

type ripInterfaceConfig struct {
	cost int
}

type RipRouter struct {
	done           chan int // write into this channel (do not close) to request end of rip router
	input          chan *udpInfo
	vrfMutex       sync.RWMutex // both main and RipRouter goroutines access the routing table (under member vrfs)
	vrfs           []*ripVrf
	ports          []*port // rip interfaces
	group          net.IP  // 224.0.0.9
	readerDone     chan int
	readerCount    int
	hardware       fwd.Dataplane
	configMutex    sync.RWMutex // both main and RipRouter goroutines access interface config
	config         map[string]*ripInterfaceConfig
	updateTicker   *time.Ticker // regular updates
	updateNext     time.Time
	triggeredTimer *time.Timer // triggered updates
	triggeredNext  time.Time
	triggeredLast  time.Time
}

func (r *RipRouter) clearInterfaceRipCost(ifname string) {
	defer r.configMutex.Unlock()
	r.configMutex.Lock()

	delete(r.config, ifname)
}

func (r *RipRouter) getInterfaceRipCost(ifname string) int {
	defer r.configMutex.RUnlock()
	r.configMutex.RLock()

	i := r.config[ifname]
	if i == nil {
		return RIP_DEFAULT_IFACE_COST // not found
	}
	return i.cost
}

func (r *RipRouter) setInterfaceRipCost(ifname string, cost int) {
	defer r.configMutex.Unlock()
	r.configMutex.Lock()

	i := r.config[ifname]
	if i == nil {
		i = &ripInterfaceConfig{}
		r.config[ifname] = i
	}
	i.cost = cost
}

const (
	RIP_PORT               = 520
	RIP_METRIC_INFINITY    = 16
	RIP_REQUEST            = 1
	RIP_RESPONSE           = 2
	RIP_FAMILY_UNSPEC      = 0  // AF_UNSPEC Unspecified
	RIP_FAMILY_INET        = 2  // AF_INET   IPv4
	RIP_FAMILY_INET6       = 10 // AF_INET6  IPv6
	RIP_V2                 = 2
	RIP_PKT_MAX_ENTRIES    = 25
	RIP_ENTRY_SIZE         = 20
	RIP_HEADER_SIZE        = 4
	RIP_PKT_MAX_SIZE       = RIP_HEADER_SIZE + RIP_ENTRY_SIZE*RIP_PKT_MAX_ENTRIES
	RIP_DEFAULT_IFACE_COST = 1
	RIP_ROUTE_TIMEOUT      = 180
	RIP_ROUTE_GC           = 120
	RIP_UPDATE_INTERVAL    = 30
)

// rip interface
type port struct {
	iface *net.Interface      // interface
	msock *sock.MulticastSock // listen-only
	send  *net.UDPConn        // send-only
}

type udpInfo struct {
	info    []byte
	src     net.UDPAddr
	dst     net.UDPAddr
	ifIndex int
	ifName  string
}

type sortByAddr []*ripRoute

func (s sortByAddr) Len() int {
	return len(s)
}
func (s sortByAddr) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s sortByAddr) Less(i, j int) bool {
	return s[i].addr.IP.String() < s[j].addr.IP.String()
}

func (r *RipRouter) ShowRoutes(c command.LineSender) {

	defer r.vrfMutex.RUnlock()
	r.vrfMutex.RLock()

	header := fmt.Sprintf("%-8s %-18s %-15s %-3s", "VRF", "NETWORK", "NEXTHOP", "MET")
	format := "%-8s %-18v %-15s %3d"

	c.Sendln("RIP local networks:")
	c.Sendln(header)

	for _, v := range r.vrfs {
		for _, n := range v.nets {
			c.Sendln(fmt.Sprintf(format, v.name, &n.addr, n.nexthop, n.metric))
		}
	}

	h := fmt.Sprintf("%s %-5s %-6s %-15s %4s %3s %-8s", header, "FLAGS", "INTERF", "NEIGHBOR", "TOUT", "GC", "UPTIME")
	f := fmt.Sprintf("%s %%-5s %%-6s %%-15s %%4d %%3d %%8s", format)

	c.Sendln("RIP routes:")
	c.Sendln("Flags: G=Garbage I=Invalid E=External F=FIB")
	c.Sendln(h)

	var routeList []*ripRoute
	var routeVrf []string

	for _, v := range r.vrfs {
		routeList = append(routeList, v.routes...)
		for range v.routes {
			routeVrf = append(routeVrf, v.name)
		}
	}

	sort.Sort(sortByAddr(routeList))

	now := time.Now()

	for i, r := range routeList {
		flags := ""
		if r.isGarbage(now) {
			flags += "G"
		}
		if !r.isValid(now) {
			flags += "I"
		}
		if r.srcExternal {
			flags += "E"
		}
		if r.installed {
			flags += "F"
		}

		var srcRouter string
		if r.srcRouter != nil {
			srcRouter = r.srcRouter.String()
		}

		timeout := int(r.timeout.Sub(now).Seconds())
		if timeout < 0 {
			timeout = 0
		}
		gc := int(r.garbageCollection.Sub(now).Seconds())
		if gc < 0 {
			gc = 0
		}
		uptime := now.Sub(r.creation)

		c.Sendln(fmt.Sprintf(f, routeVrf[i], &r.addr, r.nexthop, r.metric, flags, r.srcIfName, srcRouter, timeout, gc, uptime))
	}
}

// NewRipRouter(): Spawn new rip router.
// Write on RipRouter.done channel (do not close it) to request termination of rip router.
func NewRipRouter(hw fwd.Dataplane /*, ctx command.ConfContext*/) *RipRouter {

	RIP_GROUP := net.IPv4(224, 0, 0, 9)

	r := &RipRouter{done: make(chan int), input: make(chan *udpInfo), group: RIP_GROUP, readerDone: make(chan int), hardware: hw, config: map[string]*ripInterfaceConfig{}}

	addInterfaces(r)

	go func() {
		log.Printf("rip router: goroutine started")

		updateInterval := time.Second * time.Duration(RIP_UPDATE_INTERVAL)
		r.updateTicker = time.NewTicker(updateInterval)
		defer r.updateTicker.Stop()
		r.updateNext = time.Now().Add(updateInterval)

		r.triggeredTimer = time.NewTimer(time.Second * time.Duration(10))
		defer r.triggeredTimer.Stop()
		r.triggeredTimer.Stop() // prevent from running now

	LOOP:
		for {
			select {
			case <-r.triggeredTimer.C:
				log.Printf("rip router: FIXME WRITEME send triggered update")
				r.triggeredLast = time.Now()                                            // keep track of most recent triggered update
				r.triggeredNext = r.triggeredLast.Add(time.Second * time.Duration(-10)) // not running
			case <-r.updateTicker.C:
				r.updateNext = time.Now().Add(updateInterval)
				log.Printf("rip router: FIXME WRITEME send periodic update: nextUpdate=%v", r.updateNext)
			case <-r.done:
				// finish requested
				log.Printf("rip router: finish request received")
				delInterfaces(r) // break udpReader goroutines
			case <-r.readerDone:
				// one udpReader goroutine finished
				r.readerCount--
				if r.readerCount < 1 {
					// all udpReader goroutines finished
					break LOOP
				}
			case u, ok := <-r.input:
				if !ok {
					log.Printf("rip router: udpReader channel closed")
					break LOOP
				}
				parseRipPacket(r, u)
			}
		}

		log.Printf("rip router: goroutine finished")
	}()

	return r
}

// trigUpdate: schedule triggered update
func (r *RipRouter) trigUpdate(now time.Time) {

	if r.updateTicker == nil || r.triggeredTimer == nil {
		log.Printf("RipRouter.trigUpdate: timers not ready")
		return
	}

	if r.triggeredNext.After(now) {
		log.Printf("RipRouter.trigUpdate: timer already running: next=%v", r.triggeredNext)
		return
	}

	if r.updateNext.Sub(now) < time.Second*time.Duration(5) {
		log.Printf("RipRouter.trigUpdate: too much close to a regular update: regularNext=%v", r.updateNext)
		return
	}

	var triggeredInterval time.Duration

	if now.Sub(r.triggeredLast) < time.Second*time.Duration(5) {
		// we had a recent triggered update
		// delay update for a random interval
		log.Printf("RipRouter.trigUpdate: delaying triggered update due to recent recurrence: last=%v", r.triggeredLast)
		interval := 5
		triggeredInterval = time.Second * time.Duration(interval)
		log.Printf("RipRouter.trigUpdate: FIXME randomize interval (now using %d seconds)", interval)
	} else {
		log.Printf("RipRouter.trigUpdate: scheduling immediate update")
	}

	r.triggeredNext = now.Add(triggeredInterval)
	r.triggeredTimer.Reset(triggeredInterval)
	log.Printf("RipRouter.trigUpdate: triggered update scheduled: %v", r.triggeredNext)
}

func parseRipPacket(r *RipRouter, u *udpInfo) {
	/*
		log.Printf("parseRipPacket: recv %d bytes from %v to %v on %s ifIndex=%d",
			len(u.info), &u.src, &u.dst, u.ifName, u.ifIndex)
	*/

	size := len(u.info)
	entries := (size - RIP_HEADER_SIZE) / RIP_ENTRY_SIZE
	if entries < 1 {
		log.Printf("parseRipPacket: short packet size=%d bytes from %v to %v on %s ifIndex=%d",
			size, &u.src, &u.dst, u.ifName, u.ifIndex)
		return
	}
	if entries > RIP_PKT_MAX_ENTRIES {
		log.Printf("parseRipPacket: long packet size=%d bytes from %v to %v on %s ifIndex=%d",
			size, &u.src, &u.dst, u.ifName, u.ifIndex)
		return
	}

	cmd := u.info[0]
	version := int(u.info[1])

	/*
		log.Printf("parseRipPacket: entries=%d cmd=%d version=%d size=%d from %v to %v on %s ifIndex=%d",
			entries, cmd, version, size, &u.src, &u.dst, u.ifName, u.ifIndex)
	*/

	vrf, err := r.hardware.InterfaceVrfGet(u.ifName)
	if err != nil {
		log.Printf("parseRipPacket: unable to find VRF for interface '%s': %v", u.ifName, err)
		return
	}

	port := r.getInterfaceByIndex(u.ifIndex)
	if port == nil {
		log.Printf("ripParseRequest: unable to find RIP interface for incoming %v to %v on %s ifIndex=%d",
			&u.src, &u.dst, u.ifName, u.ifIndex)
		return
	}

	switch cmd {
	case RIP_REQUEST:
		ripParseRequest(r, u, port, size, version, entries, vrf)
	case RIP_RESPONSE:
		ripParseResponse(r, u, port, size, version, entries, vrf)
	default:
		log.Printf("parseRipPacket: unknown command %d version=%d size=%d from %v to %v on %s ifIndex=%d",
			cmd, version, size, &u.src, &u.dst, u.ifName, u.ifIndex)
	}

}

func ripParseRequest(r *RipRouter, u *udpInfo, p *port, size, version, entries int, vrf string) {
	/*
		log.Printf("ripParseRequest: entries=%d version=%d size=%d from %v to %v on %s ifIndex=%d",
			entries, version, size, &u.src, &u.dst, u.ifName, u.ifIndex)
	*/

	if entries == 1 {
		/*
			RFC2453 3.9.1 Request Messages

			There is one special case.
			If there is exactly one entry in the request,
			and it has an address family identifier of
			zero and a metric of infinity (i.e., 16),
			then this is a request to send the entire
			routing table.
		*/
		family, _, _, _, metric := parseEntry(u.info, 0)
		if family == 0 && metric == RIP_METRIC_INFINITY {
			ripSendTable(r, vrf, p, &u.src, u.ifName, u.ifIndex)
			return
		}
	}

	/*
		RFC2453 3.9.1 Request Messages

		Examine the list of RTEs in the Request one by one.  For
		each entry, look up the destination in the router's routing database
		and, if there is a route, put that route's metric in the metric field
		of the RTE.  If there is no explicit route to the specified
		destination, put infinity in the metric field.  Once all the entries
		have been filled in, change the command from Request to Response and
		send the datagram back to the requestor.
	*/

	u.info[0] = RIP_RESPONSE // change command to rip response

	// Update metric for every network in the request

	for i := 0; i < entries; i++ {
		_, _, addr, _, _ := parseEntry(u.info, i)
		route, _ := r.lookupAddressFirstMatch(vrf, addr)
		var metric int
		if route == nil {
			metric = RIP_METRIC_INFINITY
		} else {
			metric = route.metric
		}

		setEntryMetric(u.info, i, metric)
	}

	// Echo request back to source
	if err := ripSend(p, &u.src, u.info, u.ifName, u.ifIndex); err != nil {
		log.Printf("ripParseRequest: %v", err)
	}
}

func ripSendTable(r *RipRouter, vrfname string, p *port, dst *net.UDPAddr, ifname string, ifindex int) {

	defer r.vrfMutex.RUnlock()
	r.vrfMutex.RLock()

	_, v := r.vrfGet(vrfname)
	if v == nil {
		log.Printf("ripSendTable: VRF not found: vrf=[%s]", vrfname)
		return
	}

	validRoutes := []*ripRoute{}

	now := time.Now()

	for _, route := range v.routes {
		if route.isGarbage(now) {
			/*
			   Until the garbage-collection timer expires, the route
			   is included in all updates sent by this router.
			*/
			continue
		}
		validRoutes = append(validRoutes, route)
	}

	entries := len(validRoutes)
	buf := make([]byte, RIP_PKT_MAX_SIZE) // largest possible buffer

	// packet header
	buf[0] = RIP_RESPONSE // command response
	buf[1] = RIP_V2       // version 2

	// scan all valid entries
	for entry := 0; entry < entries; {

		// send batches of up to 25 entries

		bufEntries := entries - entry
		if bufEntries > RIP_PKT_MAX_ENTRIES {
			bufEntries = RIP_PKT_MAX_ENTRIES
		}
		batchSize := ripEntryOffset(bufEntries)
		b := buf[:batchSize]

		for i := 0; i < bufEntries; i++ {
			route := validRoutes[entry]
			setEntry(b, i, route.Family(), route.tag, route.addr, route.nexthop, route.metric)
			entry++
		}

		if err := ripSend(p, dst, b, ifname, ifindex); err != nil {
			log.Printf("ripSendTable: %v", err)
		}
	}
}

func ripSend(p *port, dst *net.UDPAddr, buf []byte, ifname string, ifindex int) error {

	if p.send == nil {
		log.Printf("ripSend: creating sender socket for interface '%s' ifIndex=%d dst=%v", ifname, ifindex, dst)
		localAddr := &net.UDPAddr{IP: net.IPv4zero, Port: RIP_PORT}
		var err error
		p.send, err = sock.NewUDPConn(localAddr, ifname)
		if err != nil {
			return fmt.Errorf("ripSend: error creating sender socket for interface '%s' ifIndex=%d dst=%v: %v", ifname, ifindex, dst, err)
		}
	}

	conn := p.send

	// Set 500 ms timeout
	timeout := time.Duration(500) * time.Millisecond
	deadline := time.Now().Add(timeout)
	conn.SetWriteDeadline(deadline)

	size := len(buf)

	n, err := conn.WriteToUDP(buf, dst)
	if err != nil {
		return fmt.Errorf("ripSend: error writing size=%d to %v on %s ifIndex=%d: %v", size, dst, ifname, ifindex, err)
	}
	if n != size {
		return fmt.Errorf("ripSend: partial %d/%d write to %v on %s ifIndex=%d", n, size, dst, ifname, ifindex)
	}

	log.Printf("ripSend: wrote size=%d to %v on %s ifIndex=%d", size, dst, ifname, ifindex)

	return nil
}

func ripEntryOffset(entry int) int {
	return RIP_HEADER_SIZE + RIP_ENTRY_SIZE*entry
}

func setEntryMetric(buf []byte, entry, metric int) {
	offset := ripEntryOffset(entry)
	netorder.WriteUint32(buf, offset+16, uint32(metric))
}

func setEntry(buf []byte, entry int, family int, tag uint16, netaddr net.IPNet, nexthop net.IP, metric int) {
	offset := ripEntryOffset(entry)

	netorder.WriteUint16(buf, offset, uint16(family))
	netorder.WriteUint16(buf, offset+2, tag)
	addr.WriteIPv4(buf, offset+4, netaddr.IP)
	addr.WriteIPv4Mask(buf, offset+8, netaddr.Mask)
	addr.WriteIPv4(buf, offset+12, nexthop)
	netorder.WriteUint32(buf, offset+16, uint32(metric))
}

func parseEntry(buf []byte, entry int) (family int, tag uint16, netaddr net.IPNet, nexthop net.IP, metric int) {
	offset := ripEntryOffset(entry)

	family = int(netorder.ReadUint16(buf, offset))
	tag = netorder.ReadUint16(buf, offset+2)
	netaddr = net.IPNet{IP: addr.ReadIPv4(buf, offset+4), Mask: addr.ReadIPv4Mask(buf, offset+8)}
	nexthop = addr.ReadIPv4(buf, offset+12)
	metric = int(netorder.ReadUint32(buf, offset+16))

	return
}

func ripParseResponse(r *RipRouter, u *udpInfo, p *port, size, version, entries int, vrf string) {
	/*
		log.Printf("ripParseResponse: entries=%d version=%d size=%d from %v to %v on %s ifIndex=%d",
			entries, version, size, &u.src, &u.dst, u.ifName, u.ifIndex)
	*/

	/*
		RFC2453 3.9.2 Response Messages
		The Response must be ignored if it is not from the RIP port.
	*/
	if u.src.Port != RIP_PORT {
		log.Printf("ripParseResponse: not from RIP port (520): vrf=[%s] src=%v on '%s' ifIndex=%d", vrf, u.src.IP, u.ifName, u.ifIndex)
		return
	}

	/*
		RFC2453 3.9.2 Response Messages
		the source of the datagram must be on a directly-connected network.
	*/
	ifaceAddrs, err1 := r.hardware.InterfaceAddressGet(u.ifName)
	if err1 != nil {
		log.Printf("ripParseResponse: unable to find addresses for interface %s: %v",
			u.ifName, err1)
		return
	}

	if len(ifaceAddrs) < 1 {
		log.Printf("ripParseResponse: no address on interface '%s'", u.ifName)
		return
	}

	found := false
	for _, a := range ifaceAddrs {
		found = a.Contains(u.src.IP)
		//log.Printf("ripParseResponse: if=%s addr=%v src=%v found=%v", u.ifName, a, u.src.IP, found)
		if found {
			break
		}
	}
	if !found {
		log.Printf("ripParseResponse: not directly connected response: vrf=[%s] src=%v on '%s' ifIndex=%d", vrf, u.src.IP, u.ifName, u.ifIndex)
		return // ignore response from non-directly-connected address
	}

	/*
		RFC2453 3.9.2 Response Messages
		Ignore packets from our addresses.
		(But only if is from the interface's VRF.)
	*/
	vrfAddresses, err2 := r.hardware.VrfAddresses(vrf)
	if err2 != nil {
		log.Printf("ripParseResponse: unable to find addresses for VRF %s: %v",
			vrf, err1)
		return
	}
	for _, a := range vrfAddresses {
		if a.IP.Equal(u.src.IP) {
			log.Printf("ripParseResponse: ignoring our packet from: vrf=[%s] src=%v on '%s' ifIndex=%d", vrf, u.src.IP, u.ifName, u.ifIndex)
			return // ignore packets from our addresses
		}
	}

	log.Printf("ripParseResponse: VALID RESPONSE entries=%d version=%d size=%d from %v to %v on %s ifIndex=%d",
		entries, version, size, &u.src, &u.dst, u.ifName, u.ifIndex)

	for i := 0; i < entries; i++ {
		family, tag, netaddr, nexthop, metric := parseEntry(u.info, i)

		/*
			log.Printf("ripParseResponse: entry=%d/%d family=%d tag=%d net=%v nexthop=%v metric=%d",
				i, entries, family, tag, &netaddr, nexthop, metric)
		*/

		if metric < 1 || metric > RIP_METRIC_INFINITY {
			log.Printf("ripParseResponse: bad metric entry=%d/%d family=%d tag=%d net=%v nexthop=%v metric=%d from %v to %v on %s ifIndex=%d",
				i, entries, family, tag, &netaddr, nexthop, metric, &u.src, &u.dst, u.ifName, u.ifIndex)
			continue // ignore entry with bad metric
		}

		newMetric := metric + r.getInterfaceRipCost(u.ifName)
		if newMetric > RIP_METRIC_INFINITY {
			newMetric = RIP_METRIC_INFINITY
		}

		r.extRouteAdd(vrf, tag, netaddr, nexthop, newMetric, u.ifIndex, u.ifName, u.src.IP)
	}

}

func (r *RipRouter) extRouteAdd(vrfname string, tag uint16, netaddr net.IPNet, nexthop net.IP, metric, ifindex int, ifname string, router net.IP) {

	defer r.vrfMutex.Unlock()
	r.vrfMutex.Lock()

	_, v := r.vrfGet(vrfname)
	if v == nil {
		log.Printf("RipRouter.routeAdd: VRF not found: vrf=[%s]", vrfname)
		return
	}

	now := time.Now()

	for _, route := range v.routes {
		if !route.isValid(now) {
			continue // ignore invalid routes
		}
		if !addr.NetEqual(&netaddr, &route.addr) {
			continue // ignore routes for other prefixes
		}

		/*
			If there is an existing route, compare the next hop
			address to the address of the router from which the
			datagram came.  If this datagram is from the same
			router as the existing route, reinitialize the
			timeout.
		*/
		sameNexthop := route.nexthop.Equal(router)
		if sameNexthop {
			route.resetTimer(now)
		}

		if metric < route.metric {
			// new route will be better, remove the old (current) one
			route.disable(now)
			continue
		}

		if sameNexthop {

			// same prefix/nexthop

			if metric != route.metric {
				// only update metric
				route.metric = metric
				route.routeChanged = true
				r.trigUpdate(now) // schedule triggered update
			} // else: exact same prefix/nexthop/metric: do nothing (timer was reset above)

			return // do not add route below
		}

		// distinct nexthop

		if metric > route.metric {
			return // new nexthop with worse metric: do not add route below
		}

		// distinc nexthop with same metric: could add route below as ECMP
	}

	// add new external route

	if metric == RIP_METRIC_INFINITY {
		return // refuse to add new route with metric infinity
	}

	// Set the next hop address to be the address of the router
	// from which the datagram came
	newNexthop := router

	newRoute := newRipRoute(netaddr, newNexthop, metric, now, r)
	newRoute.srcExternal = true
	newRoute.srcIfIndex = ifindex
	newRoute.srcIfName = ifname
	newRoute.srcRouter = router
	v.routeAdd(newRoute)
}

func (r *RipRouter) lookupAddressFirstMatch(vrfname string, netaddr net.IPNet) (*ripRoute, error) {

	defer r.vrfMutex.Unlock()
	r.vrfMutex.Lock()

	_, v := r.vrfGet(vrfname)
	if v == nil {
		return nil, fmt.Errorf("lookupAddressFirstMatch: VRF not found: vrf=[%s]", vrfname)
	}

	now := time.Now()

	for _, route := range v.routes {
		if !route.isValid(now) {
			continue
		}
		if addr.NetEqual(&route.addr, &netaddr) {
			return route, nil
		}
	}

	return nil, fmt.Errorf("lookupAddressFirstMatch: prefix not found: vrf=[%s] prefix=[%v]", vrfname, &netaddr)
}

func (r *RipRouter) vrfSet(vrf string) *ripVrf {
	_, v := r.vrfGet(vrf)
	if v == nil {
		v = r.vrfAdd(vrf)
	}
	return v
}

func (r *RipRouter) vrfGet(vrf string) (int, *ripVrf) {
	/*
		for i, v := range r.vrfs {
			log.Printf("vrfGet: %v %d/%d", v.name, i, len(r.vrfs))
		}
	*/
	for i, v := range r.vrfs {
		if v.name == vrf {
			return i, v // found
		}
	}
	return -1, nil // not found
}

func (r *RipRouter) vrfAdd(vrf string) *ripVrf {
	//log.Printf("vrfAdd: %s size=%d", vrf, len(r.vrfs))

	v := &ripVrf{name: vrf}
	r.vrfs = append(r.vrfs, v)
	return v
}

func (r *RipRouter) vrfDel(index int) {
	//log.Printf("vrfDel: %s size=%d", r.vrfs[index].name, len(r.vrfs))

	last := len(r.vrfs) - 1
	r.vrfs[index] = r.vrfs[last] // overwrite position with last pointer
	r.vrfs[last] = nil           // free last pointer for garbage collection
	r.vrfs = r.vrfs[:last]       // shrink
}

func (r *RipRouter) NetAdd(vrf, netAddr string) error {
	defer r.vrfMutex.Unlock()
	r.vrfMutex.Lock()

	v := r.vrfSet(vrf)
	err := v.NetAdd(netAddr, r)

	//log.Printf("RipRouter.NetAdd(%s,%s) after:", vrf, netAddr)
	//r.dump(&ripDumper{})

	return err
}

func (r *RipRouter) NetDel(vrf, netAddr string) error {
	defer r.vrfMutex.Unlock()
	r.vrfMutex.Lock()

	i, v := r.vrfGet(vrf)
	if v == nil {
		return fmt.Errorf("RipRouter.NetDel: vrf not found: vrf=[%s] addr=[%s]", vrf, netAddr)
	}
	err := v.NetDel(netAddr) // remove net from VRF
	if v.Empty() {
		r.vrfDel(i)
	}
	return err
}

func (r *RipRouter) NetNexthopAdd(vrf, netAddr string, nexthop net.IP) error {
	defer r.vrfMutex.Unlock()
	r.vrfMutex.Lock()

	v := r.vrfSet(vrf)
	return v.NetNexthopAdd(netAddr, nexthop, r)
}

func (r *RipRouter) NetNexthopDel(vrf, netAddr string, nexthop net.IP) error {
	defer r.vrfMutex.Unlock()
	r.vrfMutex.Lock()

	i, v := r.vrfGet(vrf)
	if v == nil {
		return fmt.Errorf("RipRouter.NetNexthopDel: vrf not found: vrf=[%s] addr=[%s]", vrf, netAddr)
	}
	err := v.NetNexthopDel(netAddr, nexthop)
	if v.Empty() {
		r.vrfDel(i)
	}
	return err
}

func (r *RipRouter) NetMetricAdd(vrf, netAddr string, nexthop net.IP, metric int) error {
	defer r.vrfMutex.Unlock()
	r.vrfMutex.Lock()

	v := r.vrfSet(vrf)
	err := v.NetMetricAdd(netAddr, nexthop, metric, r)

	//log.Printf("RipRouter.NetMetricAdd(%s,%s,%v,%d) after:", vrf, netAddr, nexthop, metric)
	//r.dump(&ripDumper{})

	return err
}

func (r *RipRouter) NetMetricDel(vrf, netAddr string, nexthop net.IP, metric int) error {
	defer r.vrfMutex.Unlock()
	r.vrfMutex.Lock()

	i, v := r.vrfGet(vrf)
	if v == nil {
		return fmt.Errorf("RipRouter.NetMetricDel: vrf not found: vrf=[%s] addr=[%s]", vrf, netAddr)
	}
	err := v.NetMetricDel(netAddr, nexthop, metric)
	if v.Empty() {
		r.vrfDel(i)
	}
	return err
}

func addInterfaces(r *RipRouter) {
	ifList, err1 := net.Interfaces()
	if err1 != nil {
		log.Printf("NewRipRouter: could not find local interfaces: %v", err1)
	}
	for _, i := range ifList {
		if err := r.InterfaceAdd(i.Name); err != nil {
			log.Printf("NewRipRouter: error adding interface '%s': %v", i.Name, err)
		}
	}
}

func (r *RipRouter) InterfaceAdd(s string) error {

	for _, p := range r.ports {
		if s == p.iface.Name {
			return fmt.Errorf("RipRouter.InterfaceAdd: interface '%s' exists", s)
		}
	}

	ifi, err1 := net.InterfaceByName(s)
	if err1 != nil {
		return err1
	}

	return r.Join(ifi)
}

func (r *RipRouter) getInterfaceByIndex(ifIndex int) *port {

	for _, p := range r.ports {
		if ifIndex == p.iface.Index {
			return p
		}
	}

	return nil
}

func (r *RipRouter) Join(ifi *net.Interface) error {

	ripPort := RIP_PORT

	m, err1 := sock.MulticastListener(ripPort, ifi.Name)
	if err1 != nil {
		return fmt.Errorf("RipRouter.Join: open: %v", err1)
	}

	if err := sock.Join(m, r.group, ifi.Name); err != nil {
		sock.Close(m)
		return fmt.Errorf("RipRouter.Join: join: %v", err)
	}

	newPort := &port{iface: ifi, msock: m}

	r.ports = append(r.ports, newPort)

	go udpReader(m.P, r.input, ifi.Name, r.readerDone, ripPort)

	r.readerCount++

	return nil
}

func delInterfaces(r *RipRouter) {
	for i := range r.ports {
		r.ifClose(i)
	}
	r.ports = nil // cleanup
}

func udpReader(c *ipv4.PacketConn, input chan<- *udpInfo, ifname string, readerDone chan<- int, listenPort int) {

	log.Printf("udpReader: reading from '%s'", ifname)

	defer c.Close()

	buf := make([]byte, 10000)

LOOP:
	for {
		n, cm, srcAddr, err1 := c.ReadFrom(buf)
		if err1 != nil {
			log.Printf("udpReader: ReadFrom: error %v", err1)
			break LOOP
		}

		var udpSrc *net.UDPAddr

		switch srcAddr.(type) {
		case *net.UDPAddr:
			udpSrc = srcAddr.(*net.UDPAddr)
		}

		var name string

		var ifi *net.Interface
		var err2 error

		if cm != nil {
			ifi, err2 = net.InterfaceByIndex(cm.IfIndex)
			if err2 != nil {
				log.Printf("udpReader: unable to solve ifIndex=%d: error: %v", cm.IfIndex, err2)
			}
		}

		if ifi == nil {
			name = "ifname?"
		} else {
			name = ifi.Name
		}

		udpDst := net.UDPAddr{IP: cm.Dst, Port: listenPort}

		//log.Printf("udpReader: recv %d bytes from %v to %v on %s ifIndex=%d", n, udpSrc, &udpDst, name, cm.IfIndex)

		// make a copy because we will overwrite buf
		b := make([]byte, n)
		copy(b, buf)

		// deliver udp packet to main rip goroutine
		input <- &udpInfo{info: b, src: *udpSrc, dst: udpDst, ifIndex: cm.IfIndex, ifName: name}
	}

	log.Printf("udpReader: exiting '%s' -- trying", ifname)
	readerDone <- 1 // tell rip router goroutine
	log.Printf("udpReader: exiting '%s'", ifname)
}

func (r *RipRouter) InterfaceDel(s string) error {
	log.Printf("RipRouter.InterfaceDel: %s", s)

	for i, p := range r.ports {
		if s == p.iface.Name {
			// found interface
			r.ifClose(i)
			r.ifDel(i)
			return nil
		}
	}

	return fmt.Errorf("RipRouter.InterfaceDel: interface '%s' not found", s)
}

func (r *RipRouter) ifClose(i int) {
	p := r.ports[i]

	log.Printf("RipRouter.ifDel: %s", p.iface.Name)

	if err := sock.Leave(p.msock, r.group, p.iface); err != nil {
		// warning only
		log.Printf("RipRouter.ifClose: leave group error: %v", err)
	}

	sock.Close(p.msock) // break reader goroutine
}

func (r *RipRouter) ifDel(i int) {
	size := len(r.ports)
	r.ports[i] = r.ports[size-1]
	r.ports = r.ports[:size-1]
}

/*
type ripDumper struct{}

func (d *ripDumper) Sendln(msg string) int {
	log.Printf(msg)
	return 0
}

func (r *RipRouter) dumpNets(c command.LineSender) {
	for _, v := range r.vrfs {
		c.Sendln(fmt.Sprintf("vrf %s", v.name))
		v.dumpNets(c)
	}
}

func (v *ripVrf) dumpNets(c command.LineSender) {
	for _, n := range v.nets {
		c.Sendln(fmt.Sprintf("vrf %s - %v/%v/%d", v.name, n.addr, n.nexthop, n.metric))
	}
}
*/
