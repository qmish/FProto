package session

import (
	"time"
)

type Status string

const (
	StatusConnecting Status = "connecting"
	StatusActive     Status = "active"
	StatusSleeping   Status = "sleeping"
	StatusExpired    Status = "expired"
)

type TransportType string

const (
	TransportWebSocket TransportType = "websocket"
	TransportQUIC      TransportType = "quic"
	TransportGRPC      TransportType = "grpc"
)

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

type Connection struct {
	ConnectionID  string
	Transport     TransportType
	GatewayID     string
	RemoteAddr    string
	ConnectedAt   time.Time
	LastPacketSeq uint64
}

type Config struct {
	IdleTimeout   time.Duration // Active -> Sleeping (default 5m)
	ExpiryTimeout time.Duration // Sleeping -> Expired (default 24h)
	HandshakeTimeout time.Duration // Connecting -> Destroyed (default 30s)
	MaxConnections   int           // Max concurrent connections per session (default 5)
}

func DefaultConfig() Config {
	return Config{
		IdleTimeout:      5 * time.Minute,
		ExpiryTimeout:    24 * time.Hour,
		HandshakeTimeout: 30 * time.Second,
		MaxConnections:   5,
	}
}
