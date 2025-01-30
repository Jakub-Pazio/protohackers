package main

import (
	"bean/pkg/pserver"
	"bufio"
	"flag"
	"log"
	"net"
)

const BuffSize = 1024

var portNumber = flag.Int("port", 4242, "Port number of server")

func main() {
	flag.Parse()
	server := NewServer()
	handler := pserver.WithMiddleware(
		server.handleConnection,
		pserver.LoggingMiddleware,
	)

	log.Fatal(pserver.ListenServe(handler, *portNumber))
}

type Server struct {
}

func NewServer() *Server {
	return &Server{}
}

func (s *Server) handleConnection(conn net.Conn) {
	defer pserver.HandleConnShutdown(conn)
	reader := bufio.NewReader(conn)
	for {
		msgType, err := reader.ReadByte()
		if err != nil {
			log.Printf("error reading type: %q", err)
		}
		log.Printf("msgType: %v", msgType)
	}
}
