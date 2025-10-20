package main

import (
	"bufio"
	"encoding/binary"
	"fmt"
)

type Population struct {
	Name  string
	Count uint32
}

type SiteVisitMessage struct {
	Length      uint32
	Site        uint32
	Populations []Population
	CheckSum    byte
}

func (s *SiteVisitMessage) GetChecksum() byte {
	return s.CheckSum
}

func (s *SiteVisitMessage) GetBytesSum() byte {
	sum := byte(SiteVisit)

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

func (s *SiteVisitMessage) GetCode() byte {
	return byte(SiteVisit)
}

// We don't send SiteVisitMessage so we don't need to serialize it
func (s *SiteVisitMessage) SerializeContent() []byte {
	return nil
}

func VerifyVisitSite(s SiteVisitMessage) error {
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

func ParseSiteVisit(length int, bytes []byte) (SiteVisitMessage, error) {
	offset := 0
	blen := len(bytes)

	if offset+4 > blen {
		return SiteVisitMessage{}, InvalidMessageLengthError
	}
	site := binary.BigEndian.Uint32(bytes[:offset+4])
	offset += 4

	if offset+4 > blen {
		return SiteVisitMessage{}, InvalidMessageLengthError
	}
	populationLen := binary.BigEndian.Uint32(bytes[offset : offset+4])
	offset += 4

	population := make([]Population, populationLen)

	for i := range populationLen {
		if offset+4 > blen {
			return SiteVisitMessage{}, InvalidMessageLengthError
		}
		nameLen := binary.BigEndian.Uint32(bytes[offset : offset+4])
		offset += 4

		if offset+int(nameLen) > blen {
			return SiteVisitMessage{}, InvalidMessageLengthError
		}
		name := string(bytes[offset : offset+int(nameLen)])
		offset += int(nameLen)

		if offset+4 > blen {
			return SiteVisitMessage{}, InvalidMessageLengthError
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

	return SiteVisitMessage{
		Length:      uint32(length),
		Site:        site,
		Populations: population,
		CheckSum:    checksum,
	}, nil
}

func ReadSiteVisitMessage(br *bufio.Reader) (SiteVisitMessage, error) {
	mtype, err := ReadMessageType(br)

	if err != nil {
		return SiteVisitMessage{}, err
	}

	if mtype != SiteVisit {
		return SiteVisitMessage{}, WrongMessageType
	}

	l, err := ReadMessageLength(br)
	if err != nil {
		return SiteVisitMessage{}, err
	}

	rest, err := ReadRemaining(br, l)
	if err != nil {
		return SiteVisitMessage{}, err
	}

	siteMsg, err := ParseSiteVisit(l, rest)
	if err != nil {
		return SiteVisitMessage{}, err
	}

	if !ValidateChecksum(&siteMsg) {
		return SiteVisitMessage{}, InvalidChecksumError
	}

	return siteMsg, nil
}
