package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
)

var portNumber = flag.Int("port", 4242, "Port number of server")

const maxSize = 900

type LineServer struct {
	Sessions map[Session]*SessionStruct

	// SessionsChan allows sessions to comunicate back to server
	// for example when they want to shutdown itself due to timeout
	SessionsChan chan Session
}

type DataPayload struct {
	Position int
	Data     string
}

type SessionStruct struct {
	id         Session
	remoteAddr *net.UDPAddr
	ln         *net.UDPConn

	DataChan  chan DataPayload
	AckChan   chan int
	CloseChan chan struct{}
	RetyChan  chan []byte
	AppChan   chan string

	readingOffset int

	ackExpect     int
	ackLast       int
	sendingString string

	serverChan chan Session

	al *AppLayer
}

type AppLayer struct {
	currentString string
	sendChan      chan string
}

func (al *AppLayer) AcceptString(s string) {
	working := al.currentString + s

	parts := strings.SplitN(working, "\n", 2)
	if len(parts) == 1 {
		al.currentString = working
		return
	}
	toSend, rest := parts[0], parts[1]
	// we have reached message we want to send
	// we should reverse this part, stick newline at the end
	// pass it to the lower layer and try to send new message from currentString
	al.currentString = rest

	if len(toSend) > 0 {
		// we want to send only non-empty messages
		var sb strings.Builder
		for i := range len(toSend) {
			sb.WriteByte(toSend[len(toSend)-1-i])
		}
		sb.WriteByte('\n')
		go func() {
			al.sendChan <- sb.String()
		}()
	}

	// recursive call, we want to send all possible messages from currentString
	al.AcceptString("")
}

func (ss *SessionStruct) Act() {
	for {
		select {
		case payload := <-ss.DataChan:
			log.Printf("payload[%d]: %q\n", payload.Position, payload.Data)
			if payload.Position > ss.readingOffset {
				log.Printf("Have not received previous data, resending ack at %d\n", ss.readingOffset)
				msg := fmt.Sprintf("/ack/%d/%d/", ss.id, ss.readingOffset)
				if _, err := ss.Write([]byte(msg)); err != nil {
					log.Printf("Failed sending repeat ACK for %d, at %d: %v\n", ss.id, ss.readingOffset, err)
				}
				continue
			}
			unescaped, err := unescapeMsg(payload.Data)
			if err != nil {
				fmt.Printf("Error in message, droping message: %v\n", err)
				continue
			}
			msg := fmt.Sprintf("/ack/%d/%d/", ss.id, len(unescaped))
			log.Printf("sending ack msg: %q\n", msg)
			if _, err := ss.Write([]byte(msg)); err != nil {
				log.Printf("Failed sending ack for new data for %d: %v\n", ss.id, err)
			}
			ss.al.AcceptString(unescaped)
		case ackLen := <-ss.AckChan:
			if !(ackLen > ss.ackLast) {
				//do nothing and stop
				continue
			}
			if ackLen > ss.ackExpect {
				// peer is misbehaving, close connection
				ss.serverChan <- ss.id
				return
			}
			//TODO: handle 2 more cases, move this logic to function
			ss.SendFrom(ackLen)
			log.Printf("%d\n", ackLen)

		case <-ss.CloseChan:
			msg := fmt.Sprintf("/close/%d/", ss.id)
			ss.Write([]byte(msg))
			ss.serverChan <- ss.id
			return
		case <-ss.RetyChan:
			//TODO: register last send msg, if not acked resend
		case s := <-ss.AppChan:
			ss.sendingString += s
			if ss.ackLast == ss.ackExpect {
				ss.SendFrom(ss.ackLast)
			}
			// case <-time.After(time.Second * 60):
			// 	ss.serverChan <- ss.id
			// 	return
		}
	}
}

// SendFrom sends next packet from position it received ack, when no data is available its noop
func (ss *SessionStruct) SendFrom(ackLen int) {
	currentLen := 0
	msgOffset := ackLen
	var sb strings.Builder
	for msgOffset < len(ss.sendingString) && currentLen < maxSize {
		next := ss.sendingString[msgOffset]
		if next == '\\' || next == '/' {
			sb.WriteByte('\\')
			currentLen++
		}
		sb.WriteByte(next)
		currentLen++
		msgOffset++
	}

	if sb.Len() == 0 {
		log.Printf("No more data to send for session: %d\n", ss.id)
		return
	}

	msg := fmt.Sprintf("/data/%d/%d/%s/", ss.id, ackLen, sb.String())
	log.Printf("Sending data: %q\n", msg)
	ss.Write([]byte(msg))
}

func unescapeMsg(s string) (string, error) {
	var sb strings.Builder
	escaping := false

	for _, r := range s {
		if escaping {
			if r == '\\' || r == '/' {
				sb.WriteRune(r)
				escaping = false
			} else {
				return "", fmt.Errorf("illegal backslash in data")
			}
		} else {
			switch r {
			case '\\':
				escaping = true
			case '/':
				return "", fmt.Errorf("illegal slash in data")
			default:
				sb.WriteRune(r)
			}
		}
	}

	return sb.String(), nil
}

func (ss *SessionStruct) Write(p []byte) (n int, err error) {
	return ss.ln.WriteToUDP(p, ss.remoteAddr)
}

func main() {
	flag.Parse()

	sessionsChan := make(chan Session)
	server := LineServer{
		Sessions:     map[Session]*SessionStruct{},
		SessionsChan: sessionsChan,
	}

	addr := fmt.Sprintf(":%d", *portNumber)
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	ln, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		panic(err)
	}
	log.Printf("server started successfully, running at port: %d\n", *portNumber)
	ListenerLoop(ln, server, sessionsChan)
}

func ListenerLoop(ln *net.UDPConn, server LineServer, schan chan Session) {
	buffer := make([]byte, 1000)
	for {
		n, remoteAddr, err := ln.ReadFromUDP(buffer)
		if err != nil {
			log.Printf("Error reading UDP, from %v, err: %v\n", remoteAddr, err)
		}
		data := string(buffer[:n])
		mtype, session, rest, err := ParseMessage(data)
		log.Printf("M: %q [%s]\n", data, remoteAddr)

		if err != nil {
			log.Printf("Error reading message: %v\n", err)
			continue
		}

		switch mtype {
		case Connect:
			s, ok := server.Sessions[session]
			if !ok {
				// we must create new session
				appChan := make(chan string)
				newSes := &SessionStruct{
					id:            session,
					remoteAddr:    remoteAddr,
					ln:            ln,
					DataChan:      make(chan DataPayload),
					AckChan:       make(chan int),
					CloseChan:     make(chan struct{}),
					RetyChan:      make(chan []byte),
					serverChan:    schan,
					AppChan:       appChan,
					readingOffset: 0,
					ackExpect:     0,
					ackLast:       0,
					sendingString: "",
					al: &AppLayer{
						currentString: "",
						sendChan:      appChan,
					},
				}
				server.Sessions[session] = newSes
				s = newSes
				log.Printf("Created new session %d\n", session)
			}
			msg := fmt.Sprintf("/ack/%d/0/", session)
			if _, err = ln.WriteToUDP([]byte(msg), remoteAddr); err != nil {
				log.Printf("Could not ack connection %d: %v\n", session, err)
			}
			log.Printf("acking to connect for session: %d\n", session)
			log.Printf("r: %q\n", msg)
			if !ok {
				// session was not started, it was not found in the map
				go s.Act()
			}
		case Data:
			pos, data, err := parseData(rest)
			if err != nil {
				log.Printf("invalid data message: %v\n", err)
			}
			s, ok := server.Sessions[session]
			if !ok {
				// we don't have track session associated with this ID,
				// sending close message and closing
				dial, err := net.DialUDP("udp", nil, remoteAddr)
				if err != nil {
					log.Printf("Failed to respond to unknown data: %v\n", err)
				}
				msg := fmt.Sprintf("/close/%d/", session)
				_, err = dial.Write([]byte(msg))
				continue
			}
			go func() {
				s.DataChan <- DataPayload{
					Position: pos,
					Data:     data,
				}
			}()
		}
	}
}

func parseData(rest string) (int, string, error) {
	parts := strings.SplitN(rest, "/", 2)
	if len(parts) != 2 {
		return 0, "", fmt.Errorf("invalid data message\n")
	}

	pos, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, "", fmt.Errorf("invalid data message\n")
	}
	data := parts[1]

	return pos, data, nil
}

type Type string

const (
	Connect Type = "connect"
	Data    Type = "data"
	Ack     Type = "ack"
	Close   Type = "close"
)

type Session int

func ParseMessage(msg string) (Type, Session, string, error) {
	var t Type
	var s Session

	msgLen := len(msg)

	if msgLen < 2 || msg[0] != '/' || msg[msgLen-1] != '/' {
		return t, s, "", fmt.Errorf("first and last character must be slash")
	}

	parts := strings.SplitN(msg[1:msgLen-1], "/", 3)
	if len(parts) < 2 {
		return t, s, "", fmt.Errorf("message needs at least 2 parts")
	}

	switch parts[0] {
	case "connect":
		t = Connect
	case "data":
		t = Data
	case "ack":
		t = Ack
	case "close":
		t = Close
	default:
		return t, s, "", fmt.Errorf("unknown message type %q\n", parts[1])
	}

	sessionNum, err := strconv.Atoi(parts[1])
	if err != nil {
		return t, s, "", fmt.Errorf("could not parse session: %v", err)
	}

	if sessionNum >= 2147483648 || sessionNum < 0 {
		return t, s, "", fmt.Errorf("invalid session id: %d\n", sessionNum)
	}
	s = Session(sessionNum)

	rest := ""
	if len(parts) > 2 {
		rest = parts[2]
	}

	return t, s, rest, nil
}
