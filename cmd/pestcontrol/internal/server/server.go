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
	"sync/atomic"

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

var idGenClient atomic.Int32

func newClientId() int {
	return int(idGenClient.Add(1))
}

type Server struct {
	ASClients  map[uint32]*authority.Client
	ActionChan chan func()
}

func New() *Server {
	return &Server{
		ASClients:  make(map[uint32]*authority.Client),
		ActionChan: make(chan func()),
	}
}

// Initialize starts server actor loop. Any public calls are effectivelly serial,
// making access to server safe from multiple goroutines
func (s *Server) Initialize() {
	for {
		f := <-s.ActionChan
		f()
	}
}

func (s *Server) getClient(ctx context.Context, site uint32) (*authority.Client, error) {
	type result struct {
		c   *authority.Client
		err error
	}

	ch := make(chan result)

	s.ActionChan <- func() {
		client, ok := s.ASClients[site]
		if !ok {
			asAddress := net.JoinHostPort(ASDomain, ASPort)
			conn, err := net.Dial("tcp", asAddress)
			pconn, err := pcnet.NewConn(conn)
			if err != nil {
				ch <- result{nil, fmt.Errorf("new conn: %w", err)}
				return
			}
			newclient, err := authority.NewClient(ctx, site, pconn)
			if err != nil {
				ch <- result{nil, err}
				return
			}
			s.ASClients[site] = newclient
			client = newclient
		}

		ch <- result{client, nil}
	}

	res := <-ch
	return res.c, res.err
}

func (s *Server) HandleConnection(ctx context.Context, conn net.Conn) {
	clientId := newClientId()
	clientAddress := conn.RemoteAddr().String()
	pconn, err := pcnet.NewConn(conn)
	if err != nil {
		//TODO: fix
		panic(err)
	}

	ctx, span := tracer.Start(
		ctx,
		"client-connection",
		trace.WithAttributes(attribute.Int("client-id", clientId)),
		trace.WithAttributes(attribute.String("client-address", clientAddress)),
	)

	defer func() {
		span.End()
		conn.Close()
	}()

	logger.InfoContext(ctx, "New client", "id", clientId)

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

	logger.InfoContext(ctx, "Wrote hello")

	for {
		msg, err := pconn.ReadSiteVisit(ctx)
		if errors.Is(err, io.EOF) {
			logger.Info("EOF, disconecting client", "client-id", clientId)
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
			// log.Printf("Could not create client for site %d: %v\n", msg.Site, err)
			errMsg := &message.Error{Message: err.Error()}
			message.Write(conn, errMsg)
			return
		}

		// log.Printf("client created for site %d\n", msg.Site)

		if err = client.AdjustPolicy(ctx, msg.Populations); err != nil {
			// log.Printf("Error adjusting policy for site %d: %v\n", msg.Site, err)
			errMsg := &message.Error{Message: err.Error()}
			message.Write(conn, errMsg)
			return
		}

		// log.Printf("Adjusted policy for site %d\n", msg.Site)
	}
}
