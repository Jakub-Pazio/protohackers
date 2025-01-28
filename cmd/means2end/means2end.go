package main

import (
	"fmt"
	"github.com/huandu/skiplist"
	"log"
	"math/big"
	"net"
)

type Store struct {
	prices skiplist.SkipList

	//mu sync.Mutex
}

func NewStore() *Store {
	return &Store{
		prices: *skiplist.New(skiplist.Int32Asc),
	}
}

func (s *Store) AddPrice(time, price int32) {
	log.Printf("adding value: %d, for time: %d\n", price, time)
	//s.mu.Lock()
	//defer s.mu.Unlock()

	s.prices.Set(time, price)
}

func (s *Store) AvgFromRange(start int32, end int32) int32 {
	log.Printf("query for range at: %d - %d\n", start, end)
	arr := make([]int32, 0)
	var l int64
	var sum big.Int
	//s.mu.Lock()
	//defer s.mu.Unlock()
	first := s.prices.Find(start)
	if first == nil || first.Key().(int32) > end {
		return 0
	}
	start = first.Key().(int32)
	log.Printf("%d: time: %d, value: %d", len(arr), first.Key().(int32), first.Value.(int32))
	for start <= end {
		arr = append(arr, first.Value.(int32))
		sum = *big.NewInt(0).Add(&sum, big.NewInt(int64(first.Value.(int32))))
		l++
		log.Printf("%d: time: %d, value: %d", len(arr), first.Key().(int32), first.Value.(int32))
		first = s.prices.Find(start + 1)
		if first == nil {
			break
		}
		start = first.Key().(int32)
	}
	for _, v := range arr {
		fmt.Printf("%x ", v)
	}
	log.Printf("calulationg avg from sum: %d, and len: %d\n", sum, l)
	if len(arr) == 0 {
		return 0
	}
	res := big.NewInt(0).Div(&sum, big.NewInt(l))
	return int32(res.Int64())
	//sort.Slice(arr, func(i, j int) bool {
	//	return arr[i] < arr[j]
	//})
	//log.Println("calculating result from", arr)
	//if len(arr)%2 == 1 {
	//	return arr[len(arr)/2]
	//}
	//return (arr[len(arr)/2-1] + arr[len(arr)/2]) / 2
}

func main() {
	ln, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatalf("cannot bind to tcp socket: %v\n", err)
	}
	log.Println("server started successfully")
	for {
		store := NewStore()
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("cannot accept connection: %v\n", err)
		}
		log.Println("accepted connection")
		go handleConnection(conn, store)
	}
}

func handleConnection(conn net.Conn, store *Store) {
	defer func(conn net.Conn) {
		log.Println("closing connection")
		err := conn.Close()
		if err != nil {
			log.Printf("error when closing connection: %v\n", err)
		}
		log.Println("connection closed")
	}(conn)
	//TODO: Create two goroutines; one reading from socket and sending 9 bytes to another goroutine
	// 		that will handle dealing with logic
	msg := make([]byte, 9)
	buf := make([]byte, 1024)
	offset := 0
	handled := 0
	mv := 0
	for {
		n, err := conn.Read(buf[mv:])
		if n == 0 {
			return
		}
		log.Printf("read: %d bytes\n", n)
		log.Printf("mv: %d, n: %d, first 18bytes: %x\n", mv, n, buf[:17])
		for i := range n {
			fmt.Printf("%x ", buf[i+mv])
		}
		fmt.Println()
		offset += n
		if offset < 9 {
			mv += n
			continue
		}
		// Here we have enough data to handle message

		// Until we dealt with all "full messages"
		for handled+9 <= offset {
			log.Printf("got (at least) whole message; offset: %d, handled: %d\n", offset, handled)

			msg = buf[handled : handled+9]
			fmt.Print("Processing msg: ")
			for i := range len(msg) {
				fmt.Printf("%x ", msg[i])
			}
			fmt.Println()
			handled += 9

			if err != nil {
				log.Printf("error reading from sokcet: %v\n", err)
				break
			}
			log.Printf("received bytes: %x\n", msg)
			// Here goes logic
			if msg[0] == byte('I') {
				var timestamp int32
				timestamp |= int32(msg[1]) << 24
				timestamp |= int32(msg[2]) << 16
				timestamp |= int32(msg[3]) << 8
				timestamp |= int32(msg[4])
				var price int32
				price |= int32(msg[5]) << 24
				price |= int32(msg[6]) << 16
				price |= int32(msg[7]) << 8
				price |= int32(msg[8])
				//for _, b := range msg {
				//	log.Printf("%02x ", b)
				//}
				//log.Println()
				//log.Printf("time %d, price %d\n", timestamp, price)
				store.AddPrice(timestamp, price)
			} else if msg[0] == byte('Q') {
				var start int32
				start ^= int32(msg[1]) << 24
				start ^= int32(msg[2]) << 16
				start ^= int32(msg[3]) << 8
				start ^= int32(msg[4])
				var end int32
				end ^= int32(msg[5]) << 24
				end ^= int32(msg[6]) << 16
				end ^= int32(msg[7]) << 8
				end ^= int32(msg[8])
				log.Printf("start: %d, end:%d\n", start, end)
				res := store.AvgFromRange(start, end)
				resBytes := make([]byte, 4)
				resBytes[0] = byte(res >> 24)
				resBytes[1] = byte(res >> 16)
				resBytes[2] = byte(res >> 8)
				resBytes[3] = byte(res)
				conn.Write(resBytes)
			} else {
				log.Printf("undefined message type with value: %x\n", msg[0])
				return
			}
			//Here logic ends(?)
			//wn, err := conn.Write(msg[:n])
			//log.Printf("read and send: %s", msg[:n])
			//if err != nil {
			//	log.Printf("error writing to sokcet: %v\n", err)
			//}
			//if wn != n {
			//	log.Printf("read %d bytes but wrote %d bytes\n", n, wn)
			//}
		}
		log.Printf("not enough bytes for message: offset: %d, handled: %d\n", offset, handled)
		rest := offset - handled
		log.Printf("unhandled data in the buffer (%d bytes) :\n buffer(100): ", rest)
		for i := range 100 {
			fmt.Printf("%d: %x, ", i, buf[i])
		}
		fmt.Println()
		for i := range rest {
			fmt.Printf("writing from buf at position: %d ", handled+i)
			fmt.Printf("%x\n", buf[handled+i])
			buf[i] = buf[handled+i]
		}
		fmt.Println()
		mv = rest
		offset = rest
		handled = 0
	}
}
