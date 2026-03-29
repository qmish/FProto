package transport

import (
	"context"
	"fmt"
	"io"
	"net"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
)

// GRPCTransportService defines the bidirectional streaming service for the
// transport layer.
type GRPCTransportService struct {
	UnimplementedProtoTransportServer
	handler func(Conn)
}

// ProtoTransportServer is a minimal interface for the bidirectional stream.
type ProtoTransportServer interface {
	Connect(stream grpc.BidiStreamingServer[RawMessage, RawMessage]) error
}

// UnimplementedProtoTransportServer is an empty implementation stub.
type UnimplementedProtoTransportServer struct{}

// RawMessage carries binary data over gRPC streaming.
type RawMessage struct {
	Data []byte
}

// NewGRPCTransportService creates a gRPC transport service with a connection handler.
func NewGRPCTransportService(handler func(Conn)) *GRPCTransportService {
	return &GRPCTransportService{handler: handler}
}

// GRPCStreamConn wraps a gRPC bidirectional stream as a Conn.
type GRPCStreamConn struct {
	ctx    context.Context
	cancel context.CancelFunc
	recvCh chan []byte
	sendCh chan []byte
	errCh  chan error
	addr   string
	once   sync.Once
	done   chan struct{}
}

// NewGRPCStreamConn creates a Conn from server-side gRPC stream operations.
func NewGRPCStreamConn(ctx context.Context, addr string) *GRPCStreamConn {
	ctx, cancel := context.WithCancel(ctx)
	return &GRPCStreamConn{
		ctx:    ctx,
		cancel: cancel,
		recvCh: make(chan []byte, 16),
		sendCh: make(chan []byte, 16),
		errCh:  make(chan error, 1),
		addr:   addr,
		done:   make(chan struct{}),
	}
}

func (g *GRPCStreamConn) ReadMessage() ([]byte, error) {
	select {
	case data := <-g.recvCh:
		return data, nil
	case <-g.ctx.Done():
		return nil, fmt.Errorf("grpc stream closed")
	}
}

func (g *GRPCStreamConn) WriteMessage(data []byte) error {
	select {
	case g.sendCh <- data:
		return nil
	case <-g.ctx.Done():
		return fmt.Errorf("grpc stream closed")
	}
}

func (g *GRPCStreamConn) Close() error {
	g.once.Do(func() {
		g.cancel()
		close(g.done)
	})
	return nil
}

func (g *GRPCStreamConn) Type() Type {
	return TypeGRPC
}

func (g *GRPCStreamConn) RemoteAddr() string {
	return g.addr
}

// Feed sends data into the receive channel (called by the stream pump).
func (g *GRPCStreamConn) Feed(data []byte) {
	select {
	case g.recvCh <- data:
	case <-g.ctx.Done():
	}
}

// Drain returns the next message to send (called by the stream pump).
func (g *GRPCStreamConn) Drain() ([]byte, error) {
	select {
	case data := <-g.sendCh:
		return data, nil
	case <-g.ctx.Done():
		return nil, io.EOF
	}
}

// Done returns a channel that is closed when the connection is closed.
func (g *GRPCStreamConn) Done() <-chan struct{} {
	return g.done
}

// GRPCClientConn wraps a client-side gRPC connection as a Conn.
type GRPCClientConn struct {
	sendCh chan []byte
	recvCh chan []byte
	ctx    context.Context
	cancel context.CancelFunc
	addr   string
	once   sync.Once
}

// NewGRPCClientConn creates a Conn from client-side gRPC stream operations.
func NewGRPCClientConn(ctx context.Context, addr string) *GRPCClientConn {
	ctx, cancel := context.WithCancel(ctx)
	return &GRPCClientConn{
		sendCh: make(chan []byte, 16),
		recvCh: make(chan []byte, 16),
		ctx:    ctx,
		cancel: cancel,
		addr:   addr,
	}
}

func (g *GRPCClientConn) ReadMessage() ([]byte, error) {
	select {
	case data := <-g.recvCh:
		return data, nil
	case <-g.ctx.Done():
		return nil, fmt.Errorf("grpc client closed")
	}
}

func (g *GRPCClientConn) WriteMessage(data []byte) error {
	select {
	case g.sendCh <- data:
		return nil
	case <-g.ctx.Done():
		return fmt.Errorf("grpc client closed")
	}
}

func (g *GRPCClientConn) Close() error {
	g.once.Do(func() { g.cancel() })
	return nil
}

func (g *GRPCClientConn) Type() Type        { return TypeGRPC }
func (g *GRPCClientConn) RemoteAddr() string { return g.addr }

// Feed sends data into the receive channel (called by the stream pump).
func (g *GRPCClientConn) Feed(data []byte) {
	select {
	case g.recvCh <- data:
	case <-g.ctx.Done():
	}
}

// Drain returns the next message to send (called by the stream pump).
func (g *GRPCClientConn) Drain() ([]byte, error) {
	select {
	case data := <-g.sendCh:
		return data, nil
	case <-g.ctx.Done():
		return nil, io.EOF
	}
}

// ExtractRemoteAddr gets the remote address from gRPC peer info or metadata.
func ExtractRemoteAddr(ctx context.Context) string {
	if p, ok := peer.FromContext(ctx); ok {
		return p.Addr.String()
	}
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if vals := md.Get("x-forwarded-for"); len(vals) > 0 {
			return vals[0]
		}
	}
	return "unknown"
}

// ListenGRPC starts a gRPC server on the given address.
func ListenGRPC(addr string) (net.Listener, *grpc.Server, error) {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, nil, fmt.Errorf("grpc listen: %w", err)
	}
	srv := grpc.NewServer(
		grpc.MaxRecvMsgSize(64*1024),
		grpc.MaxSendMsgSize(64*1024),
	)
	return lis, srv, nil
}
