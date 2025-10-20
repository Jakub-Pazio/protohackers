package main

import "bufio"

// var _ Message = &Message{}

type OkMessage struct {
	Length   uint32
	Checksum byte
}

func (o *OkMessage) GetChecksum() byte {
	return o.Checksum
}

func (o *OkMessage) GetBytesSum() byte {
	sum := byte(OK)

	lenSlice := GetUint32AsBytes(&o.Length)
	for _, b := range lenSlice {
		sum += b
	}

	return sum
}

func (o *OkMessage) GetCode() byte {
	return byte(OK)
}

func (o *OkMessage) SerializeContent() []byte {
	return nil
}

func ParseOk(lenght int, bytes []byte) (OkMessage, error) {
	blen := len(bytes)

	checksum := bytes[blen-1]

	return OkMessage{
		Length:   uint32(lenght),
		Checksum: checksum,
	}, nil
}

func ReadOkMessage(br *bufio.Reader) (OkMessage, error) {
	mtype, err := ReadMessageType(br)

	if err != nil {
		return OkMessage{}, nil
	}

	if mtype != SiteVisit {
		return OkMessage{}, WrongMessageType
	}

	l, err := ReadMessageLength(br)
	if err != nil {
		return OkMessage{}, err
	}

	rest, err := ReadRemaining(br, l)
	if err != nil {
		return OkMessage{}, err
	}

	okMsg, err := ParseOk(l, rest)
	if err != nil {
		return OkMessage{}, err
	}

	if !ValidateChecksum(&okMsg) {
		return OkMessage{}, InvalidChecksumError
	}

	return okMsg, nil
}
