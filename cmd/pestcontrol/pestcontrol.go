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

	mtype, err := readMessageType(br)

	if err != nil {
		errMsg := ErrorMessage{Message: "could not read type"}
		writeMessage(conn, &errMsg)
		conn.Close()
		return
	}

	if mtype != Hello {
		errMsg := ErrorMessage{Message: "expected hello message"}
		writeMessage(conn, &errMsg)
		conn.Close()
		return
	}

	l, err := readMessageLength(br)

	if err != nil {
		errMsg := ErrorMessage{Message: "error reading lenght"}
		writeMessage(conn, &errMsg)
		conn.Close()
		return
	}

	rest, err := readRemaining(br, l)
	msg, err := ParseHello(l, rest)

	if err != nil {
		errMsg := ErrorMessage{Message: "error reading message"}
		writeMessage(conn, &errMsg)
		conn.Close()
		return
	}

	if msg.Protocol != "pestcontrol" {
		errMsg := ErrorMessage{Message: "unsupported protocol"}
		writeMessage(conn, &errMsg)
		conn.Close()
		return
	}

	if msg.Version != 1 {
		errMsg := ErrorMessage{Message: "unsupported version"}
		writeMessage(conn, &errMsg)
		conn.Close()
		return
	}

	if !ValidateChecksum(&msg) {
		errMsg := ErrorMessage{Message: "wrong checksum"}
		writeMessage(conn, &errMsg)
		conn.Close()
		return
	}

	for {
		mtype, err := readMessageType(br)

		if err != nil {
			errMsg := ErrorMessage{Message: "error reading message type"}
			writeMessage(conn, &errMsg)
			conn.Close()
			return
		}

		if mtype != SiteVisit {
			errMsg := ErrorMessage{Message: "expected site visit message"}
			writeMessage(conn, &errMsg)
			conn.Close()
			return
		}

		l, err := readMessageLength(br)

		if err != nil {
			errMsg := ErrorMessage{Message: "error reading message"}
			writeMessage(conn, &errMsg)
			conn.Close()
			return
		}

		rest, err := readRemaining(br, l)

		if err != nil {
			errMsg := ErrorMessage{Message: "error reading message"}
			writeMessage(conn, &errMsg)
			conn.Close()
			return
		}

		siteMsg, err := ParseSiteVisit(l, rest)

		if err != nil {
			errMsg := ErrorMessage{Message: "error reading SiteVisit message"}
			writeMessage(conn, &errMsg)
			conn.Close()
			return
		}

		ok := ValidateChecksum(&siteMsg)

		if !ok {
			errMsg := ErrorMessage{Message: "wrong checksum"}
			writeMessage(conn, &errMsg)
			conn.Close()
			return
		}

		if err = VerifyVisitSite(siteMsg); err != nil {
			errMsg := ErrorMessage{Message: err.Error()}
			writeMessage(conn, &errMsg)
			conn.Close()
			return
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

func writeMessage(conn net.Conn, msg Message) {
	conn.Write(SerializeMessage(msg))
}
