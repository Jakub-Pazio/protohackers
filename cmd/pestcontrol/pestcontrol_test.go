package main

import (
	"bufio"
	"io"
	"testing"
)

func TestReadType(t *testing.T) {
	r, w := io.Pipe()

	go func() {
		msg := []byte{0x50, 0x00, 0x00, 0x00, 0x24}

		w.Write(msg)
	}()

	br := bufio.NewReader(r)

	got, err := ReadMessageType(br)
	if err != nil {
		t.Errorf("unexpected error: %v\n", err)
	}

	want := Hello

	if got != want {
		t.Errorf("got %v, want %v\n", got, want)
	}

	len, err := ReadMessageLength(br)

	lwant := 36
	if len != lwant {
		t.Errorf("got %d, want %d\n", len, lwant)
	}
}

func TestReadTooLarge(t *testing.T) {
	r, w := io.Pipe()

	go func() {
		msg := []byte{0x50, 0xF0, 0x00, 0x00, 0x24}

		w.Write(msg)
	}()

	br := bufio.NewReader(r)

	got, err := ReadMessageType(br)
	if err != nil {
		t.Errorf("unexpected error: %v\n", err)
	}

	want := Hello

	if got != want {
		t.Errorf("got %v, want %v\n", got, want)
	}

	len, err := ReadMessageLength(br)

	lwant := 0
	if len != lwant {
		t.Errorf("got %d, want %d\n", len, lwant)
	}

	if err == nil {
		t.Errorf("expected error but got none\n")
	}

	if err != MessageToLargeError {
		t.Errorf("incorrect error: %q\n", err)
	}
}

func TestReadUnknownType(t *testing.T) {
	r, w := io.Pipe()

	go func() {
		msg := []byte{0x01}

		w.Write(msg)
	}()

	br := bufio.NewReader(r)

	got, err := ReadMessageType(br)

	if err == nil {
		t.Errorf("expected error, but got no")
	}

	if err != InvalidMessageTypeError {
		t.Errorf("wrong error type, got: %q", err)
	}

	want := None

	if got != want {
		t.Errorf("got %v, want %v\n", got, want)
	}
}

func TestReadSiteVisit(t *testing.T) {
	r, w := io.Pipe()

	go func() {
		msg := []byte{0x58,
			0x00, 0x00, 0x00, 0x24,
			0x00, 0x00, 0x30, 0x39,
			0x00, 0x00, 0x00, 0x02,
			0x00, 0x00, 0x00, 0x03,
			0x64, 0x6f, 0x67,
			0x00, 0x00, 0x00, 0x01,
			0x00, 0x00, 0x00, 0x03,
			0x72, 0x61, 0x74,
			0x00, 0x00, 0x00, 0x05,
			0x8c}

		w.Write(msg)
	}()

	br := bufio.NewReader(r)

	got, err := ReadMessageType(br)

	if err != nil {
		t.Errorf("Unexpected error: %v\n", err)
	}

	if got != SiteVisit {
		t.Errorf("Expected SiteVisit, got %d\n", got)
	}
	l, err := ReadMessageLength(br)

	if err != nil {
		t.Errorf("Unexprected error: %v\n", err)
	}

	rest, err := ReadRemaining(br, l)

	if err != nil {
		t.Errorf("Unexpected error: %v\n", err)
	}

	t.Log(rest)

	// msg, err := ParseSiteVisit(l, rest)
	//
	// ok2 := ValidateChecksum(msg)
	//
	// if !ok2 {
	// 	t.Errorf("check sum is not correct\n")
	// }
	//
}
