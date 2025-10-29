package message

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

type Type byte

const (
	TypeHello Type = iota + 0x50
	TypeError
	TypeOK
	TypeDialAuthority
	TypeTargetPopulations
	TypeCreatePolicy
	TypeDeletePolicy
	TypePolicyResult
	TypeSiteVisit

	TypeNone Type = 0
)

const MaxLen = 1_000_000

var (
	ErrInvalidMessageType   = errors.New("invalid message type")
	ErrMessageToLarge       = errors.New("message to large")
	ErrInvalidChecksum      = errors.New("checksum is invalid")
	ErrWrongMessage         = errors.New("unexpected message type")
	ErrInvalidMessageLength = errors.New("invalid message lenght")
	ErrReadUnsupported      = errors.New("reading not supported for this type")
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

type Raw struct {
	MsgType Type
	Lenght  int
	Body    []byte
}

func ValidateChecksum(m Message) bool {
	return m.GetBytesSum()+m.GetChecksum() == 0
}

func Serialize(m Message) []byte {
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

func ReadBody(br *bufio.Reader, l int) ([]byte, error) {
	remaining := l - MsgHeaderLen

	buf := make([]byte, remaining)

	_, err := io.ReadFull(br, buf)

	if err != nil {
		return buf, fmt.Errorf("read full: %w", err)
	}

	return buf, err
}

func ReadMessageType(br *bufio.Reader) (Type, error) {
	b, err := br.ReadByte()
	if err != nil {
		return TypeNone, fmt.Errorf("read byte: %w", err)
	}

	if !validMessageType(b) {
		return TypeNone, ErrInvalidMessageType
	}

	return Type(b), nil
}

func ReadMessageLength(br *bufio.Reader) (int, error) {
	buf := make([]byte, 4)
	_, err := io.ReadFull(br, buf)

	if err != nil {
		return 0, fmt.Errorf("read full: %w", err)
	}

	length := binary.BigEndian.Uint32(buf)

	if length > MaxLen {
		return 0, ErrMessageToLarge
	}

	return int(length), nil
}
