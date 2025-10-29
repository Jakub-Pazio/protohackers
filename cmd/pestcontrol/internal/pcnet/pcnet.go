package pcnet

import (
	"bean/cmd/pestcontrol/internal/message"
	"bufio"
	"context"
	"fmt"
	"net"
)

type Conn struct {
	conn net.Conn
	br   *bufio.Reader
}

func NewConn(conn net.Conn) *Conn {
	br := bufio.NewReader(conn)
	return &Conn{
		conn: conn,
		br:   br,
	}
}

func (t *Conn) Close() error {
	t.br = nil
	return t.conn.Close()
}

// TODO: all timeouts can be implemented on this method, because it's used with all other methods
func (t *Conn) Read(ctx context.Context) (message.Message, error) {
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

func (t *Conn) ReadHello(ctx context.Context) (*message.Hello, error) {
	m, err := t.Read(ctx)
	if err != nil {
		return nil, err
	}
	h, ok := m.(*message.Hello)
	if !ok {
		return nil, message.ErrInvalidMessageType
	}
	if err = message.ValidateHello(*h); err != nil {
		return nil, fmt.Errorf("validate hello: %w", err)
	}
	return h, nil
}

func (t *Conn) ReadSiteVisit(ctx context.Context) (*message.SiteVisit, error) {
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
func (t *Conn) ReadPolicyResult(ctx context.Context) (*message.PolicyResult, error) {
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
func (t *Conn) ReadOK(ctx context.Context) (*message.OK, error) {
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

func (t *Conn) ReadTargetPopulation(ctx context.Context) (*message.TargetPopulation, error) {
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

func (t *Conn) RemoteAddr() net.Addr {
	return t.conn.RemoteAddr()
}

func (t *Conn) Write(ctx context.Context, msg message.Message) error {
	_, err := t.conn.Write(message.Serialize(msg))
	return err
}

func (t *Conn) WriteError(ctx context.Context, err error) error {
	msg := &message.Error{Message: err.Error()}
	return t.Write(ctx, msg)
}
