package command

import (
	"fmt"
	"log"
	"net"
	"strconv"
	"unicode"

	"addr"
)

type interfaceListFunc func() ([]string, []string) // ifname, ifvrf
type matchFunc func(label string) error
type optionsFunc func() []string

type keyword struct {
	label   string
	match   matchFunc
	options optionsFunc
}

type keywordTable struct {
	table      map[string]keyword
	ifScanFunc interfaceListFunc
}

var keyword_table = keywordTable{table: map[string]keyword{}}

func IsUserPatternKeyword(str string) bool {
	size := len(str)
	if size < 3 {
		return false
	}
	return str[0] == '{' && str[size-1] == '}'
}

func LoadKeywordTable(ifScannerFunc interfaceListFunc, commitScannerFunc optionsFunc) {

	keyword_table.table = map[string]keyword{} // discard previous map

	// test interface list
	ifaces, vrfs := ifScannerFunc()
	size := len(ifaces)
	if size < 1 {
		log.Printf("LoadKeywordTable: error: interface list func returned empty set")
	}
	for i := 0; i < size; i++ {
		vrfname := vrfs[i]
		ifname := ifaces[i]
		log.Printf("LoadKeywordTable: vrf=[%s] interface=%s", vrfname, ifname)
	}

	keyword_table.ifScanFunc = ifScannerFunc

	listIf := func() []string {
		ifnames, _ := ifScannerFunc()
		return ifnames
	}

	keywordAdd("{ANY}", matchAny, nil)
	keywordAdd("{IFNAME}", matchIfName, listIf)
	keywordAdd("{IFADDR}", matchIfAddr, nil)
	keywordAdd("{IFADDR6}", matchIfAddr6, nil)
	keywordAdd("{COMMITID}", matchCommitId, commitScannerFunc)
	keywordAdd("{NETWORK}", matchNetwork, nil)
	keywordAdd("{RIPMETRIC}", matchRipMetric, nil)
}

func MatchKeyword(word, label string) error {
	requirePattern(word)

	err := matchK(word, label)

	//log.Printf("MatchKeyword: pattern=%s string=%s error=%v", word, label, err)

	return err
}

func matchK(word, label string) error {
	kw, found := keyword_table.table[word]
	if !found {
		// accept unknown keyword
		//log.Printf("MatchKeyword: accepting unknown keyword: '%s'", word)
		return nil
	}

	if err := kw.match(label); err != nil {
		return fmt.Errorf("bad value=%s for pattern=%s: %v", label, word, err)
	}

	return nil // accept
}

func findKeyword(word string) *keyword {
	requirePattern(word)

	kw, found := keyword_table.table[word]
	if !found {
		return nil
	}

	return &kw
}

func keywordAdd(word string, m matchFunc, o optionsFunc) {
	requireIfScanner()
	requirePattern(word)
	if _, found := keyword_table.table[word]; found {
		log.Fatalf("keywordAdd: duplicate keyword=%s", word)
	}
	kw := keyword{label: word, match: m, options: o}
	keyword_table.table[word] = kw
	log.Printf("keywordAdd: new keyword registered: '%s'", word)
}

func requireIfScanner() {
	if keyword_table.ifScanFunc == nil {
		msg := fmt.Sprintf("missing interface scanner func")
		log.Printf(msg)
		panic(msg)
	}
}

func requirePattern(word string) {
	if !IsUserPatternKeyword(word) {
		msg := fmt.Sprintf("not a keyword pattern: '%s'", word)
		log.Printf(msg)
		panic(msg)
	}
}

func matchAny(str string) error {
	return fmt.Errorf("cannot match against {ANY}")
}

func matchIfAddr(ifaddr string) error {
	ip, _, err := net.ParseCIDR(ifaddr)
	if err != nil {
		return err
	}
	ip4 := ip.To4()
	if ip4 == nil {
		return fmt.Errorf("address '%s' is not IPv4", ifaddr)
	}
	return nil // accept
}

func matchIfAddr6(ifaddr string) error {
	ip, _, err := net.ParseCIDR(ifaddr)
	if err != nil {
		return err
	}
	ip6 := ip.To16()
	if ip6 == nil {
		return fmt.Errorf("address '%s' is not IPv6", ifaddr)
	}
	return nil // accept
}

func matchIfName(ifname string) error {
	requireIfScanner()
	ifNames, _ := keyword_table.ifScanFunc()
	for _, i := range ifNames {
		if i == ifname {
			return nil // found interface
		}
	}
	return fmt.Errorf("interface '%s' does not exist", ifname)
}

func matchCommitId(id string) error {
	for i, d := range id {
		if !unicode.IsDigit(d) {
			return fmt.Errorf("non-digit char in commit id '%s': decimal=%d (index %d)", id, d, i)
		}
	}
	if _, err := strconv.Atoi(id); err != nil {
		return fmt.Errorf("could not parse commit id '%s': %v", id, err)
	}
	return nil // accept
}

func matchNetwork(s string) error {
	_, n, err := net.ParseCIDR(s)
	if err != nil {
		return fmt.Errorf("matchNetwork: parse error addr=[%s]: %v", s, err)
	}
	if err1 := addr.CheckMask(n); err1 != nil {
		return fmt.Errorf("matchNetwork: bad mask: addr=[%s]: %v", s, err1)
	}
	return nil // accept
}

func matchRipMetric(costStr string) error {

	cost, err := strconv.Atoi(costStr)
	if err != nil {
		return fmt.Errorf("bad rip network metric: '%s': %v", costStr, err)
	}

	if cost < 1 || cost > 15 {
		return fmt.Errorf("invalid rip network cost: %d", cost)
	}

	return nil // accept
}
