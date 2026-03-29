package sdk

import (
	"context"
	"crypto/tls"
	"fmt"

	"github.com/gorilla/websocket"
	"github.com/quic-go/quic-go"

	"encoding/binary"
	"io"
)

// Transport abstracts the network connection.
type Transport interface {
	ReadMessage() ([]byte, error)
	WriteMessage(data []byte) error
	Close() error
}

func dial(ctx context.Context, cfg Config) (Transport, error) {
	switch cfg.TransportType {
	case "websocket", "ws", "":
		return dialWS(cfg.ServerAddr)
	case "quic":
		return dialQUIC(ctx, cfg.ServerAddr, cfg.TLSSkipVerify)
	default:
		return nil, fmt.Errorf("unknown transport: %s", cfg.TransportType)
	}
}

// --- WebSocket ---

type wsTransport struct {
	conn *websocket.Conn
}

func dialWS(url string) (*wsTransport, error) {
	c, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		return nil, err
	}
	return &wsTransport{conn: c}, nil
}

func (w *wsTransport) ReadMessage() ([]byte, error) {
	_, data, err := w.conn.ReadMessage()
	return data, err
}

func (w *wsTransport) WriteMessage(data []byte) error {
	return w.conn.WriteMessage(websocket.BinaryMessage, data)
}

func (w *wsTransport) Close() error {
	return w.conn.Close()
}

// --- QUIC ---

type quicTransport struct {
	conn   *quic.Conn
	stream *quic.Stream
}

func dialQUIC(ctx context.Context, addr string, insecure bool) (*quicTransport, error) {
	tlsConf := &tls.Config{
		NextProtos:         []string{"proto/1"},
		InsecureSkipVerify: insecure,
	}
	conn, err := quic.DialAddr(ctx, addr, tlsConf, &quic.Config{})
	if err != nil {
		return nil, err
	}
	stream, err := conn.OpenStreamSync(ctx)
	if err != nil {
		return nil, err
	}
	return &quicTransport{conn: conn, stream: stream}, nil
}

func (q *quicTransport) ReadMessage() ([]byte, error) {
	var lenBuf [4]byte
	if _, err := io.ReadFull(q.stream, lenBuf[:]); err != nil {
		return nil, err
	}
	msgLen := binary.BigEndian.Uint32(lenBuf[:])
	buf := make([]byte, msgLen)
	if _, err := io.ReadFull(q.stream, buf); err != nil {
		return nil, err
	}
	return buf, nil
}

func (q *quicTransport) WriteMessage(data []byte) error {
	var lenBuf [4]byte
	binary.BigEndian.PutUint32(lenBuf[:], uint32(len(data)))
	if _, err := q.stream.Write(lenBuf[:]); err != nil {
		return err
	}
	_, err := q.stream.Write(data)
	return err
}

func (q *quicTransport) Close() error {
	_ = q.stream.Close()
	return q.conn.CloseWithError(0, "close")
}
