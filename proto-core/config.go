package core

import (
	"time"

	"github.com/qmish/FProto/proto-core/session"
)

// Config holds the top-level configuration for the proto-core framework.
type Config struct {
	WebSocketAddr string
	QUICAddr      string
	GRPCAddr      string

	RedisAddr string
	PgDSN     string

	OTLPEndpoint   string
	ServiceName    string
	ServiceVersion string

	TLSCertFile string
	TLSKeyFile  string

	Session session.Config

	HandshakeTimeout time.Duration
	MaxPayloadSize   int
}

// DefaultConfig returns sensible defaults for proto-core configuration.
func DefaultConfig() Config {
	return Config{
		WebSocketAddr:    ":8080",
		QUICAddr:         ":8443",
		GRPCAddr:         ":9000",
		RedisAddr:        "localhost:6379",
		OTLPEndpoint:     "localhost:4317",
		ServiceName:      "proto-core",
		ServiceVersion:   "1.0.0",
		Session:          session.DefaultConfig(),
		HandshakeTimeout: 30 * time.Second,
		MaxPayloadSize:   65536,
	}
}
