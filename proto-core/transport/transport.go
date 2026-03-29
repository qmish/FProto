package transport

// Type identifies the transport protocol.
type Type string

const (
	TypeWebSocket Type = "websocket"
	TypeQUIC      Type = "quic"
	TypeGRPC      Type = "grpc"
)

// Conn abstracts a transport connection.
// Implementations: WebSocket, QUIC, gRPC streaming.
type Conn interface {
	ReadMessage() ([]byte, error)
	WriteMessage(data []byte) error
	Close() error
	Type() Type
	RemoteAddr() string
}
