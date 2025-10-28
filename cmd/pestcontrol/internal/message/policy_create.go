package message

import "bytes"

var _ Message = &CreatePolicy{}

type CreatePolicy struct {
	Length   uint32
	Specie   string
	Action   byte
	Checksum byte
}

func (c *CreatePolicy) GetChecksum() byte {
	return c.Checksum
}

func (c *CreatePolicy) GetBytesSum() byte {
	sum := byte(TypeCreatePolicy)

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

func (c *CreatePolicy) SerializeContent() []byte {
	var b bytes.Buffer

	slen := uint32(len(c.Specie))
	slenbytes := GetUint32AsBytes(&slen)

	b.Write(slenbytes)
	b.Write([]byte(c.Specie))

	b.Write([]byte{c.Action})

	return b.Bytes()
}

func (c *CreatePolicy) GetCode() byte {
	return byte(TypeCreatePolicy)
}
