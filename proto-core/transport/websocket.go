package transport

import (
	"fmt"
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  65536,
	WriteBufferSize: 65536,
	CheckOrigin:     func(r *http.Request) bool { return true },
	Subprotocols:    []string{"proto.v1"},
}

// WSConn implements Conn over gorilla/websocket.
type WSConn struct {
	conn *websocket.Conn
}

// UpgradeHTTP performs WebSocket upgrade for an HTTP request.
func UpgradeHTTP(w http.ResponseWriter, r *http.Request) (*WSConn, error) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return nil, fmt.Errorf("websocket upgrade: %w", err)
	}
	return &WSConn{conn: c}, nil
}

// DialWS connects to a WebSocket server (client-side).
func DialWS(url string) (*WSConn, error) {
	c, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		return nil, fmt.Errorf("websocket dial: %w", err)
	}
	return &WSConn{conn: c}, nil
}

func (w *WSConn) ReadMessage() ([]byte, error) {
	_, data, err := w.conn.ReadMessage()
	if err != nil {
		return nil, fmt.Errorf("ws read: %w", err)
	}
	return data, nil
}

func (w *WSConn) WriteMessage(data []byte) error {
	if err := w.conn.WriteMessage(websocket.BinaryMessage, data); err != nil {
		return fmt.Errorf("ws write: %w", err)
	}
	return nil
}

func (w *WSConn) Close() error {
	return w.conn.Close()
}

func (w *WSConn) Type() Type {
	return TypeWebSocket
}

func (w *WSConn) RemoteAddr() string {
	return w.conn.RemoteAddr().String()
}
