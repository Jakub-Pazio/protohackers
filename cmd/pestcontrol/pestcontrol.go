package main

import (
	"bean/pkg/pserver"
	"bufio"
	"flag"
	"log"
	"net"
)

var portNumber = flag.Int("port", 4242, "Port number of server")

func main() {

	flag.Parse()

	s := Server{ASClients: make(map[uint32]*Client)}
	go s.Initialize()

	handler := pserver.WithMiddleware(
		s.handleConnection,
		pserver.LoggingMiddleware,
	)

	log.Fatal(pserver.ListenServe(handler, *portNumber))
}

type Server struct {
	ASClients  map[uint32]*Client
	ActionChan chan func()
}

func (s *Server) Initialize() {
	for {
		f := <-s.ActionChan
		f()
	}
}

func (s *Server) GetClient(site uint32) (*Client, error) {
	type result struct {
		c   *Client
		err error
	}

	ch := make(chan result)

	s.ActionChan <- func() {
		client, ok := s.ASClients[site]
		if !ok {
			newclient, err := NewClient(int(site))
			if err != nil {
				ch <- result{nil, err}
				return
			}
			s.ASClients[site] = &newclient
			client = &newclient
		}

		ch <- result{client, nil}
	}

	res := <-ch
	return res.c, res.err
}

func (s *Server) handleConnection(conn net.Conn) {
	br := bufio.NewReader(conn)

	_, err := ReadHelloMessage(br)
	if err != nil {
		log.Printf("Error reading Hello message: %v\n", err)
		errMsg := &ErrorMessage{Message: err.Error()}
		WriteMessage(conn, errMsg)
		conn.Close()
		return
	}
	log.Println("Received valid hello message")

	replyMsg := &HelloMessage{Protocol: "pestcontrol", Version: 1}
	err = WriteMessage(conn, replyMsg)
	if err != nil {
		log.Printf("failed sending message: %v\n", err)
		conn.Close()
		return
	}

	for {
		msg, err := ReadSiteVisitMessage(br)
		if err != nil {
			log.Printf("Error reading SiteVisit message: %v\n", err)
			errMsg := &ErrorMessage{Message: err.Error()}
			WriteMessage(conn, errMsg)
			conn.Close()
			return
		}

		if err = VerifyVisitSite(msg); err != nil {
			log.Printf("Invalid SiteVisit message: %v\n", err)
			errMsg := &ErrorMessage{Message: err.Error()}
			WriteMessage(conn, errMsg)
			conn.Close()
			return
		}

		log.Printf("SiteVisit message: %+v\n", msg)

		client, err := s.GetClient(msg.Site)
		if err != nil {
			log.Printf("Could not create client for site %d: %v\n", msg.Site, err)
		}

		if err = client.AdjustPolicy(msg.Populations); err != nil {
			log.Printf("Error adjusting policy for site %d: %v\n", msg.Site, err)
		}
	}
}
