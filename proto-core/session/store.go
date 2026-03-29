package session

import "context"

// Store is the persistence interface for session state.
// Implementations: RedisStore (hot path), PGStore (recovery/durability).
type Store interface {
	Create(ctx context.Context, s *Session) error
	Get(ctx context.Context, sessionID []byte) (*Session, error)
	UpdateStatus(ctx context.Context, sessionID []byte, status Status) error
	UpdateCryptoState(ctx context.Context, sessionID []byte, cryptoState []byte) error
	UpdateLastActive(ctx context.Context, sessionID []byte) error
	Delete(ctx context.Context, sessionID []byte) error

	AddConnection(ctx context.Context, sessionID []byte, conn *Connection) error
	RemoveConnection(ctx context.Context, sessionID []byte, connectionID string) error
	GetConnections(ctx context.Context, sessionID []byte) ([]Connection, error)

	GetUserSessions(ctx context.Context, userID []byte) ([][]byte, error)
}
