// Package core provides a transport-agnostic, cryptographically secure
// protocol framework for building real-time messaging systems.
//
// Key components:
//   - transport: Multi-transport connections (WebSocket, QUIC, gRPC)
//   - session: Session lifecycle management with Redis/PostgreSQL storage
//   - crypto: Noise_XX_25519_ChaChaPoly_BLAKE2s handshake + AEAD
//   - observability: OpenTelemetry tracing, Prometheus metrics, zap logging
//
// Usage:
//
//	cfg := core.DefaultConfig()
//	cfg.WebSocketAddr = ":8080"
//	cfg.QUICAddr = ":8443"
//
//	// Use individual packages directly:
//	conn, _ := transport.DialWS("ws://localhost:8080/ws")
//	session := crypto.ClientHandshake(key, conn.ReadMessage, conn.WriteMessage)
package core
