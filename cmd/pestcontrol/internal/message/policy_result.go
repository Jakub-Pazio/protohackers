package message

import (
	"bufio"
	"encoding/binary"
	"fmt"
)

var _ Message = &PolicyResult{}

type PolicyResult struct {
	Length   uint32
	PolicyID uint32
	Checksum byte
}

func (p *PolicyResult) GetChecksum() byte {
	return p.Checksum
}

func (p *PolicyResult) GetBytesSum() byte {
	sum := byte(TypePolicyResult)

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
func (p *PolicyResult) SerializeContent() []byte {
	return nil
}

func (p *PolicyResult) GetCode() byte {
	return byte(TypePolicyResult)
}

// TODO: Maybe those parse functions could be generated from the struct itself
// Later I could write preprocessor that would generate those go function from struct definition
// and struct tags, so in case of other messages those methods could be generated automatically
func ParsePolicyResult(lenght int, bytes []byte) (*PolicyResult, error) {
	blen := len(bytes)

	policy := binary.BigEndian.Uint32(bytes[:4])

	checksum := bytes[blen-1]

	return &PolicyResult{
		Length:   uint32(lenght),
		PolicyID: policy,
		Checksum: checksum,
	}, nil
}

func ReadPolicyResult(br *bufio.Reader) (PolicyResult, error) {
	mtype, err := ReadMessageType(br)

	if err != nil {
		return PolicyResult{}, fmt.Errorf("read message type: %w", err)
	}

	if mtype != TypePolicyResult {
		return PolicyResult{}, ErrWrongMessage
	}

	l, err := ReadMessageLength(br)
	if err != nil {
		return PolicyResult{}, fmt.Errorf("read message length: %w", err)
	}

	rest, err := ReadBody(br, l)
	if err != nil {
		return PolicyResult{}, fmt.Errorf("read remaining message: %w", err)
	}

	policyResult, err := ParsePolicyResult(l, rest)
	if err != nil {
		return PolicyResult{}, fmt.Errorf("parse policy result: %w", err)
	}

	if !ValidateChecksum(policyResult) {
		return PolicyResult{}, ErrInvalidChecksum
	}

	return *policyResult, nil
}
