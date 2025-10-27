package message

import "testing"

func TestValidateHello(t *testing.T) {
	msg := HelloMessage{
		Length:   0x19,
		Protocol: "pestcontrol",
		Version:  1,
		Checksum: 0xCE,
	}

	if !ValidateChecksum(&msg) {
		t.Error("message should be valid")
	}
}

func TestSerializeMessage(t *testing.T) {
	msg := &HelloMessage{Protocol: "pestcontrol", Version: 1}

	serMsg := SerializeMessage(msg)

	if serMsg[len(serMsg)-1] != 0xCE {
		t.Errorf("checksum calculated incorectly")
	}
}
