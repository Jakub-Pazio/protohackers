package main

import "testing"

func TestParseMessage(t *testing.T) {
	validDataMessage := "/data/1234/0/Hello world!/"

	tp, s, rest, err := ParseMessage(validDataMessage)

	if err != nil {
		t.Errorf("Unexpected error: %v\n", err)
	}

	wantRest := `0/Hello world!`
	if rest != wantRest {
		t.Errorf("got %q, want %q\n", rest, wantRest)
	}

	if tp != Type("data") {
		t.Errorf("Wrong type: %q\n", err)
	}

	if s != Session(1234) {
		t.Errorf("Wrong session: %d\n", s)
	}

	invalidMessage := "/data/0x12/1/o7/"
	_, _, _, err = ParseMessage(invalidMessage)

	if err == nil {
		t.Errorf("expected error")
	}
}

func TestUnescape(t *testing.T) {
	msg := "\\/"

	if len(msg) != 2 {
		t.Errorf("I don't understand strings, len: %d\n", len(msg))
	}

	unescaped, err := unescapeMsg(msg)

	if err != nil {
		t.Errorf("unexpected error: %v\n", err)
	}

	want := "/"
	if unescaped != want {
		t.Errorf("unescaped: %v\n", unescaped)
	}
}
