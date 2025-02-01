package main

import (
	"net"
	"testing"
	"time"
)

func TestUnknownMessage(t *testing.T) {
	msg := []byte{0x0F, 0x21}
	serverConn, clientConn := net.Pipe()
	_ = clientConn.SetReadDeadline(time.Now().Add(2 * time.Second))

	s := NewServer()
	go s.handleConnection(serverConn)

	_, _ = clientConn.Write(msg)

	buff := make([]byte, 1024)
	n, _ := clientConn.Read(buff)

	if buff[0] != 0x10 {
		t.Errorf("wrong message, should be 0x10 and is %x\n", buff[0])
	}

	l := buff[1]
	if int(l)+2 != n {
		t.Errorf("should end %d, but got %d bytes\n", int(l)+2, n)
	}

	_, err := clientConn.Write(msg)

	if err == nil {
		t.Errorf("should error but did not (connection should been closed by the server\n")
	}
}
