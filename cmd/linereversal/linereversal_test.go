package main

import "testing"

func TestParseMessage(t *testing.T) {
	validDataMessage := "/data/1234/0/Hello world!/"

	tp, s, rest, err := ParseMessage(validDataMessage)

	if err != nil {
		t.Errorf("Unexpected error: %v\n", err)
	}

	t.Errorf("%s\n", rest)
	t.Errorf("%v\n", s)

	if tp != Type("data") {
		t.Errorf("Wrong type: %q\n", err)
	}

	if s != Session(1234) {
		t.Errorf("Wrong session: %d\n", s)
	}

	invalidMessage := "/data/0x12/1/o7/"
	_, _, _, err = ParseMessage(invalidMessage)

	t.Errorf("error %v\n", err)
}
