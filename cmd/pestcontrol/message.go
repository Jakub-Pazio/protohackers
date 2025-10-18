package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
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
	// GetChecksum returns value of checksum in the struct
	// Should be called only on messages that came to server
	GetChecksum() byte

	// GetBytesSum returns sum of bytes of serialized message without checksum
	GetBytesSum() byte
	// SerializeContent serializes "body" bytes without type, size and checksum
	SerializeContent() []byte

	GetCode() byte
}

func ValidateChecksum(m Message) bool {
	return m.GetBytesSum()+m.GetChecksum() == 0
}

func SerializeMessage(m Message) []byte {
	bodyBytes := m.SerializeContent()

	bodyLen := len(bodyBytes)
	totalLen := uint32(bodyLen + 6) // Type (1byte) + TotalLen (4bytes) + Checksum (1byte)

	buf := make([]byte, totalLen)

	code := m.GetCode()
	buf[0] = code

	totalLenSlice := getUint32AsBytes(&totalLen)
	copy(buf[1:], totalLenSlice)

	copy(buf[5:], bodyBytes)

	var checkSum byte

	for i := range totalLen - 1 {
		checkSum -= buf[i]
	}

	buf[len(buf)-1] = checkSum

	return buf
}

type HelloMessage struct {
	Length   uint32
	Protocol string
	Version  uint32
	Checksum byte
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

func (h *HelloMessage) SerializeContent() []byte {
	var b bytes.Buffer

	plen := uint32(len(h.Protocol))
	plenbytes := getUint32AsBytes(&plen)

	b.Write(plenbytes)
	b.Write([]byte(h.Protocol))

	pver := uint32(h.Version)
	pverbytes := getUint32AsBytes(&pver)

	b.Write(pverbytes)

	return b.Bytes()
}

func (h *HelloMessage) GetCode() byte {
	return byte(Hello)
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

	lenSlice := getUint32AsBytes(&e.Length)
	for _, b := range lenSlice {
		sum += b
	}

	eStrLen := uint32(len(e.Message))
	eStrLenSlice := getUint32AsBytes(&eStrLen)
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
	mlenbytes := getUint32AsBytes(&mlen)

	b.Write(mlenbytes)
	b.Write([]byte(e.Message))

	return b.Bytes()
}

func (e *ErrorMessage) GetCode() byte {
	return byte(Error)
}

func getUint32AsBytes(u *uint32) []byte {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, *u)
	return b
}
