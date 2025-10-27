package message

import "bytes"

type Error struct {
	Length   uint32
	Message  string
	Checksum byte
}

func (e *Error) GetChecksum() byte {
	return e.Checksum
}

func (e *Error) GetBytesSum() byte {
	sum := byte(MessageTypeError)

	lenSlice := GetUint32AsBytes(&e.Length)
	for _, b := range lenSlice {
		sum += b
	}

	eStrLen := uint32(len(e.Message))
	eStrLenSlice := GetUint32AsBytes(&eStrLen)
	for _, b := range eStrLenSlice {
		sum += b
	}

	for i := range e.Message {
		sum += e.Message[i]
	}

	return sum
}

func (e *Error) SerializeContent() []byte {
	var b bytes.Buffer

	mlen := uint32(len(e.Message))
	mlenbytes := GetUint32AsBytes(&mlen)

	b.Write(mlenbytes)
	b.Write([]byte(e.Message))

	return b.Bytes()
}

func (e *Error) GetCode() byte {
	return byte(MessageTypeError)
}
