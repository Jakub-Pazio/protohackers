package main

type DeletePolicyMessage struct {
	Length   uint32
	PolicyID uint32
	Checksum byte
}

func (d *DeletePolicyMessage) GetChecksum() byte {
	return d.Checksum
}

func (d *DeletePolicyMessage) GetBytesSum() byte {
	sum := byte(DeletePolicy)

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

func (c *DeletePolicyMessage) SerializeContent() []byte {
	return GetUint32AsBytes(&c.PolicyID)
}

func (c *DeletePolicyMessage) GetCode() byte {
	return byte(DeletePolicy)
}
