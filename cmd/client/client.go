package main

import (
	"net"
	"time"
)

func main() {
	conn, err := net.Dial("tcp", "localhost:8080")
	if err != nil {
		// handle error
	}
	//buff := []byte{0x49, 0, 0, 0x30, 0x39, 0, 0, 0, 0x65}
	//conn.Write(buff)
	//buff = []byte{0x49, 0, 0, 0x30, 0x3a, 0, 0, 0, 0x66}
	//conn.Write(buff)
	//buff = []byte{0x49, 0, 0, 0x30, 0x3b, 0, 0, 0, 0x64}
	//conn.Write(buff)
	//buff = []byte{0x49, 0, 0, 0xa0, 0, 0, 0, 0, 0x5}
	//conn.Write(buff)
	//buff = []byte{0x51, 0, 0, 0x30, 0, 0, 0, 0x40, 0}
	//conn.Write(buff)
	//res := make([]byte, 4)
	//conn.Read(res)

	buff := []byte{0x49}
	conn.Write(buff)
	time.Sleep(100 * time.Millisecond) // Introduce a delay

	buff = []byte{0, 0, 0x30, 0x39}
	conn.Write(buff)
	time.Sleep(100 * time.Millisecond) // Introduce a delay

	buff = []byte{0, 0, 0, 0x65}
	conn.Write(buff)
	time.Sleep(100 * time.Millisecond) // Introduce a delay

	buff = []byte{0x49}
	conn.Write(buff)
	time.Sleep(100 * time.Millisecond) // Introduce a delay

	buff = []byte{0, 0, 0x30, 0x39}
	conn.Write(buff)
	time.Sleep(100 * time.Millisecond) // Introduce a delay

	buff = []byte{0, 0, 0, 0x65}
	conn.Write(buff)
	time.Sleep(100 * time.Millisecond) // Introduce a delay
	//
	//buff = []byte{0x49}
	//conn.Write(buff)
	//
	//buff = []byte{0x49}
	//conn.Write(buff)

	//for _, b := range res {
	//	fmt.Printf("%x ", b)
	//}
}
