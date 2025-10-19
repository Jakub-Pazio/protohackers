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
		log.Println(err)
		errMsg := ErrorMessage{Message: "could not read type"}
		writeMessage(conn, &errMsg)
		conn.Close()
		return
	}

	if mtype != Hello {
		log.Println("message is not hello")
		errMsg := ErrorMessage{Message: "expected hello message"}
		writeMessage(conn, &errMsg)
		conn.Close()
		return
	}

	l, err := readMessageLength(br)
	if err != nil {
		log.Println(err)
		errMsg := ErrorMessage{Message: "error reading lenght"}
		writeMessage(conn, &errMsg)
		conn.Close()
		return
	}

	rest, err := readRemaining(br, l)
	if err != nil {
		log.Println(err)
		errMsg := ErrorMessage{Message: "error reading message"}
		writeMessage(conn, &errMsg)
		conn.Close()
		return
	}

	msg, err := ParseHello(l, rest)
	if err != nil {
		log.Println(err)
		errMsg := ErrorMessage{Message: "error reading message"}
		writeMessage(conn, &errMsg)
		conn.Close()
		return
	}

	if msg.Protocol != "pestcontrol" {
		log.Printf("unknown protocol: %q\n", msg.Protocol)
		errMsg := ErrorMessage{Message: "unsupported protocol"}
		writeMessage(conn, &errMsg)
		conn.Close()
		return
	}

	if msg.Version != 1 {
		log.Printf("unsupported version %d\n", msg.Version)
		errMsg := ErrorMessage{Message: "unsupported version"}
		writeMessage(conn, &errMsg)
		conn.Close()
		return
	}

	if !ValidateChecksum(&msg) {
		log.Println("wrong checksum")
		errMsg := ErrorMessage{Message: "wrong checksum"}
		writeMessage(conn, &errMsg)
		conn.Close()
		return
	}

	log.Println("Received valid hello message")

	replyMsg := HelloMessage{Protocol: "pestcontrol", Version: 1}
	err = writeMessage(conn, &replyMsg)
	if err != nil {
		log.Printf("failed sending message: %v\n", err)
		conn.Close()
		return
	}

	for {
		mtype, err := readMessageType(br)

		if err == io.EOF {
			conn.Close()
			return
		}

		if err != nil {
			log.Println(err)
			errMsg := ErrorMessage{Message: "error reading message type"}
			writeMessage(conn, &errMsg)
			conn.Close()
			return
		}

		if mtype != SiteVisit {
			log.Println(err)
			errMsg := ErrorMessage{Message: "expected site visit message"}
			writeMessage(conn, &errMsg)
			conn.Close()
			return
		}

		l, err := readMessageLength(br)

		if err != nil {
			log.Println(err)
			errMsg := ErrorMessage{Message: "error reading message"}
			writeMessage(conn, &errMsg)
			conn.Close()
			return
		}

		rest, err := readRemaining(br, l)

		if err != nil {
			log.Println(err)
			errMsg := ErrorMessage{Message: "error reading message"}
			writeMessage(conn, &errMsg)
			conn.Close()
			return
		}

		siteMsg, err := ParseSiteVisit(l, rest)

		if err != nil {
			log.Println(err)
			errMsg := ErrorMessage{Message: "error reading SiteVisit message"}
			writeMessage(conn, &errMsg)
			conn.Close()
			return
		}

		ok := ValidateChecksum(&siteMsg)

		if !ok {
			log.Println("Checksum is wrong")
			errMsg := ErrorMessage{Message: "wrong checksum"}
			writeMessage(conn, &errMsg)
			conn.Close()
			return
		}

		if err = VerifyVisitSite(siteMsg); err != nil {
			log.Println(err)
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

func writeMessage(conn net.Conn, msg Message) error {
	_, err := conn.Write(SerializeMessage(msg))
	return err
}
