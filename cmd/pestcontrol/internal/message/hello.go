package message

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
)

var _ Message = (*Hello)(nil)

type Hello struct {
	Length   uint32
	Protocol string
	Version  uint32
	Checksum byte
}

func (h *Hello) GetChecksum() byte {
	return h.Checksum
}

func (h *Hello) GetBytesSum() byte {
	sum := byte(MessageTypeHello)

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

func (h *Hello) SerializeContent() []byte {
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

func (h *Hello) GetCode() byte {
	return byte(MessageTypeHello)
}

func ParseHello(length int, bytes []byte) (Hello, error) {
	blen := len(bytes)
	protoLen := binary.BigEndian.Uint32(bytes[:4])

	if length-14 != int(protoLen) {
		return Hello{}, fmt.Errorf("protocol length incorrect")
	}

	protocol := string(bytes[4 : protoLen+4])

	vbytes := bytes[protoLen+4 : protoLen+8]
	version := binary.BigEndian.Uint32(vbytes)

	checksum := bytes[blen-1]

	return Hello{
		Length:   uint32(length),
		Protocol: protocol,
		Version:  version,
		Checksum: checksum,
	}, nil
}

var (
	ErrUnknownProtocol    = fmt.Errorf("unknown protocol name")
	ErrUnsupportedVersion = fmt.Errorf("unsupported protocol version")
)

var ValidHello = Hello{Protocol: "pestcontrol", Version: 1}

// TODO: this function shold not do any validation, then we could implement it as
// function that is generic for any T Message, and get the correct type of this Message
func ReadHello(br *bufio.Reader) (Hello, error) {
	mtype, err := ReadMessageType(br)
	if err != nil {
		return Hello{}, err
	}

	if mtype != MessageTypeHello {
		return Hello{}, ErrWrongMessage
	}

	l, err := ReadMessageLength(br)
	if err != nil {
		return Hello{}, err
	}

	rest, err := ReadRemaining(br, l)
	if err != nil {
		return Hello{}, err
	}

	msg, err := ParseHello(l, rest)
	if err != nil {
		return Hello{}, err
	}

	if msg.Protocol != "pestcontrol" {
		return Hello{}, ErrUnknownProtocol
	}

	if msg.Version != 1 {
		return Hello{}, ErrUnsupportedVersion
	}

	if !ValidateChecksum(&msg) {
		return Hello{}, ErrInvalidChecksum
	}

	return msg, nil
}
