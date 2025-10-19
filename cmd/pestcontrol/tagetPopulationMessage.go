package main

import (
	"bufio"
	"encoding/binary"
)

type TargetPopulationMessage struct {
	Length   uint32
	Site     uint32
	Targets  []TargetPopulation
	Checksum byte
}

type TargetPopulation struct {
	Specie string
	Min    uint32
	Max    uint32
}

func (t *TargetPopulationMessage) GetChecksum() byte {
	return t.Checksum
}

func (t *TargetPopulationMessage) GetBytesSum() byte {
	sum := byte(TargetPopulations)

	lenSlice := GetUint32AsBytes(&t.Length)
	for _, b := range lenSlice {
		sum += b
	}

	siteSlice := GetUint32AsBytes(&t.Site)
	for _, b := range siteSlice {
		sum += b
	}

	targetLen := uint32(len(t.Targets))
	targetSlice := GetUint32AsBytes(&targetLen)
	for _, b := range targetSlice {
		sum += b
	}

	for _, target := range t.Targets {
		nameLen := uint32(len(target.Specie))
		nameSlice := GetUint32AsBytes(&nameLen)
		for _, b := range nameSlice {
			sum += b
		}

		for i := range len(target.Specie) {
			sum += target.Specie[i]
		}

		maxV := uint32(target.Max)
		maxSlice := GetUint32AsBytes(&maxV)
		for _, b := range maxSlice {
			sum += b
		}

		minV := uint32(target.Min)
		minSlice := GetUint32AsBytes(&minV)
		for _, b := range minSlice {
			sum += b
		}
	}

	return sum
}

func (t *TargetPopulationMessage) GetCode() byte {
	return byte(TargetPopulations)
}

// We don't send TargetPopulationMessage so we don't need to serialize it
func (t *TargetPopulationMessage) SerializeContent() []byte {
	return nil
}

func ParseTargetPopulations(length int, bytes []byte) (TargetPopulationMessage, error) {
	offset := 0
	blen := len(bytes)

	site := binary.BigEndian.Uint32(bytes[:offset+4])
	offset += 4

	poplen := binary.BigEndian.Uint32(bytes[offset : offset+4])
	offset += 4

	population := make([]TargetPopulation, poplen)

	for i := range poplen {
		namelen := binary.BigEndian.Uint32(bytes[offset : offset+4])
		offset += 4
		name := string(bytes[offset : offset+int(namelen)])
		offset += int(namelen)

		minV := binary.BigEndian.Uint32(bytes[offset : offset+4])
		offset += 4
		maxV := binary.BigEndian.Uint32(bytes[offset : offset+4])
		offset += 4

		pop := TargetPopulation{
			Specie: name,
			Min:    minV,
			Max:    maxV,
		}
		population[i] = pop
	}

	checksum := bytes[blen-1]

	return TargetPopulationMessage{
		Length:   uint32(length),
		Site:     site,
		Targets:  population,
		Checksum: checksum,
	}, nil
}

// TODO: This function could be generic or maybe implemented on the Message interface
func ReadTargetPopulationsMessage(br *bufio.Reader) (TargetPopulationMessage, error) {
	mtype, err := ReadMessageType(br)

	if err != nil {
		return TargetPopulationMessage{}, err
	}

	if mtype != TargetPopulations {
		return TargetPopulationMessage{}, WrongMessageType
	}

	l, err := ReadMessageLength(br)
	if err != nil {
		return TargetPopulationMessage{}, err
	}

	rest, err := ReadRemaining(br, l)
	if err != nil {
		return TargetPopulationMessage{}, err
	}

	siteMsg, err := ParseTargetPopulations(l, rest)
	if err != nil {
		return TargetPopulationMessage{}, err
	}

	if !ValidateChecksum(&siteMsg) {
		return TargetPopulationMessage{}, InvalidChecksumError
	}

	return siteMsg, nil
}
