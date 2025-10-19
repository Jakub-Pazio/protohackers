package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
)

type HelloMessage struct {
	Length   uint32
	Protocol string
	Version  uint32
	Checksum byte
}

func (h *HelloMessage) GetChecksum() byte {
	return h.Checksum
}

func (h *HelloMessage) GetBytesSum() byte {
	sum := byte(Hello)

	lenSlice := GetUint32AsBytes(&h.Length)

	for _, b := range lenSlice {
		sum += b
	}

	pStrLen := uint32(len(h.Protocol))
	pStrLenSlice := GetUint32AsBytes(&pStrLen)

	for _, b := range pStrLenSlice {
		sum += b
	}

	for i := range len(h.Protocol) {
		sum += h.Protocol[i]
	}

	versionSlice := GetUint32AsBytes(&h.Version)
	for _, b := range versionSlice {
		sum += b
	}

	return sum
}

func (h *HelloMessage) SerializeContent() []byte {
	var b bytes.Buffer

	plen := uint32(len(h.Protocol))
	plenbytes := GetUint32AsBytes(&plen)

	b.Write(plenbytes)
	b.Write([]byte(h.Protocol))

	pver := uint32(h.Version)
	pverbytes := GetUint32AsBytes(&pver)

	b.Write(pverbytes)

	return b.Bytes()
}

func (h *HelloMessage) GetCode() byte {
	return byte(Hello)
}

func ParseHello(length int, bytes []byte) (HelloMessage, error) {
	blen := len(bytes)
	protoLen := binary.BigEndian.Uint32(bytes[:4])

	if length-14 != int(protoLen) {
		return HelloMessage{}, fmt.Errorf("protocol length incorrect")
	}

	protocol := string(bytes[4 : protoLen+4])

	vbytes := bytes[protoLen+4 : protoLen+8]
	version := binary.BigEndian.Uint32(vbytes)

	checksum := bytes[blen-1]

	return HelloMessage{
		Length:   uint32(length),
		Protocol: protocol,
		Version:  version,
		Checksum: checksum,
	}, nil
}

var (
	UnknownProtocol    = fmt.Errorf("unknown protocol name")
	UnsupportedVersion = fmt.Errorf("unsupported protocol version")
	ValidHelloMessage  = HelloMessage{Protocol: "pestcontrol", Version: 1}
)

// TODO: this function shold not do any validation, then we could implement it as
// function that is generic for any T Message, and get the correct type of this Message
func ReadHelloMessage(br *bufio.Reader) (HelloMessage, error) {
	mtype, err := ReadMessageType(br)
	if err != nil {
		return HelloMessage{}, err
	}

	if mtype != Hello {
		return HelloMessage{}, WrongMessageType
	}

	l, err := ReadMessageLength(br)
	if err != nil {
		return HelloMessage{}, err
	}

	rest, err := ReadRemaining(br, l)
	if err != nil {
		return HelloMessage{}, err
	}

	msg, err := ParseHello(l, rest)
	if err != nil {
		return HelloMessage{}, err
	}

	if msg.Protocol != "pestcontrol" {
		return HelloMessage{}, UnknownProtocol
	}

	if msg.Version != 1 {
		return HelloMessage{}, UnsupportedVersion
	}

	if !ValidateChecksum(&msg) {
		return HelloMessage{}, InvalidChecksumError
	}

	return msg, nil
}
