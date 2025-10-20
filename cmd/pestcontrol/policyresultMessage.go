package main

import (
	"bufio"
	"encoding/binary"
)

var _ Message = &PolicyResultMessage{}

type PolicyResultMessage struct {
	Length   uint32
	PolicyID uint32
	Checksum byte
}

func (p *PolicyResultMessage) GetChecksum() byte {
	return p.Checksum
}

func (p *PolicyResultMessage) GetBytesSum() byte {
	sum := byte(PolicyResult)

	lenSlice := GetUint32AsBytes(&p.Length)
	for _, b := range lenSlice {
		sum += b
	}

	policySlice := GetUint32AsBytes(&p.PolicyID)
	for _, b := range policySlice {
		sum += b
	}

	return sum
}

// We don't send this message
func (p *PolicyResultMessage) SerializeContent() []byte {
	return nil
}

func (p *PolicyResultMessage) GetCode() byte {
	return byte(PolicyResult)
}

// TODO: Maybe those parse functions could be generated from the struct itself
// Later I could write preprocessor that would generate those go function from struct definition
// and struct tags, so in case of other messages those methods could be generated automatically
func ParsePolicyResult(lenght int, bytes []byte) (PolicyResultMessage, error) {
	blen := len(bytes)

	policy := binary.BigEndian.Uint32(bytes[:4])

	checksum := bytes[blen-1]

	return PolicyResultMessage{
		Length:   uint32(lenght),
		PolicyID: policy,
		Checksum: checksum,
	}, nil
}

func ReadPolicyResultMessage(br *bufio.Reader) (PolicyResultMessage, error) {
	mtype, err := ReadMessageType(br)

	if err != nil {
		return PolicyResultMessage{}, err
	}

	if mtype != PolicyResult {
		return PolicyResultMessage{}, WrongMessageType
	}

	l, err := ReadMessageLength(br)
	if err != nil {
		return PolicyResultMessage{}, err
	}

	rest, err := ReadRemaining(br, l)
	if err != nil {
		return PolicyResultMessage{}, err
	}

	policyResult, err := ParsePolicyResult(l, rest)
	if err != nil {
		return PolicyResultMessage{}, err
	}

	if !ValidateChecksum(&policyResult) {
		return PolicyResultMessage{}, InvalidChecksumError
	}

	return policyResult, nil
}
