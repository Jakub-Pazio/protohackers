package pcnet

import (
	"bean/cmd/pestcontrol/internal/message"
	"bufio"
	"context"
	"fmt"
	"net"
)

const (
	ASDomain = "pestcontrol.protohackers.com"
	ASPort   = "20547"
)

type Conn interface {
	Read(ctx context.Context) (message.Message, error)
	ReadHello(ctx context.Context) (*message.Hello, error)
	ReadOK(ctx context.Context) (*message.OK, error)
	ReadSiteVisit(ctx context.Context) (*message.SiteVisit, error)
	ReadPolicyResult(ctx context.Context) (*message.PolicyResult, error)
	ReadTargetPopulation(ctx context.Context) (*message.TargetPopulation, error)
	Write(ctx context.Context, msg message.Message) error
	WriteError(ctx context.Context, err error) error
	Close() error
	RemoteAddr() net.Addr
}

var _ Conn = (*tcpConn)(nil)

type tcpConn struct {
	conn net.Conn
	br   *bufio.Reader
}

func NewConn() (Conn, error) {
	asAddress := net.JoinHostPort(ASDomain, ASPort)
	conn, err := net.Dial("tcp", asAddress)
	if err != nil {
		return nil, fmt.Errorf("dial %q: %w", asAddress, err)
	}
	br := bufio.NewReader(conn)
	return &tcpConn{
		conn: conn,
		br:   br,
	}, nil
}

func (t *tcpConn) Close() error {
	t.br = nil
	return t.conn.Close()
}

// TODO: all timeouts can be implemented on this method, because it's used with all other methods
func (t *tcpConn) Read(ctx context.Context) (message.Message, error) {
	mtype, err := message.ReadMessageType(t.br)
	if err != nil {
		return nil, fmt.Errorf("read message type: %w", err)
	}
	mlen, err := message.ReadMessageLength(t.br)
	if err != nil {
		return nil, fmt.Errorf("read message length: %w", err)
	}
	body, err := message.ReadBody(t.br, mlen)
	if err != nil {
		return nil, fmt.Errorf("read remaining: %w", err)
	}

	switch mtype {
	case message.TypeHello:
		return message.ParseHello(mlen, body)
	case message.TypeError:
		return message.ParseHello(mlen, body)
	case message.TypeOK:
		return message.ParseOk(mlen, body)
	case message.TypePolicyResult:
		return message.ParsePolicyResult(mlen, body)
	case message.TypeSiteVisit:
		return message.ParseSiteVisit(mlen, body)
	case message.TypeTargetPopulations:
		return message.ParseTargetPopulations(mlen, body)
	case message.TypeCreatePolicy, message.TypeDeletePolicy, message.TypeDialAuthority:
		return nil, message.ErrReadUnsupported
	default:
		return nil, message.ErrInvalidMessageType
	}
}

func (t *tcpConn) ReadHello(ctx context.Context) (*message.Hello, error) {
	m, err := t.Read(ctx)
	if err != nil {
		return nil, err
	}
	h, ok := m.(*message.Hello)
	if !ok {
		return nil, message.ErrInvalidMessageType
	}
	return h, nil
}

func (t *tcpConn) ReadSiteVisit(ctx context.Context) (*message.SiteVisit, error) {
	m, err := t.Read(ctx)
	if err != nil {
		return nil, err
	}
	h, ok := m.(*message.SiteVisit)
	if !ok {
		return nil, message.ErrInvalidMessageType
	}
	return h, nil
}

// ReadPolicyResult implements Conn.
func (t *tcpConn) ReadPolicyResult(ctx context.Context) (*message.PolicyResult, error) {
	m, err := t.Read(ctx)
	if err != nil {
		return nil, err
	}
	h, ok := m.(*message.PolicyResult)
	if !ok {
		return nil, message.ErrInvalidMessageType
	}
	return h, nil
}

// ReadOk implements Conn.
func (t *tcpConn) ReadOK(ctx context.Context) (*message.OK, error) {
	m, err := t.Read(ctx)
	if err != nil {
		return nil, err
	}
	h, ok := m.(*message.OK)
	if !ok {
		return nil, message.ErrInvalidMessageType
	}
	return h, nil
}

func (t *tcpConn) ReadTargetPopulation(ctx context.Context) (*message.TargetPopulation, error) {
	m, err := t.Read(ctx)
	if err != nil {
		return nil, err
	}
	h, ok := m.(*message.TargetPopulation)
	if !ok {
		return nil, message.ErrInvalidMessageType
	}
	return h, nil
}

func (t *tcpConn) RemoteAddr() net.Addr {
	return t.conn.RemoteAddr()
}

func (t *tcpConn) Write(ctx context.Context, msg message.Message) error {
	_, err := t.conn.Write(message.Serialize(msg))
	return err
}

func (t *tcpConn) WriteError(ctx context.Context, err error) error {
	msg := &message.Error{Message: err.Error()}
	return t.Write(ctx, msg)
}
