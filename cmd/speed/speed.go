package main

import (
	"bean/pkg/pserver"
	"bufio"
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"sync"
	"time"
)

var portNumber = flag.Int("port", 4242, "Port number of server")

func main() {
	flag.Parse()
	server := NewServer()
	log.SetOutput(os.Stdout) // Redirect logs to stdout
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	handler := pserver.WithMiddleware(
		server.handleConnection,
		pserver.LoggingMiddleware,
	)

	log.Fatal(pserver.ListenServe(handler, *portNumber))
}

type Server struct {
	roads       map[uint16]*Road
	sendTickets map[string]*PastTickets // for a certain car

	mu sync.Mutex
}

type PastTickets struct {
	days map[int]struct{}

	mu sync.Mutex
}

func (s *Server) addNew(car string, day1, day2 int) bool {
	s.mu.Lock()
	if _, ok := s.sendTickets[car]; !ok {
		d := make(map[int]struct{})
		d[day1] = struct{}{}
		d[day2] = struct{}{}
		s.sendTickets[car] = &PastTickets{
			days: d,
			mu:   sync.Mutex{},
		}
		s.mu.Unlock()
		return true
	}
	s.mu.Unlock()
	defer s.sendTickets[car].mu.Unlock()

	s.sendTickets[car].mu.Lock()
	if _, ok := s.sendTickets[car].days[day1]; ok {
		return false
	}
	if _, ok := s.sendTickets[car].days[day2]; ok {
		return false
	}
	s.sendTickets[car].days[day1] = struct{}{}
	s.sendTickets[car].days[day2] = struct{}{}
	return true
}

type Road struct {
	limit        uint16
	measurements map[string][]Measurement // for a plate number
	dispatcher   []Dispatcher
	number       uint16
	oldTickets   [][]byte

	mu sync.Mutex
}

type Measurement struct {
	timestamp uint32
	mile      uint16
}

type Dispatcher struct {
	dChan chan []byte
}

func NewServer() *Server {
	return &Server{
		roads:       make(map[uint16]*Road),
		sendTickets: make(map[string]*PastTickets),
	}
}

func (s *Server) handleConnection(conn net.Conn) {
	defer pserver.HandleConnShutdown(conn)

	isCamera := false
	isDispatcher := false
	isHeartBiting := false

	// If we are camera we will set those values

	var connRoad uint16
	var connMile uint16

	reader := bufio.NewReader(conn)
	for {
		msgType, err := reader.ReadByte()
		if err != nil {
			log.Printf("error reading type: %q", err)
			return
		}
		switch msgType {
		case 0x20:
			// Plate
			_, plate, err := readString(reader)
			if err != nil {
				log.Printf("error reading plate: %v\n", err)
				return
			}
			timestamp, err := read32(reader)
			if !isCamera {
				s.sendError("you are not camera", conn)
			}
			s.addMeasurement(connRoad, connMile, timestamp, plate)
		case 0x40:
			// WantHeartbeat
			if isHeartBiting {
				log.Print("Already sending hb\n")
				return
			}
			interval, err := read32(reader)
			if err != nil {
				log.Printf("error when getting interval: %v\n", err)
				return
			}
			isHeartBiting = true
			log.Printf("interval: %d\n", interval)
			if interval == 0 {
				continue
			}
			go func(interval uint32) {
				for {
					_, err := conn.Write([]byte{0x41})
					if err != nil {
						log.Printf("error sending hb: %v\n", err)
						return
					}
					time.Sleep(time.Second * time.Duration(interval) / 10)
				}
			}(interval)
		case 0x80:
			// IAmCamera
			road, err := read16(reader)
			if err != nil {
				log.Printf("error road: %v\n", err)
				return
			}
			mile, err := read16(reader)
			if err != nil {
				log.Printf("error mile: %v\n", err)
				return
			}
			limit, err := read16(reader)
			if err != nil {
				log.Printf("error limit: %v\n", err)
				return
			}
			if isDispatcher || isCamera {
				s.sendError("already registered", conn)
				return
			}
			isCamera = true
			connRoad = road
			connMile = mile
			s.addCamera(road, mile, limit)
		case 0x81:
			// IAmDispatcher
			numRoads, err := read8(reader)
			log.Printf("new dispatcher for %d roads\n", numRoads)
			if err != nil {
				log.Printf("error numRoads: %v\n", err)
				return
			}
			roads := make([]uint16, numRoads)
			for i := range roads {
				log.Println("adding dispatcher to road")
				road, err := read16(reader)
				if err != nil {
					log.Printf("error road: %v\n", err)
					return
				}
				roads[i] = road
			}
			if isCamera || isDispatcher {
				s.sendError("already registered", conn)
				return
			}
			isDispatcher = true
			dCh := s.addDispatcher(roads)
			log.Printf("dispatcher added correctly: %v\n", roads)
			go func(dCh chan []byte, conn net.Conn) {
				for {
					msg := <-dCh
					_, err := conn.Write(msg)
					if err != nil {
						fmt.Printf("error when writing to dispatcher: %v\n", err)
						return
					}
				}
			}(dCh, conn)
		default:
			log.Printf("illegal message with code: %02x\n", msgType)
			s.sendError("illegal message", conn)
			return
		}
	}
}

func read8(reader *bufio.Reader) (uint8, error) {
	buff := make([]byte, 1)
	n, err := io.ReadFull(reader, buff)
	if n != 1 {
		return 0, fmt.Errorf("read not 1 but %d bytes", n)
	}
	if err != nil {
		return 0, err
	}
	return buff[0], nil
}

func read16(reader *bufio.Reader) (uint16, error) {
	buff := make([]byte, 2)
	n, err := io.ReadFull(reader, buff)
	if n != 2 {
		return 0, fmt.Errorf("read not 2 but %d bytes", n)
	}
	if err != nil {
		return 0, err
	}
	return binary.BigEndian.Uint16(buff), nil
}

func read32(reader *bufio.Reader) (uint32, error) {
	buff := make([]byte, 4)
	n, err := io.ReadFull(reader, buff)
	if n != 4 {
		return 0, fmt.Errorf("read not 4 but %d bytes", n)
	}
	if err != nil {
		return 0, err
	}
	return binary.BigEndian.Uint32(buff), nil
}

func readString(reader *bufio.Reader) (int, string, error) {
	l, err := reader.ReadByte()
	if err != nil {
		return 0, "", err
	}
	strBuff := make([]byte, int(l))
	n, err := io.ReadFull(reader, strBuff)
	if err != nil {
		return 0, "", err
	}
	if n != int(l) {
		return 0, "", fmt.Errorf("should read %d bytes, but read %d", int(l), n)
	}
	return int(l) + 1, string(strBuff), nil
}

func (s *Server) addMeasurement(road, mile uint16, timestamp uint32, plate string) {
	r := s.roads[road]
	r.mu.Lock()
	defer r.mu.Unlock()
	ms := r.measurements[plate]
	for _, m := range ms {
		var distance uint16
		var m1 uint16
		var m2 uint16
		if mile > m.mile {
			distance = mile - m.mile
		} else {
			distance = m.mile - mile
		}
		var timeDelta uint32
		var t1 uint32
		var t2 uint32
		if timestamp > m.timestamp {
			t1 = m.timestamp
			t2 = timestamp
			m1 = m.mile
			m2 = mile
			timeDelta = t2 - t1
		} else {
			t1 = timestamp
			t2 = m.timestamp
			m1 = mile
			m2 = m.mile
			timeDelta = t2 - t1
		}
		speed := uint32(distance) * 3600 / timeDelta

		if speed > uint32(r.limit) || (speed >= uint32(r.limit) && uint32(distance)*3600%timeDelta > 0) {
			log.Printf("speed is: %d, limit is %d\n", speed, r.limit)
			if uint32(distance)*3600%timeDelta > timeDelta/2 {
				speed += 1
			}
			day := t1 / 86400
			day2 := t2 / 86400

			if s.addNew(plate, int(day), int(day2)) {
				log.Printf("issuing ticket for %s at day %d\n", plate, day)
				ticket := r.generateTicket(plate, m1, m2, t1, t2, speed)
				if len(r.dispatcher) == 0 {
					log.Println("no dispatchers, storing ticket for later")
					r.oldTickets = append(r.oldTickets, ticket)
				} else {
					r.dispatcher[0].dChan <- ticket
				}
			}
		}
	}
	ms = append(ms, Measurement{
		timestamp: timestamp,
		mile:      mile,
	})
	r.measurements[plate] = ms
}

func (s *Server) addCamera(numRoad, mile, limit uint16) {
	r, ok := s.roads[numRoad]
	if !ok {
		nr := &Road{
			limit:        limit,
			measurements: make(map[string][]Measurement),
			dispatcher:   make([]Dispatcher, 0),
			number:       numRoad,
			oldTickets:   make([][]byte, 0),
			mu:           sync.Mutex{},
		}
		s.roads[numRoad] = nr
	} else {
		r.limit = limit
	}
	log.Printf("road: %d, mile: %d, limit: %d\n",
		numRoad, mile, limit)
}

func (s *Server) addDispatcher(roads []uint16) chan []byte {
	log.Printf("adding new dispatcher on roads %v\n", roads)
	dChan := make(chan []byte, 10)
	for _, rn := range roads {
		r, ok := s.roads[rn]
		dispatcher := Dispatcher{
			dChan: dChan,
		}
		if !ok {
			nr := &Road{
				limit:        0,
				measurements: make(map[string][]Measurement),
				dispatcher:   []Dispatcher{dispatcher},
				number:       rn,
				oldTickets:   make([][]byte, 0),
				mu:           sync.Mutex{},
			}
			s.roads[rn] = nr
		} else {
			r.dispatcher = append(r.dispatcher, dispatcher)
			for _, oldT := range r.oldTickets {
				dChan <- oldT
			}
			r.oldTickets = nil
		}
	}
	return dChan
}

func (s *Server) sendError(msg string, conn net.Conn) {
	l := len(msg)
	response := make([]byte, l+2)
	response[0] = 0x10
	response[1] = byte(l)
	bMsg := []byte(msg)
	copy(bMsg, response[2:])
	conn.Write(response)
}

func (r *Road) generateTicket(plate string, mile1, mile2 uint16, ts1, ts2, speed uint32) []byte {
	ticket := make([]byte, 2+len(plate)+2+2+4+2+4+2)
	ticket[0] = 0x21
	ticket[1] = byte(len(plate))
	copy(ticket[2:], plate)
	offset := 2 + len(plate)

	binary.BigEndian.PutUint16(ticket[offset:], r.number) // Road
	offset += 2
	binary.BigEndian.PutUint16(ticket[offset:], mile1) // Mile 1
	offset += 2
	binary.BigEndian.PutUint32(ticket[offset:], ts1) // Timestamp 1
	offset += 4
	binary.BigEndian.PutUint16(ticket[offset:], mile2) // Mile 2
	offset += 2
	binary.BigEndian.PutUint32(ticket[offset:], ts2)
	offset += 4
	binary.BigEndian.PutUint16(ticket[offset:], uint16(speed)*100)
	return ticket
}
