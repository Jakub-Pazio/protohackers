package main

import (
	"bufio"
	"io"
	"log"
	"net"
	"slices"
	"testing"
)

func TestParseEmptyCipher(t *testing.T) {
	in, out := net.Pipe()

	go in.Write([]byte{0})
	c, err := parseCipher(out)
	if err != nil {
		t.Error("should parse empty cipher\n")
	}

	if len(c) != 0 {
		t.Errorf("cipher should be empty but is %d long\n", len(c))
	}
}

func TestParseGoodCipher(t *testing.T) {
	in, out := net.Pipe()

	go in.Write([]byte{2, 1, 1, 0})
	c, err := parseCipher(out)
	if err != nil {
		t.Error("should parse incorrect cipher\n")
	}

	if len(c) != 2 {
		t.Errorf("cipher should be 2 ops long but is %d long\n", len(c))
	}

	if c[0].Opid != 2 {
		t.Errorf("opid should be 2 but is %d\n", c[0].Opid)
	}

	if c[0].Value != 1 {
		t.Errorf("operation value should be 1 but is %d\n", c[0].Value)
	}

	if c[1].Opid != 1 {
		t.Errorf("opid should be 1 but is %d\n", c[1].Opid)
	}
}

func TestParseBadCipher(t *testing.T) {
	in, out := net.Pipe()

	go in.Write([]byte{6, 0})
	_, err := parseCipher(out)
	if err == nil {
		t.Error("Should error with unknowwn operation id")
	}
}

func TestReduceReverseCipher(t *testing.T) {
	ci := []CipherOp{{Opid: XorPos}, {Opid: Reverse}, {Opid: Reverse}, {Opid: Add, Value: 0}, {Opid: Xor, Value: 0}, {Opid: End}}

	ci, change := reduceCipher(ci)
	for change {
		ci, change = reduceCipher(ci)
	}

	if !slices.Equal(ci, []CipherOp{{Opid: XorPos}, {Opid: End}}) {
		t.Errorf("The slice should have just end but has: %+v\n", ci)
	}

	c2 := []CipherOp{{Opid: Xor, Value: 0xa0}, {Opid: Xor, Value: 0x0b}, {Opid: Xor, Value: 0xab}, {Opid: End}}
	ci, change = reduceCipher(c2)

	if !slices.Equal(ci, []CipherOp{{Opid: End}}) {
		t.Errorf("The slice should have just end but has: %+v\n", ci)
	}

}

func TestDecodePlain(t *testing.T) {
	cipher := []CipherOp{{Opid: End}}
	r, w := io.Pipe()
	n := 0
	go func() {
		w.Write([]byte("4x dog, 5x car\n"))
	}()
	res, _ := decodeLine(r, cipher, &n)

	if res != "4x dog, 5x car\n" {
		t.Errorf("expected: %q, got %q\n", "4x dog, 5x car\n", res)
	}
}

func TestDecodeAddPoss(t *testing.T) {
	cipher := []CipherOp{{Opid: AddPoss}, {Opid: AddPoss}, {Opid: End}}
	r, w := io.Pipe()
	n := 0
	go func() {
		w.Write([]byte{0x68, 0x67, 0x70, 0x72, 0x77, 20})
	}()
	res, _ := decodeLine(r, cipher, &n)

	if res != string("hello\n") {
		t.Errorf("expected: %q, got %q\n", "hello\n", res)
	}
}

func TestDecodeFullMessage(t *testing.T) {
	cipher := []CipherOp{{Opid: Xor, Value: 123}, {Opid: AddPoss}, {Opid: Reverse}, {Opid: End}}
	r, w := io.Pipe()
	n := 0
	go func() {
		w.Write([]byte{0xf2, 0x20, 0xba, 0x44, 0x18, 0x84, 0xba, 0xaa,
			0xd0, 0x26, 0x44, 0xa4, 0xa8, 0x7e})
	}()
	res, _ := decodeLine(r, cipher, &n)

	want := "4x dog,5x car\n"
	if res != want {
		t.Errorf("expected: %q, got %q\n", want, res)
	}
}

func TestMostCopiesOf(t *testing.T) {
	line := "10x toy car,15x dog on a string,4x inflatable motorcycle"

	got := mostCopiesOf(line)
	want := "15x dog on a string\n"

	if got != want {
		t.Errorf("want %q, got %q\n", want, got)
	}
}

func TestHandleConnection(t *testing.T) {
	in, out := net.Pipe()
	t.Log("test")
	go func() {
		in.Write([]byte{0x02, 0x7b, 0x05, 0x01, 0x00})
		in.Write([]byte{0xf2, 0x20, 0xba, 0x44, 0x18, 0x84, 0xba, 0xaa,
			0xd0, 0x26, 0x44, 0xa4, 0xa8, 0x7e})
		br := bufio.NewReader(in)
		for i := range 7 {
			b, _ := br.ReadByte()
			log.Printf("%d, read: %x\n", i, b)
		}
		in.Write([]byte{0x6a, 0x48, 0xd6, 0x58, 0x34, 0x44, 0xd6, 0x7a,
			0x98, 0x4e, 0x0c, 0xcc, 0x94, 0x31})
		for range 7 {
			b, _ := br.ReadByte()
			log.Printf("read: %x\n", b)
		}
		in.Close()
	}()
	handleConnection(out)
}
