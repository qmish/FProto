package transport

// Conn абстрагирует транспортное соединение.
// Реализации: WebSocket, QUIC (будущее), gRPC (будущее).
type Conn interface {
	// ReadMessage читает следующее бинарное сообщение из соединения.
	ReadMessage() ([]byte, error)

	// WriteMessage отправляет бинарное сообщение в соединение.
	WriteMessage(data []byte) error

	// Close закрывает соединение.
	Close() error
}
