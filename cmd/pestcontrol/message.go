package main

import (
	"encoding/binary"
	"fmt"
	"unsafe"
)

type Type byte

const (
	Hello Type = iota + 0x50
	Error
	OK
	DialAuthority
	TargetPopulations
	CreatePolicy
	DeletePolicy
	PolicyResult
	SiteVisit

	None Type = 0
)

var invalidMessageTypeError = fmt.Errorf("invalid message type")
var messageToLargeError = fmt.Errorf("message to large")

func validMessageType(b byte) bool {
	if b >= 0x50 && b <= 0x58 {
		return true
	}
	return false
}

const (
	MsgHeaderLen = 5
)

type Message interface {
	SetChecksum(val byte)
	GetChecksum() byte
	GetBytesSum() byte
}

func SetCorrectChecksum(m Message) {
	remaining := byte(0 - m.GetBytesSum())

	m.SetChecksum(remaining)
}

func ValidateChecksum(m Message) bool {
	return m.GetBytesSum()+m.GetChecksum() == 0
}

type HelloMessage struct {
	Length   uint32
	Protocol string
	Version  uint32
	Checksum byte
}

func (h *HelloMessage) SetChecksum(val byte) {
	h.Checksum = val
}

func (h *HelloMessage) GetChecksum() byte {
	return h.Checksum
}

func (h *HelloMessage) GetBytesSum() byte {
	sum := byte(Hello)

	lenSlice := getUint32AsBytes(&h.Length)

	for _, b := range lenSlice {
		sum += b
	}

	pStrLen := uint32(len(h.Protocol))
	pStrLenSlice := getUint32AsBytes(&pStrLen)

	for _, b := range pStrLenSlice {
		sum += b
	}

	for i := range len(h.Protocol) {
		sum += h.Protocol[i]
	}

	versionSlice := getUint32AsBytes(&h.Version)
	for _, b := range versionSlice {
		sum += b
	}

	return sum
}

func ParseHello(length int, bytes []byte) (HelloMessage, error) {
	blen := len(bytes)
	protoLen := binary.BigEndian.Uint32(bytes[:4])

	if length-14 != int(protoLen) {
		return HelloMessage{}, fmt.Errorf("protocol length incorrect")
	}

	protocol := string(bytes[4 : protoLen+5])

	vbytes := bytes[protoLen+5 : protoLen+9]
	version := binary.BigEndian.Uint32(vbytes)

	checksum := bytes[blen-1]

	return HelloMessage{
		Length:   uint32(length),
		Protocol: protocol,
		Version:  version,
		Checksum: checksum,
	}, nil
}

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

func (s *SiteVisitMessage) SetChecksum(val byte) {
	s.CheckSum = val
}

func (s *SiteVisitMessage) GetChecksum() byte {
	return s.CheckSum
}

func (s *SiteVisitMessage) GetBytesSum() byte {
	sum := byte(SiteVisit)

	lenSlice := getUint32AsBytes(&s.Length)

	for _, b := range lenSlice {
		sum += b
	}

	siteSlice := getUint32AsBytes(&s.Site)

	for _, b := range siteSlice {
		sum += b
	}

	popLen := uint32(len(s.Populations))
	popSlice := getUint32AsBytes(&popLen)

	for _, b := range popSlice {
		sum += b
	}

	for _, population := range s.Populations {
		nameLen := uint32(len(population.Name))
		nameSlice := getUint32AsBytes(&nameLen)
		for _, b := range nameSlice {
			sum += b
		}

		for i := range len(population.Name) {
			sum += population.Name[i]
		}

		popCount := uint32(population.Count)
		popCountSlice := getUint32AsBytes(&popCount)
		for _, b := range popCountSlice {
			sum += b
		}
	}

	return sum
}

func ParseSiteVisit(length int, bytes []byte) (SiteVisitMessage, error) {
	offset := 0
	blen := len(bytes)

	site := binary.BigEndian.Uint32(bytes[:offset+4])
	offset += 4

	populationLen := binary.BigEndian.Uint32(bytes[offset : offset+4])
	offset += 4

	population := make([]Population, populationLen)

	for i := range populationLen {
		nameLen := binary.BigEndian.Uint32(bytes[offset : offset+4])
		offset += 4
		name := string(bytes[offset : offset+int(nameLen)])
		offset += int(nameLen)

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

func getUint32AsBytes(p *uint32) []byte {
	return unsafe.Slice((*byte)(unsafe.Pointer(p)), 4)
}
