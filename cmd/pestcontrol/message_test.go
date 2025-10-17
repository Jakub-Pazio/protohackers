package main

import "testing"

func TestValidateHello(t *testing.T) {
	msg := HelloMessage{
		Length:   0x19,
		Protocol: "pestcontrol",
		Version:  1,
		Checksum: 0xCE,
	}

	if !msg.validChecksum() {
		t.Error("message should be valid")
	}
}
