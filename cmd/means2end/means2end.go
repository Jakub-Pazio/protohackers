package main

import (
	"bean/pkg/pserver"
	"errors"
	"fmt"
	"github.com/huandu/skiplist"
	"log"
	"math/big"
	"net"
)

func main() {
	log.Fatal(pserver.ListenServe(handleConnection, 4242))
}

func handleConnection(conn net.Conn) {
	defer pserver.HandleConnShutdown(conn)
	store := NewStore()
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

			switch {
			case msg[0] == byte('I'):
				timestamp, err := ConvMsg(msg[1:5])
				if err != nil {
					log.Printf("could not parse timstamp: %v", err)
					//TODO: handle closing connection
				}
				price, err := ConvMsg(msg[5:9])
				if err != nil {
					log.Printf("could not parse timstamp: %v", err)
					//TODO: handle closing connection
				}
				store.AddPrice(timestamp, price)
			case msg[0] == byte('Q'):
				start, err := ConvMsg(msg[1:5])
				if err != nil {
					log.Printf("could not parse timstamp: %v", err)
					//TODO: handle closing connection
				}
				end, err := ConvMsg(msg[5:9])
				if err != nil {
					log.Printf("could not parse timstamp: %v", err)
					//TODO: handle closing connection
				}
				log.Printf("start: %d, end:%d\n", start, end)
				res := store.AvgFromRange(start, end)
				resBytes := make([]byte, 4)
				resBytes[0] = byte(res >> 24)
				resBytes[1] = byte(res >> 16)
				resBytes[2] = byte(res >> 8)
				resBytes[3] = byte(res)
				conn.Write(resBytes)
			default:
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

type Store struct {
	prices skiplist.SkipList
}

func NewStore() *Store {
	return &Store{
		prices: *skiplist.New(skiplist.Int32Asc),
	}
}

func (s *Store) AddPrice(time, price int32) {
	log.Printf("adding value: %d, for time: %d\n", price, time)
	s.prices.Set(time, price)
}

func (s *Store) AvgFromRange(start int32, end int32) int32 {
	log.Printf("query for range at: %d - %d\n", start, end)
	var l int64
	var sum big.Int

	first := s.prices.Find(start)
	if first == nil || first.Key().(int32) > end {
		return 0
	}
	start = first.Key().(int32)
	for start <= end {
		sum = *big.NewInt(0).Add(&sum, big.NewInt(int64(first.Value.(int32))))
		l++
		first = s.prices.Find(start + 1)
		if first == nil {
			break
		}
		start = first.Key().(int32)
	}

	log.Printf("calulationg avg from sum: %v, and len: %d\n", sum, l)
	res := big.NewInt(0).Div(&sum, big.NewInt(l))
	return int32(res.Int64())
}

func WrongLenMessage(l int) error { return errors.New(fmt.Sprintf("Message with wrong lenght: %d", l)) }

func ConvMsg(msg []byte) (int32, error) {
	if len(msg) != 4 {
		return 0, WrongLenMessage(len(msg))
	}
	var result int32
	result |= int32(msg[0]) << 24
	result |= int32(msg[1]) << 16
	result |= int32(msg[2]) << 8
	result |= int32(msg[3])
	return result, nil
}
