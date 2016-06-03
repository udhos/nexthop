package route

// See http://golang.org/pkg/go/build/

import (
	"log"
)

func Routes() (chan Route, chan Route) {
	log.Printf("Routes(): compile-time operating system: linux")
	return nil, nil
}
