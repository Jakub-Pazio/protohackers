package message

import (
	"bufio"
	"fmt"
)

var _ Message = (*OK)(nil)

type OK struct {
	Length   uint32
	Checksum byte
}

func (o *OK) GetChecksum() byte {
	return o.Checksum
}

func (o *OK) GetBytesSum() byte {
	sum := byte(TypeOK)

	lenSlice := GetUint32AsBytes(&o.Length)
	for _, b := range lenSlice {
		sum += b
	}

	return sum
}

func (o *OK) GetCode() byte {
	return byte(TypeOK)
}

func (o *OK) SerializeContent() []byte {
	return nil
}

func ParseOk(lenght int, bytes []byte) (*OK, error) {
	blen := len(bytes)

	checksum := bytes[blen-1]

	return &OK{
		Length:   uint32(lenght),
		Checksum: checksum,
	}, nil
}

func ReadOK(br *bufio.Reader) (OK, error) {
	mtype, err := ReadMessageType(br)

	if err != nil {
		return OK{}, fmt.Errorf("read message type: %w", err)
	}

	if mtype != TypeOK {
		return OK{}, ErrWrongMessage
	}

	l, err := ReadMessageLength(br)
	if err != nil {
		return OK{}, fmt.Errorf("read message length: %w", err)
	}

	rest, err := ReadBody(br, l)
	if err != nil {
		return OK{}, fmt.Errorf("read remaining: %w", err)
	}

	okMsg, err := ParseOk(l, rest)
	if err != nil {
		return OK{}, fmt.Errorf("parse ok: %w", err)
	}

	if !ValidateChecksum(okMsg) {
		return OK{}, ErrInvalidChecksum
	}

	return *okMsg, nil
}
