package main

import "bytes"

var _ Message = &CreatePolicyMessage{}

type CreatePolicyMessage struct {
	Length   uint32
	Specie   string
	Action   byte
	Checksum byte
}

func (c *CreatePolicyMessage) GetChecksum() byte {
	return c.Checksum
}

func (c *CreatePolicyMessage) GetBytesSum() byte {
	sum := byte(CreatePolicy)

	lenSlice := GetUint32AsBytes(&c.Length)
	for _, b := range lenSlice {
		sum += b
	}

	pStrLen := uint32(len(c.Specie))
	pStrLenSlice := GetUint32AsBytes(&pStrLen)
	for _, b := range pStrLenSlice {
		sum += b
	}

	sum += c.Action

	return sum
}

func (c *CreatePolicyMessage) SerializeContent() []byte {
	var b bytes.Buffer

	slen := uint32(len(c.Specie))
	slenbytes := GetUint32AsBytes(&slen)

	b.Write(slenbytes)
	b.Write([]byte(c.Specie))

	b.Write([]byte{c.Action})

	return b.Bytes()
}

func (c *CreatePolicyMessage) GetCode() byte {
	return byte(CreatePolicy)
}
