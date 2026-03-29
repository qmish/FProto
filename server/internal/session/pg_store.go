package session

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const createTableSQL = `
CREATE TABLE IF NOT EXISTS sessions (
    session_id  BYTEA PRIMARY KEY,
    user_id     BYTEA NOT NULL,
    device_id   BYTEA NOT NULL,
    crypto_state BYTEA NOT NULL,
    status      VARCHAR(20) NOT NULL DEFAULT 'connecting'
                CHECK (status IN ('connecting', 'active', 'sleeping', 'expired')),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at  TIMESTAMPTZ NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_sessions_device_id ON sessions(device_id);
CREATE INDEX IF NOT EXISTS idx_sessions_expires_at ON sessions(expires_at) WHERE status != 'expired';
`

// PGStore implements Store for durable session persistence (recovery path).
type PGStore struct {
	pool *pgxpool.Pool
}

func NewPGStore(pool *pgxpool.Pool) *PGStore {
	return &PGStore{pool: pool}
}

func (p *PGStore) Migrate(ctx context.Context) error {
	_, err := p.pool.Exec(ctx, createTableSQL)
	return err
}

func (p *PGStore) Create(ctx context.Context, s *Session) error {
	_, err := p.pool.Exec(ctx,
		`INSERT INTO sessions (session_id, user_id, device_id, crypto_state, status, created_at, updated_at, expires_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		s.SessionID, s.UserID, s.DeviceID, s.CryptoState,
		string(s.Status), s.CreatedAt, s.CreatedAt, s.ExpiresAt,
	)
	return err
}

func (p *PGStore) Get(ctx context.Context, sessionID []byte) (*Session, error) {
	row := p.pool.QueryRow(ctx,
		`SELECT session_id, user_id, device_id, crypto_state, status, created_at, updated_at, expires_at
		 FROM sessions WHERE session_id = $1`, sessionID)

	s := &Session{}
	var status string
	var updatedAt time.Time
	err := row.Scan(&s.SessionID, &s.UserID, &s.DeviceID, &s.CryptoState,
		&status, &s.CreatedAt, &updatedAt, &s.ExpiresAt)
	if err != nil {
		return nil, fmt.Errorf("session not found: %w", err)
	}
	s.Status = Status(status)
	s.LastActive = updatedAt
	return s, nil
}

func (p *PGStore) UpdateStatus(ctx context.Context, sessionID []byte, status Status) error {
	_, err := p.pool.Exec(ctx,
		`UPDATE sessions SET status = $2, updated_at = NOW() WHERE session_id = $1`,
		sessionID, string(status))
	return err
}

func (p *PGStore) UpdateCryptoState(ctx context.Context, sessionID []byte, cryptoState []byte) error {
	_, err := p.pool.Exec(ctx,
		`UPDATE sessions SET crypto_state = $2, updated_at = NOW() WHERE session_id = $1`,
		sessionID, cryptoState)
	return err
}

func (p *PGStore) UpdateLastActive(ctx context.Context, sessionID []byte) error {
	_, err := p.pool.Exec(ctx,
		`UPDATE sessions SET updated_at = NOW() WHERE session_id = $1`, sessionID)
	return err
}

func (p *PGStore) Delete(ctx context.Context, sessionID []byte) error {
	_, err := p.pool.Exec(ctx, `DELETE FROM sessions WHERE session_id = $1`, sessionID)
	return err
}

func (p *PGStore) AddConnection(_ context.Context, _ []byte, _ *Connection) error {
	return nil
}

func (p *PGStore) RemoveConnection(_ context.Context, _ []byte, _ string) error {
	return nil
}

func (p *PGStore) GetConnections(_ context.Context, _ []byte) ([]Connection, error) {
	return nil, nil
}

func (p *PGStore) GetUserSessions(ctx context.Context, userID []byte) ([][]byte, error) {
	rows, err := p.pool.Query(ctx,
		`SELECT session_id FROM sessions WHERE user_id = $1 AND status != 'expired'`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result [][]byte
	for rows.Next() {
		var id []byte
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		result = append(result, id)
	}
	return result, rows.Err()
}
