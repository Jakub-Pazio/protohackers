package main

import (
	"bean/pkg/pserver"
	"bufio"
	"flag"
	"io"
	"log"
	"net"
	"sync/atomic"
)

var portNumber = flag.Int("port", 4242, "Port number of server")

var idGenClient atomic.Int32

func newClientId() int {
	return int(idGenClient.Add(1))
}

func main() {
	flag.Parse()

	s := Server{ASClients: make(map[uint32]*Client), ActionChan: make(chan func())}
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
		log.Printf("Getting/Creating client for AS\n")
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
			newclient, err := NewClient(site)
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
	clientId := newClientId()
	log.Printf("New client with Id: %d\n", clientId)

	_, err := ReadHelloMessage(br)
	if err != nil {
		log.Printf("Error reading Hello message: %v\n", err)
		replyMsg := &HelloMessage{Protocol: "pestcontrol", Version: 1}
		WriteMessage(conn, replyMsg)
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
		if err == io.EOF {
			log.Printf("EOF, disconecting client %d\n", clientId)
			conn.Close()
		} else if err != nil {
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
			errMsg := &ErrorMessage{Message: err.Error()}
			WriteMessage(conn, errMsg)
			conn.Close()
			return
		}

		log.Printf("client created for site %d\n", msg.Site)

		if err = client.AdjustPolicy(msg.Populations); err != nil {
			log.Printf("Error adjusting policy for site %d: %v\n", msg.Site, err)
			errMsg := &ErrorMessage{Message: err.Error()}
			WriteMessage(conn, errMsg)
			conn.Close()
			return
		}

		log.Printf("Adjusted policy for site %d\n", msg.Site)
	}
}
