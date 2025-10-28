package message

import (
	"bufio"
	"encoding/binary"
	"fmt"
)

type Population struct {
	Name  string
	Count uint32
}

type SiteVisit struct {
	Length      uint32
	Site        uint32
	Populations []Population
	CheckSum    byte
}

func (s *SiteVisit) GetChecksum() byte {
	return s.CheckSum
}

func (s *SiteVisit) GetBytesSum() byte {
	sum := byte(TypeSiteVisit)

	lenSlice := GetUint32AsBytes(&s.Length)
	for _, b := range lenSlice {
		sum += b
	}

	siteSlice := GetUint32AsBytes(&s.Site)
	for _, b := range siteSlice {
		sum += b
	}

	popLen := uint32(len(s.Populations))
	popSlice := GetUint32AsBytes(&popLen)
	for _, b := range popSlice {
		sum += b
	}

	for _, population := range s.Populations {
		nameLen := uint32(len(population.Name))
		nameSlice := GetUint32AsBytes(&nameLen)
		for _, b := range nameSlice {
			sum += b
		}

		for i := range len(population.Name) {
			sum += population.Name[i]
		}

		popCount := uint32(population.Count)
		popCountSlice := GetUint32AsBytes(&popCount)
		for _, b := range popCountSlice {
			sum += b
		}
	}

	return sum
}

func (s *SiteVisit) GetCode() byte {
	return byte(TypeSiteVisit)
}

// We don't send SiteVisitMessage so we don't need to serialize it
func (s *SiteVisit) SerializeContent() []byte {
	return nil
}

func VerifySiteVisit(s *SiteVisit) error {
	popMap := make(map[string]uint32)

	for _, p := range s.Populations {
		count, ok := popMap[p.Name]
		if !ok {
			popMap[p.Name] = p.Count
		} else {
			if count != p.Count {
				return fmt.Errorf("conflicting count for %s", p.Name)
			}
		}
	}

	return nil
}

func ParseSiteVisit(length int, bytes []byte) (*SiteVisit, error) {
	offset := 0
	blen := len(bytes)

	if offset+4 > blen {
		return nil, ErrInvalidMessageLength
	}
	site := binary.BigEndian.Uint32(bytes[:offset+4])
	offset += 4

	if offset+4 > blen {
		return nil, ErrInvalidMessageLength
	}
	populationLen := binary.BigEndian.Uint32(bytes[offset : offset+4])
	offset += 4

	population := make([]Population, populationLen)

	for i := range populationLen {
		if offset+4 > blen {
			return nil, ErrInvalidMessageLength
		}
		nameLen := binary.BigEndian.Uint32(bytes[offset : offset+4])
		offset += 4

		if offset+int(nameLen) > blen {
			return nil, ErrInvalidMessageLength
		}
		name := string(bytes[offset : offset+int(nameLen)])
		offset += int(nameLen)

		if offset+4 > blen {
			return nil, ErrInvalidMessageLength
		}
		count := binary.BigEndian.Uint32(bytes[offset : offset+4])
		offset += 4

		pop := Population{
			Name:  name,
			Count: count,
		}

		population[i] = pop
	}

	checksum := bytes[blen-1]

	return &SiteVisit{
		Length:      uint32(length),
		Site:        site,
		Populations: population,
		CheckSum:    checksum,
	}, nil
}

func ReadSiteVisit(br *bufio.Reader) (SiteVisit, error) {
	mtype, err := ReadMessageType(br)

	if err != nil {
		return SiteVisit{}, fmt.Errorf("read message type: %w", err)
	}

	if mtype != TypeSiteVisit {
		return SiteVisit{}, ErrWrongMessage
	}

	l, err := ReadMessageLength(br)
	if err != nil {
		return SiteVisit{}, fmt.Errorf("read message length: %w", err)
	}

	rest, err := ReadBody(br, l)
	if err != nil {
		return SiteVisit{}, fmt.Errorf("read remainig: %w", err)
	}

	siteMsg, err := ParseSiteVisit(l, rest)
	if err != nil {
		return SiteVisit{}, fmt.Errorf("parse site visit: %w", err)
	}

	if !ValidateChecksum(siteMsg) {
		return SiteVisit{}, ErrInvalidChecksum
	}

	return *siteMsg, nil
}
