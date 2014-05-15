package telnet

import (
	"fmt"
	"log"
	"net"
)

type HandlerFunc func(conn net.Conn)

type Server struct {
	Addr    string
	Handler HandlerFunc
}

func (s *Server) ListenAndServe() error {

	listener, err := net.Listen("tcp", s.Addr)
	if err != nil {
		return fmt.Errorf("telnet.ListenAndServe: listen on TCP %s: %s", s.Addr, err)
	}

	defer listener.Close()

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("telnet.ListenAndServe: accept on TCP %s: %s", s.Addr, err)
			continue
		}
		go s.Handler(conn)
	}
}
