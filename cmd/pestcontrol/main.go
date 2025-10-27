package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"io"
	"log"
	"net"
	"sync/atomic"

	"bean/cmd/pestcontrol/internal/authority"
	"bean/cmd/pestcontrol/internal/message"
	pserver2 "bean/pkg/pserver/v2"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

const name = "jakubpazio.site/protohackers/pestcontrol"

var tracer = otel.Tracer(name)

var portNumber = flag.Int("port", 4242, "Port number of server")

var idGenClient atomic.Int32

func newClientId() int {
	return int(idGenClient.Add(1))
}

func main() {
	flag.Parse()

	ctx := context.Background()
	shutdown, err := setupOtelSDK(ctx)
	defer shutdown(ctx)
	if err != nil {
		panic(err)
	}

	s := Server{ASClients: make(map[uint32]*authority.Client), ActionChan: make(chan func())}
	go s.Initialize()

	handler := pserver2.WithMiddleware(
		s.handleConnection,
		pserver2.LoggingMiddleware,
	)

	log.Fatal(pserver2.ListenServe(ctx, handler, *portNumber))
}

type Server struct {
	ASClients  map[uint32]*authority.Client
	ActionChan chan func()
}

func (s *Server) Initialize() {
	for {
		f := <-s.ActionChan
		log.Printf("Getting/Creating client for AS\n")
		f()
	}
}

func (s *Server) GetClient(site uint32) (*authority.Client, error) {
	type result struct {
		c   *authority.Client
		err error
	}

	ch := make(chan result)

	s.ActionChan <- func() {
		client, ok := s.ASClients[site]
		if !ok {
			newclient, err := authority.NewClient(site)
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

func (s *Server) handleConnection(ctx context.Context, conn net.Conn) {
	clientId := newClientId()
	ctx, span := tracer.Start(ctx, "client", trace.WithAttributes(attribute.Int("client id", clientId)))
	defer span.End()

	br := bufio.NewReader(conn)
	log.Printf("New client with Id: %d\n", clientId)

	_, err := message.ReadHello(br)
	if err != nil {
		log.Printf("Error reading Hello message: %v\n", err)
		replyMsg := &message.Hello{Protocol: "pestcontrol", Version: 1}
		message.Write(conn, replyMsg)
		errMsg := &message.Error{Message: err.Error()}
		message.Write(conn, errMsg)
		conn.Close()
		return
	}
	log.Println("Received valid hello message")

	err = message.Write(conn, &message.ValidHello)
	if err != nil {
		log.Printf("failed sending message: %v\n", err)
		conn.Close()
		return
	}

	for {
		msg, err := message.ReadSiteVisit(br)
		if errors.Is(err, io.EOF) {
			log.Printf("EOF, disconecting client %d\n", clientId)
			conn.Close()
		} else if err != nil {
			log.Printf("Error reading SiteVisit: %v\n", err)
			errMsg := &message.Error{Message: err.Error()}
			message.Write(conn, errMsg)
			conn.Close()
			return
		}

		if err = message.VerifySiteVisit(msg); err != nil {
			log.Printf("Error veryfying site visit: %v\n", err)
			errMsg := &message.Error{Message: err.Error()}
			message.Write(conn, errMsg)
			conn.Close()
			return
		}

		log.Printf("SiteVisit message: %+v\n", msg)

		client, err := s.GetClient(msg.Site)
		if err != nil {
			log.Printf("Could not create client for site %d: %v\n", msg.Site, err)
			errMsg := &message.Error{Message: err.Error()}
			message.Write(conn, errMsg)
			conn.Close()
			return
		}

		log.Printf("client created for site %d\n", msg.Site)

		if err = client.AdjustPolicy(ctx, msg.Populations); err != nil {
			log.Printf("Error adjusting policy for site %d: %v\n", msg.Site, err)
			errMsg := &message.Error{Message: err.Error()}
			message.Write(conn, errMsg)
			conn.Close()
			return
		}

		log.Printf("Adjusted policy for site %d\n", msg.Site)
	}
}
