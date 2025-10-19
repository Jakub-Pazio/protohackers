package main

import (
	"bean/pkg/pserver"
	"bufio"
	"flag"
	"log"
	"net"
)

var portNumber = flag.Int("port", 4242, "Port number of server")

func main() {

	flag.Parse()

	s := Server{}

	handler := pserver.WithMiddleware(
		s.handleConnection,
		pserver.LoggingMiddleware,
	)

	log.Fatal(pserver.ListenServe(handler, *portNumber))
}

type Server struct {
}

func (s *Server) handleConnection(conn net.Conn) {
	br := bufio.NewReader(conn)

	_, err := ReadHelloMessage(br)
	if err != nil {
		log.Printf("Error reading Hello message: %v\n", err)
		errMsg := &ErrorMessage{Message: err.Error()}
		WriteMessage(conn, errMsg)
		conn.Close()
		return
	}
	log.Println("Received valid hello message")

	replyMsg := &HelloMessage{Protocol: "pestcontrol", Version: 1}
	err = WriteMessage(conn, replyMsg)
	if err != nil {
		log.Printf("failed sending message: %v\n", err)
		conn.Close()
		return
	}

	for {
		msg, err := ReadSiteVisitMessage(br)
		if err != nil {
			log.Printf("Error reading SiteVisit message: %q\n", err)
			errMsg := &ErrorMessage{Message: err.Error()}
			WriteMessage(conn, errMsg)
			conn.Close()
			return
		}

		log.Printf("Recived SiteVisit message: %+v\n", msg)
	}
}
