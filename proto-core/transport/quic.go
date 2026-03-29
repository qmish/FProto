package transport

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/binary"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"net"
	"time"

	"github.com/quic-go/quic-go"
)

const (
	quicALPN        = "proto/1"
	quicMaxMsgSize  = 65536
	quicIdleTimeout = 120 * time.Second
)

// QUICConn wraps a QUIC stream as a Conn.
type QUICConn struct {
	conn   *quic.Conn
	stream *quic.Stream
	addr   string
}

// ListenQUIC starts a QUIC listener on the given address with the provided TLS config.
func ListenQUIC(addr string, tlsConf *tls.Config) (*quic.Listener, error) {
	tlsCopy := tlsConf.Clone()
	tlsCopy.NextProtos = []string{quicALPN}
	listener, err := quic.ListenAddr(addr, tlsCopy, &quic.Config{
		MaxIdleTimeout: quicIdleTimeout,
	})
	if err != nil {
		return nil, fmt.Errorf("quic listen: %w", err)
	}
	return listener, nil
}

// AcceptQUIC accepts a QUIC connection and opens the control stream.
func AcceptQUIC(ctx context.Context, listener *quic.Listener) (*QUICConn, error) {
	conn, err := listener.Accept(ctx)
	if err != nil {
		return nil, fmt.Errorf("quic accept: %w", err)
	}

	stream, err := conn.AcceptStream(ctx)
	if err != nil {
		return nil, fmt.Errorf("quic accept stream: %w", err)
	}

	return &QUICConn{
		conn:   conn,
		stream: stream,
		addr:   conn.RemoteAddr().String(),
	}, nil
}

// LocalAddr returns the local address of the QUIC connection.
func (q *QUICConn) LocalAddr() net.Addr {
	return q.conn.LocalAddr()
}

// DialQUIC connects to a QUIC server.
func DialQUIC(ctx context.Context, addr string, tlsConf *tls.Config) (*QUICConn, error) {
	tlsCopy := tlsConf.Clone()
	tlsCopy.NextProtos = []string{quicALPN}
	conn, err := quic.DialAddr(ctx, addr, tlsCopy, &quic.Config{
		MaxIdleTimeout: quicIdleTimeout,
	})
	if err != nil {
		return nil, fmt.Errorf("quic dial: %w", err)
	}

	stream, err := conn.OpenStreamSync(ctx)
	if err != nil {
		return nil, fmt.Errorf("quic open stream: %w", err)
	}

	return &QUICConn{
		conn:   conn,
		stream: stream,
		addr:   conn.RemoteAddr().String(),
	}, nil
}

func (q *QUICConn) ReadMessage() ([]byte, error) {
	var lenBuf [4]byte
	if _, err := io.ReadFull(q.stream, lenBuf[:]); err != nil {
		return nil, fmt.Errorf("quic read length: %w", err)
	}
	msgLen := binary.BigEndian.Uint32(lenBuf[:])
	if msgLen > quicMaxMsgSize {
		return nil, fmt.Errorf("quic message too large: %d", msgLen)
	}
	buf := make([]byte, msgLen)
	if _, err := io.ReadFull(q.stream, buf); err != nil {
		return nil, fmt.Errorf("quic read payload: %w", err)
	}
	return buf, nil
}

func (q *QUICConn) WriteMessage(data []byte) error {
	var lenBuf [4]byte
	binary.BigEndian.PutUint32(lenBuf[:], uint32(len(data)))
	if _, err := q.stream.Write(lenBuf[:]); err != nil {
		return fmt.Errorf("quic write length: %w", err)
	}
	if _, err := q.stream.Write(data); err != nil {
		return fmt.Errorf("quic write payload: %w", err)
	}
	return nil
}

func (q *QUICConn) Close() error {
	_ = q.stream.Close()
	return q.conn.CloseWithError(0, "close")
}

func (q *QUICConn) Type() Type {
	return TypeQUIC
}

func (q *QUICConn) RemoteAddr() string {
	return q.addr
}

// GenerateSelfSignedTLS creates a self-signed TLS config for testing.
func GenerateSelfSignedTLS() *tls.Config {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		panic(err)
	}
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(24 * time.Hour),
		IPAddresses:  []net.IP{net.IPv4(127, 0, 0, 1)},
		DNSNames:     []string{"localhost"},
	}
	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		panic(err)
	}
	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		panic(err)
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})

	tlsCert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		panic(err)
	}
	return &tls.Config{
		Certificates: []tls.Certificate{tlsCert},
	}
}
