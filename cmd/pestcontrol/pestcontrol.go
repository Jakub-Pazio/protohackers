package main

import (
	"bean/pkg/pserver"
	"bufio"
	"encoding/binary"
	"flag"
	"fmt"
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

	for {
		mtype, err := readMessageType(br)

		if err != nil {
			panic(err)
		}

		l, err := readMessageLength(br)

		if err != nil {
			panic(err)
		}

		rest, err := readRemaining(br, l)

		if err != nil {
			//TODO: send error message to client and close connection
			break
		}

		switch mtype {
		case Hello:
			msg, err := ParseHello(l, rest)
			if err != nil {
				panic("TODO: could not read message")
			}
			if !ValidateChecksum(&msg) {
				panic("TODO: invalid checksum")
			}
		case SiteVisit:
			msg, err := ParseSiteVisit(l, rest)
			if err != nil {
				panic(err)
			}
			if !ValidateChecksum(&msg) {
				panic("TODO: invalid checksum")
			}

			fmt.Printf("msg: %v\n", msg)
		}
	}

}

func readRemaining(br *bufio.Reader, l int) ([]byte, error) {
	remaining := l - MsgHeaderLen

	buf := make([]byte, remaining)

	_, err := io.ReadFull(br, buf)

	if err != nil {
		return buf, fmt.Errorf("could not read whole message: %q", err)
	}

	return buf, err
}

func readMessageType(br *bufio.Reader) (Type, error) {
	b, err := br.ReadByte()
	if err != nil {
		return None, fmt.Errorf("could not read type: %v", err)
	}

	if !validMessageType(b) {
		return None, invalidMessageTypeError
	}

	return Type(b), nil
}

func readMessageLength(br *bufio.Reader) (int, error) {
	buf := make([]byte, 4)
	_, err := io.ReadFull(br, buf)

	if err != nil {
		return 0, fmt.Errorf("could not read message length: %v", err)
	}

	length := binary.BigEndian.Uint32(buf)

	if length > 1_000_000 {
		return 0, messageToLargeError
	}

	return int(length), nil
}
