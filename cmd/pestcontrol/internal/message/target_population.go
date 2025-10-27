package message

import (
	"bean/cmd/pestcontrol/internal/animal"
	"bufio"
	"encoding/binary"
	"fmt"
)

type TargetPopulation struct {
	Length   uint32
	Site     uint32
	Targets  []animal.TargetPopulation
	Checksum byte
}

func (t *TargetPopulation) GetChecksum() byte {
	return t.Checksum
}

func (t *TargetPopulation) GetBytesSum() byte {
	sum := byte(MessageTypeTargetPopulations)

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

func (t *TargetPopulation) GetCode() byte {
	return byte(MessageTypeTargetPopulations)
}

// We don't send TargetPopulationMessage so we don't need to serialize it
func (t *TargetPopulation) SerializeContent() []byte {
	return nil
}

func ParseTargetPopulations(length int, bytes []byte) (TargetPopulation, error) {
	offset := 0
	blen := len(bytes)

	site := binary.BigEndian.Uint32(bytes[:offset+4])
	offset += 4

	poplen := binary.BigEndian.Uint32(bytes[offset : offset+4])
	offset += 4

	population := make([]animal.TargetPopulation, poplen)

	for i := range poplen {
		namelen := binary.BigEndian.Uint32(bytes[offset : offset+4])
		offset += 4
		name := string(bytes[offset : offset+int(namelen)])
		offset += int(namelen)

		minV := binary.BigEndian.Uint32(bytes[offset : offset+4])
		offset += 4
		maxV := binary.BigEndian.Uint32(bytes[offset : offset+4])
		offset += 4

		pop := animal.TargetPopulation{
			Specie: name,
			Min:    minV,
			Max:    maxV,
		}
		population[i] = pop
	}

	checksum := bytes[blen-1]

	return TargetPopulation{
		Length:   uint32(length),
		Site:     site,
		Targets:  population,
		Checksum: checksum,
	}, nil
}

// TODO: This function could be generic or maybe implemented on the Message interface
func ReadTargetPopulations(br *bufio.Reader) (TargetPopulation, error) {
	mtype, err := ReadMessageType(br)

	if err != nil {
		return TargetPopulation{}, fmt.Errorf("read message type: %w", err)
	}

	if mtype != MessageTypeTargetPopulations {
		return TargetPopulation{}, ErrWrongMessage
	}

	l, err := ReadMessageLength(br)
	if err != nil {
		return TargetPopulation{}, fmt.Errorf("read message length: %w", err)
	}

	rest, err := ReadRemaining(br, l)
	if err != nil {
		return TargetPopulation{}, fmt.Errorf("read remaining message: %w", err)
	}

	siteMsg, err := ParseTargetPopulations(l, rest)
	if err != nil {
		return TargetPopulation{}, fmt.Errorf("parse target population: %w", err)
	}

	if !ValidateChecksum(&siteMsg) {
		return TargetPopulation{}, ErrInvalidChecksum
	}

	return siteMsg, nil
}
