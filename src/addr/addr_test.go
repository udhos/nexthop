package addr

import (
	"net"
	"testing"
)

func TestIntersect(t *testing.T) {
	_, net1, _ := net.ParseCIDR("1.1.1.1/24")
	_, net2, _ := net.ParseCIDR("1.1.0.2/16")
	_, net3, _ := net.ParseCIDR("1.1.1.3/25")
	_, net4, _ := net.ParseCIDR("1.2.0.4/16")

	test(t, net1, net2, true)
	test(t, net2, net1, true)
	test(t, net1, net3, true)
	test(t, net3, net1, true)
	test(t, net1, net4, false)
	test(t, net4, net1, false)
}

func test(t *testing.T, n1, n2 *net.IPNet, expected bool) {
	result := netIntersect(n1, n2)
	if result != expected {
		t.Errorf("intersect(%v,%v)=%v expected=%v", n1, n2, result, expected)
	}
}
