package command

import (
	"fmt"
	"log"
	"net"
)

type interfaceScanner interface {
	interfaceList() []string
}

type matchFunc func(label string) error
type strListFunc func() []string

type keyword struct {
	label string
	match matchFunc
}

type keywordTable struct {
	table      map[string]keyword
	ifScanFunc strListFunc // list existing interface names
}

var keyword_table = keywordTable{table: map[string]keyword{}}

func IsUserPatternKeyword(str string) bool {
	size := len(str)
	if size < 3 {
		return false
	}
	return str[0] == '{' && str[size-1] == '}'
}

func LoadKeywordTable(ifScannerFunc strListFunc) {
	keyword_table.ifScanFunc = ifScannerFunc

	keywordAdd("{ANY}", matchAny)
	keywordAdd("{IFNAME}", matchIfName)
	keywordAdd("{IFADDR}", matchIfAddr)
}

func MatchKeyword(word, label string) error {
	requirePattern(word)
	kw, found := keyword_table.table[word]
	if !found {
		return nil // accept unknown keyword
	}

	if err := kw.match(label); err != nil {
		return fmt.Errorf("bad value=%s for pattern=%s: %v", label, word, err)
	}

	return nil // accept
}

func keywordAdd(word string, m matchFunc) {
	requireIfScanner()
	requirePattern(word)
	if _, found := keyword_table.table[word]; found {
		log.Fatalf("keywordAdd: duplicate keyword=%s", word)
	}
	kw := keyword{label: word, match: m}
	keyword_table.table[word] = kw
}

func requireIfScanner() {
	if keyword_table.ifScanFunc == nil {
		log.Fatalf("missing interface scanner func")
	}
}

func requirePattern(word string) {
	if !IsUserPatternKeyword(word) {
		log.Fatalf("not a keyword pattern: '%s'", word)
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
	if ip4 := ip.To4(); ip4 == nil {
		return fmt.Errorf("address '%s' is not IPv4", ifaddr)
	}
	return nil // accept
}

func matchIfName(ifname string) error {
	requireIfScanner()
	ifNames := keyword_table.ifScanFunc()
	for _, i := range ifNames {
		if i == ifname {
			return nil // found interface
		}
	}
	return fmt.Errorf("interface '%s' does not exist", ifname)
}
