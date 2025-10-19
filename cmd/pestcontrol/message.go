package main

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"net"
)

type Type byte

const (
	Hello Type = iota + 0x50
	Error
	OK
	DialAuthority
	TargetPopulations
	CreatePolicy
	DeletePolicy
	PolicyResult
	SiteVisit

	None Type = 0
)

var (
	InvalidMessageTypeError = fmt.Errorf("invalid message type")
	MessageToLargeError     = fmt.Errorf("message to large")
	InvalidChecksumError    = fmt.Errorf("checksum is invalid")
)

func validMessageType(b byte) bool {
	if b >= 0x50 && b <= 0x58 {
		return true
	}
	return false
}

const (
	MsgHeaderLen = 5
)

type Message interface {
	// GetChecksum returns value of checksum in the struct
	// Should be called only on messages that came to server
	GetChecksum() byte

	// GetBytesSum returns sum of bytes of serialized message without checksum
	GetBytesSum() byte
	// SerializeContent serializes "body" bytes without type, size and checksum
	SerializeContent() []byte

	GetCode() byte
}

func ValidateChecksum(m Message) bool {
	return m.GetBytesSum()+m.GetChecksum() == 0
}

func SerializeMessage(m Message) []byte {
	bodyBytes := m.SerializeContent()

	bodyLen := len(bodyBytes)
	totalLen := uint32(bodyLen + 6) // Type (1byte) + TotalLen (4bytes) + Checksum (1byte)

	buf := make([]byte, totalLen)

	code := m.GetCode()
	buf[0] = code

	totalLenSlice := GetUint32AsBytes(&totalLen)
	copy(buf[1:], totalLenSlice)

	copy(buf[5:], bodyBytes)

	var checkSum byte

	for i := range totalLen - 1 {
		checkSum -= buf[i]
	}

	buf[len(buf)-1] = checkSum

	return buf
}

func GetUint32AsBytes(u *uint32) []byte {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, *u)
	return b
}

func WriteMessage(conn net.Conn, msg Message) error {
	_, err := conn.Write(SerializeMessage(msg))
	return err
}

func ReadRemaining(br *bufio.Reader, l int) ([]byte, error) {
	remaining := l - MsgHeaderLen

	buf := make([]byte, remaining)

	_, err := io.ReadFull(br, buf)

	if err != nil {
		return buf, fmt.Errorf("could not read whole message: %q", err)
	}

	return buf, err
}

func ReadMessageType(br *bufio.Reader) (Type, error) {
	b, err := br.ReadByte()
	if err != nil {
		return None, fmt.Errorf("could not read type: %v", err)
	}

	if !validMessageType(b) {
		return None, InvalidMessageTypeError
	}

	return Type(b), nil
}

func ReadMessageLength(br *bufio.Reader) (int, error) {
	buf := make([]byte, 4)
	_, err := io.ReadFull(br, buf)

	if err != nil {
		return 0, fmt.Errorf("could not read message length: %v", err)
	}

	length := binary.BigEndian.Uint32(buf)

	if length > 1_000_000 {
		return 0, MessageToLargeError
	}

	return int(length), nil
}
