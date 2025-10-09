package main

import (
	"bean/pkg/pserver"
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"math/bits"
	"net"
	"slices"
	"strconv"
	"strings"
)

var portNumber = flag.Int("port", 4242, "Port number of server")

func main() {
	flag.Parse()
	handler := pserver.WithMiddleware(
		handleConnection,
		pserver.LoggingMiddleware,
	)
	log.Fatal(pserver.ListenServe(handler, *portNumber))
}

func handleConnection(conn net.Conn) {
	defer pserver.HandleConnShutdown(conn)

	cipher, err := parseCipher(conn)
	log.Printf("Cipher-preopt: %+v", cipher)
	if err != nil {
		log.Printf("Could not parse cipher: %s", err)
		conn.Close()
	}

	cipher, change := reduceCipher(cipher)
	for change {
		cipher, change = reduceCipher(cipher)
	}
	log.Printf("Cipher post-opt: %+v", cipher)

	if !cipherCorrect(cipher) {
		log.Println("Cipher is no-op, cannot continue")
		conn.Close()
		return
	}

	log.Printf("Cipher: %+v", cipher)

	n := -1
	nOut := 0
	for {
		n += 1
		log.Println("New Line from client")
		line, err := decodeLine(conn, cipher, &n)
		log.Println(line)
		if err != nil {
			log.Printf("Closing connction, err: %v\n", err)
			break
		}
		res := mostCopiesOf(line)
		fmt.Printf("Response to send: %q\n", res)
		encodedRes := encodeLine(res, cipher, &nOut)
		//Cheat to see if cipher is no-op
		if encodedRes == res {
			fmt.Println("response is same as request, dropping connection")
			break
		}
		conn.Write([]byte(encodedRes))
		log.Println("End of line")
	}

	conn.Close()
}

func decode(b byte, cipher []CipherOp, n int) (byte, bool) {
	for i := len(cipher) - 2; i >= 0; i-- {
		c := cipher[i]
		switch c.Opid {
		case Reverse:
			b = bits.Reverse8(b)
		case Xor:
			b = b ^ c.Value
		case XorPos:
			b = b ^ byte(n)
		case Add:
			b -= byte(c.Value)
		case AddPoss:
			b -= byte(n)
		}
	}
	if b == '\n' {
		return b, true
	}
	return b, false
}

func decodeLine(r io.Reader, c []CipherOp, n *int) (string, error) {
	log.Printf("Decoding line, start offset: %d\n", *n)
	bs := make([]byte, 0)
	br := bufio.NewReader(r)
	b, err := br.ReadByte()
	if err != nil {
		return "", err
	}
	dec, end := decode(b, c, *n)
	bs = append(bs, dec)
	for !end {
		*n++
		b, err := br.ReadByte()
		if err != nil {
			return "", err
		}
		dec, end = decode(b, c, *n)
		bs = append(bs, dec)
	}
	*n++
	return string(bs), nil
}

func encode(b byte, cipher []CipherOp, n int) byte {
	for _, c := range cipher {
		switch c.Opid {
		case Reverse:
			b = bits.Reverse8(b)
		case Xor:
			b = b ^ c.Value
		case XorPos:
			b = b ^ byte(n)
		case Add:
			b += byte(c.Value)
		case AddPoss:
			b += byte(n)
		}
	}
	return b
}

func encodeLine(line string, cipher []CipherOp, n *int) string {
	var res strings.Builder
	for _, b := range []byte(line) {
		eb := encode(b, cipher, *n)
		res.WriteByte(eb)
		*n++
	}

	return res.String()
}

// reduceCipher returnes reduceCipher and if it changed anything
func reduceCipher(cipher []CipherOp) ([]CipherOp, bool) {
	if len(cipher) < 2 {
		return cipher, false
	}

	contAdd := make([]int, 0)
	contXor := make([]int, 0)

	for i, c := range cipher {
		if c.Opid == Add && c.Value == 0 {
			return slices.Delete(cipher, i, i+1), true
		}
		if c.Opid == Xor && c.Value == 0 {
			return slices.Delete(cipher, i, i+1), true
		}
		if c.Opid != Xor {
			if len(contXor) > 0 {
				sum := byte(0)
				for _, v := range contXor {
					sum += byte(v)
				}
				log.Printf("XOR SUM: %d\n", sum)
				if sum == 0 {
					return slices.Delete(cipher, i-len(contXor), i), true
				} else {
					contXor = nil
				}
			}
		} else {
			contXor = append(contXor, int(c.Value))
		}
		if c.Opid != Add {
			// this could be the end of ADD's that sum to 0, we need to check
			// and remove them from cipher
			if len(contAdd) > 0 {
				log.Printf("summing...")
				sum := byte(0)
				for _, v := range contAdd {
					sum += byte(v)
				}
				log.Printf("sum is: %d\n", sum)
				if sum == 0 {
					return slices.Delete(cipher, i-len(contAdd), i), true
				} else {
					contAdd = nil
				}
			}
		} else {
			log.Printf("Adding %d to list\n", c.Value)
			contAdd = append(contAdd, int(c.Value))
		}
	}

	currOp := cipher[1]
	prevOp := cipher[0]

	for i := 2; i < len(cipher); i++ {
		// reducing ciphers if we need to reverse twice in a row
		if currOp.Opid == Reverse && prevOp.Opid == Reverse {
			cipher = slices.Delete(cipher, i-2, i)
			return cipher, true
		}

		if currOp.Opid == Xor && prevOp.Opid == Xor && currOp.Value == prevOp.Value {
			cipher = slices.Delete(cipher, i-2, i)
			return cipher, true
		}

		prevOp = currOp
		currOp = cipher[i]
	}

	if len(cipher) > 3 {
		first := cipher[0]
		next := cipher[1]
		last := cipher[2]
		for i := 3; i < len(cipher); i++ {
			if first.Opid == Xor && next.Opid == Xor && last.Opid == Xor {
				f := first.Value
				n := next.Value
				l := last.Value
				if (f | n) == l {
					cipher = slices.Delete(cipher, i-3, i)
					return cipher, true
				}
			}
		}
	}

	return cipher, false
}

// If a client tries to use a cipher that leaves every byte of
// input unchanged, the server must immediately disconnect
// without sending any data back.
func cipherCorrect(cipher []CipherOp) bool {
	if len(cipher) == 1 {
		return false
	}

	return true
}

// OPid's
// 00 - End of cipher spec (we end if we find it)
// 01 (reversebits) - reverse the order of bits in byte
// 02 N (xor N) - XOR the byte by the value N (every byte?)
// 03 (xorpos) - XOR the byte by its position in the stream, starting from 0
//		 (every byte?)
// 04 N (add N) - Add n to the byte, modulo 256, can add 0 and it wraps
// 05 (addpos) - Add the position in the stream, so for 2nd byte it adds 2

type Opid int

const (
	End Opid = iota
	Reverse
	Xor
	XorPos
	Add
	AddPoss
)

type CipherOp struct {
	Opid  Opid // value from 0 to 5 denoting operation
	Value byte // optional value of operation
}

func parseCipher(conn net.Conn) ([]CipherOp, error) {
	res := make([]CipherOp, 0)

	reader := bufio.NewReader(conn)

	loop := true
	for loop {
		first, err := reader.ReadByte()
		if err != nil {
			return res, err
		}

		switch Opid(first) {
		case End:
			res = append(res, CipherOp{Opid: End})
			loop = false
		case Reverse:
			res = append(res, CipherOp{Opid: Reverse})
		case Xor:
			snd, err := reader.ReadByte()
			if err != nil {
				return res, err
			}
			res = append(res, CipherOp{Opid: Xor, Value: snd})
		case XorPos:
			res = append(res, CipherOp{Opid: XorPos})
		case Add:
			snd, err := reader.ReadByte()
			if err != nil {
				return res, err
			}
			res = append(res, CipherOp{Opid: Add, Value: snd})
		case AddPoss:
			res = append(res, CipherOp{Opid: AddPoss})
		default:
			return res, fmt.Errorf("could not parse, opcode %x", first)
		}
	}

	return res, nil
}

func mostCopiesOf(line string) string {
	topF := 0
	topV := ""
	list := strings.Split(line, ",")
	for _, item := range list {
		parts := strings.Split(item, "x")
		amount, _ := strconv.Atoi(parts[0])
		if amount > topF {
			topF = amount
			topV = strings.TrimSpace(parts[1])
		}
	}
	return fmt.Sprintf("%dx %s\n", topF, topV)
}
