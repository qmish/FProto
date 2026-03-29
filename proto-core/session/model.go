package session

import (
	"time"
)

// Status represents the session lifecycle state.
type Status string

const (
	StatusConnecting Status = "connecting"
	StatusActive     Status = "active"
	StatusSleeping   Status = "sleeping"
	StatusExpired    Status = "expired"
)

// TransportType identifies the transport used for a connection.
type TransportType string

const (
	TransportWebSocket TransportType = "websocket"
	TransportQUIC      TransportType = "quic"
	TransportGRPC      TransportType = "grpc"
)

// Session holds the full session state.
type Session struct {
	SessionID   []byte
	UserID      []byte
	DeviceID    []byte
	Status      Status
	CryptoState []byte
	CreatedAt   time.Time
	LastActive  time.Time
	ExpiresAt   time.Time
	Connections []Connection
}

// Connection represents a single transport connection within a session.
type Connection struct {
	ConnectionID  string
	Transport     TransportType
	GatewayID     string
	RemoteAddr    string
	ConnectedAt   time.Time
	LastPacketSeq uint64
}

// Config holds session lifecycle timeouts and limits.
type Config struct {
	IdleTimeout      time.Duration
	ExpiryTimeout    time.Duration
	HandshakeTimeout time.Duration
	MaxConnections   int
}

// DefaultConfig returns sensible defaults for session configuration.
func DefaultConfig() Config {
	return Config{
		IdleTimeout:      5 * time.Minute,
		ExpiryTimeout:    24 * time.Hour,
		HandshakeTimeout: 30 * time.Second,
		MaxConnections:   5,
	}
}
