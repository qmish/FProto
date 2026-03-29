package sdk

import (
	"context"
	"fmt"
	"sync"

	"github.com/flynn/noise"
)

// Client is the reference FProto client SDK for Go.
type Client struct {
	mu       sync.Mutex
	conn     Transport
	session  *noiseSession
	key      *KeyPair
	config   Config
	closed   bool
}

// Config holds client configuration.
type Config struct {
	ServerAddr      string
	TransportType   string // "websocket", "quic"
	TLSSkipVerify   bool
	Reconnect       ReconnectConfig
}

// New creates a new FProto client.
func New(key *KeyPair, config Config) *Client {
	return &Client{
		key:    key,
		config: config,
	}
}

// Connect establishes a transport connection and performs Noise handshake.
func (c *Client) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	conn, err := dial(ctx, c.config)
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}
	c.conn = conn

	session, err := clientHandshake(c.key, conn)
	if err != nil {
		conn.Close()
		return fmt.Errorf("handshake: %w", err)
	}
	c.session = session
	c.closed = false
	return nil
}

// Send encrypts and sends a message.
func (c *Client) Send(plaintext []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed || c.conn == nil {
		return fmt.Errorf("client not connected")
	}

	ct, err := c.session.Encrypt(plaintext)
	if err != nil {
		return err
	}
	return c.conn.WriteMessage(ct)
}

// Receive reads and decrypts a message.
func (c *Client) Receive() ([]byte, error) {
	c.mu.Lock()
	conn := c.conn
	session := c.session
	c.mu.Unlock()

	if conn == nil {
		return nil, fmt.Errorf("client not connected")
	}

	ct, err := conn.ReadMessage()
	if err != nil {
		return nil, err
	}

	return session.Decrypt(ct)
}

// Close shuts down the client connection.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.closed = true
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

type noiseSession struct {
	send *noise.CipherState
	recv *noise.CipherState
}

func (s *noiseSession) Encrypt(plaintext []byte) ([]byte, error) {
	return s.send.Encrypt(nil, nil, plaintext)
}

func (s *noiseSession) Decrypt(ciphertext []byte) ([]byte, error) {
	return s.recv.Decrypt(nil, nil, ciphertext)
}
