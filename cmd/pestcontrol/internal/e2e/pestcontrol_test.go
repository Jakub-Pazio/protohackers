package e2e

import (
	"bean/cmd/pestcontrol/internal/message"
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

	got, err := message.ReadMessageType(br)
	if err != nil {
		t.Errorf("unexpected error: %v\n", err)
	}

	want := message.MessageTypeHello

	if got != want {
		t.Errorf("got %v, want %v\n", got, want)
	}

	len, err := message.ReadMessageLength(br)

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

	got, err := message.ReadMessageType(br)
	if err != nil {
		t.Errorf("unexpected error: %v\n", err)
	}

	want := message.MessageTypeHello

	if got != want {
		t.Errorf("got %v, want %v\n", got, want)
	}

	len, err := message.ReadMessageLength(br)

	lwant := 0
	if len != lwant {
		t.Errorf("got %d, want %d\n", len, lwant)
	}

	if err == nil {
		t.Errorf("expected error but got none\n")
	}

	if err != message.ErrMessageToLarge {
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

	got, err := message.ReadMessageType(br)

	if err == nil {
		t.Errorf("expected error, but got no")
	}

	if err != message.ErrInvalidMessageType {
		t.Errorf("wrong error type, got: %q", err)
	}

	want := message.MessageTypeNone

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

	got, err := message.ReadMessageType(br)

	if err != nil {
		t.Errorf("Unexpected error: %v\n", err)
	}

	if got != message.MessageTypeSiteVisit {
		t.Errorf("Expected SiteVisit, got %d\n", got)
	}
	l, err := message.ReadMessageLength(br)

	if err != nil {
		t.Errorf("Unexprected error: %v\n", err)
	}

	rest, err := message.ReadRemaining(br, l)

	if err != nil {
		t.Errorf("Unexpected error: %v\n", err)
	}

	t.Log(rest)
}
