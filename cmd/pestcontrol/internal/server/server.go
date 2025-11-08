package server

import (
	"bean/cmd/pestcontrol/internal/authority"
	"bean/cmd/pestcontrol/internal/message"
	"bean/cmd/pestcontrol/internal/pcnet"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

const name = "jakubpazio.site/protohackers/server"

const (
	ASDomain = "pestcontrol.protohackers.com"
	ASPort   = "20547"
)

var (
	tracer = otel.Tracer(name)
	logger = otelslog.NewLogger(name)
)

var genClientID atomic.Int32

func newClientID() int {
	return int(genClientID.Add(1))
}

type Server struct {
	asClients    map[uint32]*authority.Client
	actionChan   chan func()
	clientWg     sync.WaitGroup
	serverWg     sync.WaitGroup
	shuttingDown bool
	DeadChan     chan error
}

func New() *Server {
	return &Server{
		asClients:  make(map[uint32]*authority.Client),
		actionChan: make(chan func()),
		DeadChan:   make(chan error),
	}
}

// Initialize starts server actor loop. Any public calls are effectivelly serial,
// making access to server safe from multiple goroutines
func (s *Server) Initialize(ctx context.Context) {
	for {
		select {
		case f := <-s.actionChan:
			f()
		case <-ctx.Done():
			// We don't want to start any new connections, wait for current to end
			// for some resonable time, them shutdown all AS clients, and return
			s.shuttingDown = true
			ch := s.shutdown(ctx)
			select {
			case err := <-ch:
				s.DeadChan <- err
			case <-time.After(time.Second * 15):
				s.DeadChan <- errors.New("failed to close server on time")
			}
			return
		}
	}
}

// We start the process of shuting down the server, first we wait some time for clients to close connections,
// then we close all outgoing AS connections
func (s *Server) shutdown(ctx context.Context) chan error {
	chErr := make(chan error)

	go func() {
		var (
			err error
			cCh = s.waitForClients(ctx)
		)

		select {
		case <-cCh:
			// try to close outgoing calls
		case <-time.After(time.Second * 10):
			errors.Join(err, errors.New("not all client shutdown in time"))
		}

		sCh := s.waitForSevers()
		select {
		case <-sCh:
			// we closed both clients and server, we confirm shutdown and return
		case <-time.After(time.Second * 5):
			chErr <- errors.New("could not shutdown all HA connections")
		}

		chErr <- err
	}()

	return chErr
}

func (s *Server) waitForClients(ctx context.Context) chan struct{} {
	ch := make(chan struct{})
	go func() {
		logger.InfoContext(ctx, "Started waiting for clients to finnish")
		// We simply wait for clients to end their tasks
		s.clientWg.Wait()
		logger.InfoContext(ctx, "All clients disconnected")
		ch <- struct{}{}
	}()
	return ch
}

func (s *Server) waitForSevers() chan struct{} {
	ch := make(chan struct{})
	go func() {
		for _, c := range s.asClients {
			c.Shutdown()
		}
		s.serverWg.Wait()
		ch <- struct{}{}
	}()
	return ch
}

// getClient returns already created client for certain site, or bootstaps new connection,
// sends all necesary messages and returns ready to use client
func (s *Server) getClient(ctx context.Context, site uint32) (*authority.Client, error) {
	type result struct {
		c   *authority.Client
		err error
	}

	ch := make(chan result)

	s.actionChan <- func() {
		client, ok := s.asClients[site]
		if !ok {
			asAddress := net.JoinHostPort(ASDomain, ASPort)
			conn, err := net.Dial("tcp", asAddress)
			if err != nil {
				ch <- result{nil, fmt.Errorf("new conn: %w", err)}
				return
			}
			pconn := pcnet.NewConn(conn)
			newclient, err := authority.NewClient(ctx, site, pconn)
			if err != nil {
				ch <- result{nil, fmt.Errorf("new client: %w", err)}
				return
			}
			s.asClients[site] = newclient
			s.serverWg.Add(1)
			client = newclient
		}

		ch <- result{client, nil}
	}

	res := <-ch
	return res.c, res.err
}

func (s *Server) HandleConnection(ctx context.Context, conn net.Conn) {
	defer conn.Close()
	return
	if s.shuttingDown {
		logger.WarnContext(ctx, "connection refused due to shutdown",
			"address", conn.RemoteAddr().String(),
		)
		return
	}

	s.clientWg.Add(1)
	defer s.clientWg.Done()

	var (
		clientID      = newClientID()
		clientAddress = conn.RemoteAddr().String()
		pconn         = pcnet.NewConn(conn)
	)

	ctx, span := tracer.Start(
		ctx,
		"client-connection",
		trace.WithAttributes(attribute.Int("client-id", clientID)),
		trace.WithAttributes(attribute.String("client-address", clientAddress)),
	)
	defer span.End()

	logger.InfoContext(ctx, "New client", "id", clientID)

	if _, err := pconn.ReadHello(ctx); err != nil {
		logger.WarnContext(ctx, "Reading hello failed", "error", err)
		pconn.Write(ctx, message.ValidHello)
		pconn.WriteError(ctx, err)
		return
	}

	logger.InfoContext(ctx, "Received hello")

	if err := pconn.Write(ctx, message.ValidHello); err != nil {
		logger.WarnContext(ctx, "Sending message failed", "error", err)
		return
	}

	logger.InfoContext(ctx, "Wrote hello", "client", clientID)

	for {
		msg, err := pconn.ReadSiteVisit(ctx)
		if errors.Is(err, io.EOF) {
			logger.Info("EOF, disconecting client", "client-id", clientID)
			break
		} else if err != nil {
			logger.WarnContext(ctx, "Error reading SiteVisit", "error", err)
			pconn.WriteError(ctx, err)
			return
		}

		if err = message.VerifySiteVisit(msg); err != nil {
			logger.WarnContext(ctx, "Failed verifying site", "error", err)
			pconn.WriteError(ctx, err)
			return
		}

		client, err := s.getClient(ctx, msg.Site)
		if err != nil {
			logger.WarnContext(ctx, "Failed getting client",
				"site", msg.Site,
				"error", err,
			)
			pconn.WriteError(ctx, err)
			return
		}

		if err = client.AdjustPolicy(ctx, msg.Populations); err != nil {
			logger.WarnContext(ctx, "Failed adjusting policy",
				"site", msg.Site,
				"error", err,
			)
			pconn.WriteError(ctx, err)
			return
		}
	}
}
