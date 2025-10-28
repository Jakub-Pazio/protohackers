package message

type DeletePolicy struct {
	Length   uint32
	PolicyID uint32
	Checksum byte
}

func (d *DeletePolicy) GetChecksum() byte {
	return d.Checksum
}

func (d *DeletePolicy) GetBytesSum() byte {
	sum := byte(TypeDeletePolicy)

	lenSlice := GetUint32AsBytes(&d.Length)
	for _, b := range lenSlice {
		sum += b
	}

	policySlice := GetUint32AsBytes(&d.PolicyID)
	for _, b := range policySlice {
		sum += b
	}

	return sum
}

func (c *DeletePolicy) SerializeContent() []byte {
	return GetUint32AsBytes(&c.PolicyID)
}

func (c *DeletePolicy) GetCode() byte {
	return byte(TypeDeletePolicy)
}
