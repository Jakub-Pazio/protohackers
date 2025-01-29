package main

import (
	"io"
	"net"
	"testing"
	"time"
)

func TestHandler(t *testing.T) {
	var echotests = []struct {
		input string
	}{
		{"some data"},
		{"lorem ipsum lorem ipsum lorem ipsum lorem ipsum lorem ipsum lorem ipsum"},
		{""},
	}

	for _, tt := range echotests {
		t.Run("check if (echo) server respond with same data", func(t *testing.T) {
			data := tt.input
			serverConn, clientConn := net.Pipe()

			// We want to wait for the whole message to arrive or for at most 2 seconds
			_ = clientConn.SetReadDeadline(time.Now().Add(2 * time.Second))
			go handleConnection(serverConn)

			_, err := clientConn.Write([]byte(data))
			if err != nil {
				t.Fatalf("could not write data to sever: %v", err)
			}
			buf := make([]byte, len(data))
			_, err = io.ReadFull(clientConn, buf)
			if err != nil {
				t.Fatalf("could not read data from sever: %v", err)
			}
			if string(buf) != data {
				t.Fatalf("exprected %q data, got %q", data, string(buf))
			}
		})
	}
}
