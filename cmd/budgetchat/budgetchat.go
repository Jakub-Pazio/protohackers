package main

import (
	"bean/pkg/pserver"
	"bufio"
	"flag"
	"fmt"
	"log"
	"net"
	"regexp"
	"strings"
	"sync"
)

const BuffSize = 1024

var portNumber = flag.Int("port", 4242, "Port number of server")

func main() {
	flag.Parse()
	server := NewServer()
	handler := pserver.WithMiddleware(
		server.handleConnection,
		pserver.LoggingMiddleware,
	)

	log.Fatal(pserver.ListenServe(handler, *portNumber))
}

func isValidUsername(username string) bool {
	re := regexp.MustCompile(`^[a-zA-Z0-9]+$`)
	return re.MatchString(username)
}

type Server struct {
	users map[string]chan string

	mu sync.Mutex
}

func NewServer() *Server {
	return &Server{
		users: make(map[string]chan string),
		mu:    sync.Mutex{},
	}
}

func (s *Server) AddUser(name string) (chan string, error) {
	if ok := isValidUsername(name); !ok {
		return nil, fmt.Errorf("user name %s is invalid", name)
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.users[name]; ok {
		return nil, fmt.Errorf("user named %s already exists", name)
	}
	ch := make(chan string, 100)
	s.users[name] = ch
	for uname, userCh := range s.users {
		if uname != name {
			userCh <- fmt.Sprintf("* %s has entered the room\n", name)
		}
	}
	return ch, nil
}

func (s *Server) RemoveUser(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.users, name)
	for _, userChan := range s.users {
		userChan <- fmt.Sprintf("* %s has left the room\n", name)
	}

	return nil
}

func (s *Server) SendMessage(name string, msg string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for uname, userChan := range s.users {
		if uname != name {
			userChan <- fmt.Sprintf("[%s] %s\n", name, msg)
		}
	}
}

func (s *Server) GetParticipants(name string) []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := make([]string, 0)
	for uname, _ := range s.users {
		if uname != name {
			result = append(result, uname)
		}
	}
	return result
}

func (s *Server) handleConnection(conn net.Conn) {
	defer pserver.HandleConnShutdown(conn)
	_, _ = conn.Write([]byte("Welcome to budgetchat! What shall I call you?\n"))
	reader := bufio.NewReaderSize(conn, BuffSize)
	line, err := reader.ReadString('\n')
	if err != nil {
		log.Printf("error when reading from socket: %v", err)
		return
	}
	line = strings.TrimSpace(line)

	ch, err := s.AddUser(line)

	if err != nil {
		_, _ = conn.Write([]byte("Something went wrong"))
		log.Printf("error: %s\n", err)
		return
	}
	uname := line

	defer s.RemoveUser(line)
	go func(ch chan string) {
		for {
			msg := <-ch
			conn.Write([]byte(msg))
		}
	}(ch)

	msg := "* The room contains:"
	for _, name := range s.GetParticipants(line) {
		msg += fmt.Sprintf(" %s,", name)
	}
	if msg[len(msg)-1] == ',' {
		msg = msg[:len(msg)-1]
	}
	msg += "\n"
	_, _ = conn.Write([]byte(msg))

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			log.Printf("error when reading from socket: %v", err)
			return
		}
		line = strings.TrimSpace(line)
		log.Printf("%s: %s", uname, line)
		s.SendMessage(uname, line)
	}
}
