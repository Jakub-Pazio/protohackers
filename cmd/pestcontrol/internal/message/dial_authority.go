package message

type DialAuthority struct {
	Length   uint32
	Site     uint32
	Checksum byte
}

func (d *DialAuthority) GetChecksum() byte {
	return d.Checksum
}

func (d *DialAuthority) GetBytesSum() byte {
	sum := byte(MessageTypeDialAuthority)

	lenSlice := GetUint32AsBytes(&d.Length)
	for _, b := range lenSlice {
		sum += b
	}

	siteSlice := GetUint32AsBytes(&d.Site)
	for _, b := range siteSlice {
		sum += b
	}

	return sum
}

func (d *DialAuthority) SerializeContent() []byte {
	site := uint32(d.Site)
	return GetUint32AsBytes(&site)
}

func (d *DialAuthority) GetCode() byte {
	return byte(MessageTypeDialAuthority)
}
