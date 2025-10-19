package main

type DialAuthorityMessage struct {
	Length   uint32
	Site     uint32
	Checksum byte
}

func (d *DialAuthorityMessage) GetChecksum() byte {
	return d.Checksum
}

func (d *DialAuthorityMessage) GetBytesSum() byte {
	sum := byte(DialAuthority)

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

func (d *DialAuthorityMessage) SerializeContent() []byte {
	site := uint32(d.Site)
	return GetUint32AsBytes(&site)
}

func (d *DialAuthorityMessage) GetCode() byte {
	return byte(DialAuthority)
}
