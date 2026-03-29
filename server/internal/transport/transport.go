package transport

// Type identifies the transport protocol.
type Type string

const (
	TypeWebSocket Type = "websocket"
	TypeQUIC      Type = "quic"
	TypeGRPC      Type = "grpc"
)

// Conn абстрагирует транспортное соединение.
// Реализации: WebSocket, QUIC, gRPC streaming.
type Conn interface {
	// ReadMessage читает следующее бинарное сообщение из соединения.
	ReadMessage() ([]byte, error)

	// WriteMessage отправляет бинарное сообщение в соединение.
	WriteMessage(data []byte) error

	// Close закрывает соединение.
	Close() error

	// Type возвращает тип транспорта.
	Type() Type

	// RemoteAddr возвращает адрес удалённой стороны.
	RemoteAddr() string
}
