package main

import (
	"bean/pkg/pserver"
	"bufio"
	"flag"
	"io"
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

	log.Println("Received valid hello message")

	replyMsg := HelloMessage{Protocol: "pestcontrol", Version: 1}
	err = WriteMessage(conn, &replyMsg)
	if err != nil {
		log.Printf("failed sending message: %v\n", err)
		conn.Close()
		return
	}

	for {
		mtype, err := ReadMessageType(br)

		if err == io.EOF {
			conn.Close()
			return
		}

		log.Printf("Got %d message type\n", mtype)

		if err != nil {
			log.Println(err)
			errMsg := ErrorMessage{Message: "error reading message type"}
			WriteMessage(conn, &errMsg)
			conn.Close()
			return
		}

		if mtype != SiteVisit {
			log.Println(err)
			errMsg := ErrorMessage{Message: "expected site visit message"}
			WriteMessage(conn, &errMsg)
			conn.Close()
			return
		}

		l, err := ReadMessageLength(br)

		if err != nil {
			log.Println(err)
			errMsg := ErrorMessage{Message: "error reading message"}
			WriteMessage(conn, &errMsg)
			conn.Close()
			return
		}

		rest, err := ReadRemaining(br, l)

		if err != nil {
			log.Println(err)
			errMsg := ErrorMessage{Message: "error reading message"}
			WriteMessage(conn, &errMsg)
			conn.Close()
			return
		}

		siteMsg, err := ParseSiteVisit(l, rest)

		if err != nil {
			log.Println(err)
			errMsg := ErrorMessage{Message: "error reading SiteVisit message"}
			WriteMessage(conn, &errMsg)
			conn.Close()
			return
		}

		log.Printf("REQ: %+v\n", siteMsg)

		ok := ValidateChecksum(&siteMsg)

		if !ok {
			log.Println("Checksum is wrong")
			errMsg := ErrorMessage{Message: "wrong checksum"}
			WriteMessage(conn, &errMsg)
			conn.Close()
			return
		}

		log.Printf("SiteVisit message is valid\n")
	}
}
