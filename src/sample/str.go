package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// set gctrace then run to see garbage collection
// windows: set GODEBUG=gctrace=1

const N = 10000000

func main() {
	block := strings.Repeat("a", N)

	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Printf("waiting input from stdin\n")
		_, err := reader.ReadByte()
		if err != nil {
			fmt.Printf("stdin error: %v\n", err)
			break
		}
		push(block)
		fmt.Printf("string size=%d\n", len(buf))
	}
}

var buf string

func push(s string) {
	buf += s
	if len(buf) > N {
		buf = buf[len(buf)-N:] // can this line leak memory in underlying byte array?
	}
}
