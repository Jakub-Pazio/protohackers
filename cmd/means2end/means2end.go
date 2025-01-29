package main

import (
	"bean/pkg/pserver"
	"encoding/binary"
	"flag"
	"fmt"
	"github.com/huandu/skiplist"
	"log"
	"math/big"
	"net"
)

const BufferSize = 1024
const MessageLength = 9

var portNumber = flag.Int("port", 4242, "Port number of server")

func main() {
	log.Fatal(pserver.ListenServe(handleConnection, *portNumber))
}

func handleConnection(conn net.Conn) {
	defer pserver.HandleConnShutdown(conn)

	store := NewStore()
	buf := make([]byte, BufferSize)
	offset := 0
	handled := 0

	for {
		n, err := conn.Read(buf[offset:])
		if n == 0 {
			log.Println("end of data from client")
			return
		}
		offset += n

		if offset < 9 {
			// No full message in the buffer, we need to gather more bytes from connection
			continue
		}

		messagesNumber := offset / MessageLength
		rest := offset % MessageLength
		handled = messagesNumber * MessageLength

		messagesData := buf[:handled]
		err = HandleMessages(conn, messagesData, store)

		for i := range rest {
			buf[i] = buf[handled+i]
		}

		if err != nil {
			log.Printf("could not handle message: %v\n", err)
			return
		}

		offset = rest
		handled = 0
	}
}

func HandleMessages(conn net.Conn, buf []byte, store *Store) error {
	handled := 0
	offset := len(buf)
	for handled+9 <= offset {
		log.Println("Processing msg")
		msg := buf[handled : handled+9]
		handled += 9

		switch msg[0] {
		case byte('I'):
			timestamp, err := ConvMsg(msg[1:5])
			if err != nil {
				return fmt.Errorf("could not parse timstamp: %v", err)
			}
			price, err := ConvMsg(msg[5:9])
			if err != nil {
				return fmt.Errorf("could not parse price: %v", err)
			}
			store.AddPrice(timestamp, price)
		case byte('Q'):
			start, err := ConvMsg(msg[1:5])
			if err != nil {
				return fmt.Errorf("could not parse timstamp: %v", err)
			}
			end, err := ConvMsg(msg[5:9])
			if err != nil {
				return fmt.Errorf("could not parse timstamp: %v", err)
			}
			res := store.AvgFromRange(start, end)
			resBytes := writeToBytes(res)
			_, _ = conn.Write(resBytes)
		default:
			return fmt.Errorf("undefined message type with value: %x\n", msg[0])
		}
	}
	return nil
}

func writeToBytes(res int32) []byte {
	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf, uint32(res))
	return buf
}

type Store struct {
	prices *skiplist.SkipList
}

func NewStore() *Store {
	return &Store{
		prices: skiplist.New(skiplist.Int32Asc),
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

	log.Printf("deriving avg from sum: %v, and len: %d\n", sum, l)
	res := big.NewInt(0).Div(&sum, big.NewInt(l))
	return int32(res.Int64())
}
func ConvMsg(msg []byte) (int32, error) {
	if len(msg) != 4 {
		return 0, fmt.Errorf("invalid message length: %d", len(msg))
	}
	var result int32
	result |= int32(msg[0]) << 24
	result |= int32(msg[1]) << 16
	result |= int32(msg[2]) << 8
	result |= int32(msg[3])
	return result, nil
}
