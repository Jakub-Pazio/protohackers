package main

import "bytes"

type ErrorMessage struct {
	Length   uint32
	Message  string
	Checksum byte
}

func (e *ErrorMessage) GetChecksum() byte {
	return e.Checksum
}

func (e *ErrorMessage) GetBytesSum() byte {
	sum := byte(Error)

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

func (e *ErrorMessage) SerializeContent() []byte {
	var b bytes.Buffer

	mlen := uint32(len(e.Message))
	mlenbytes := GetUint32AsBytes(&mlen)

	b.Write(mlenbytes)
	b.Write([]byte(e.Message))

	return b.Bytes()
}

func (e *ErrorMessage) GetCode() byte {
	return byte(Error)
}
