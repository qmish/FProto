package inbox

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

const migrateSQL = `
CREATE TABLE IF NOT EXISTS inbox (
    id              BIGSERIAL PRIMARY KEY,
    user_id         BYTEA NOT NULL,
    message_id      BYTEA NOT NULL,
    conversation_id BYTEA,
    sender_id       BYTEA NOT NULL,
    payload         BYTEA NOT NULL,
    delivered       BOOLEAN NOT NULL DEFAULT FALSE,
    read            BOOLEAN NOT NULL DEFAULT FALSE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, message_id)
);
CREATE INDEX IF NOT EXISTS idx_inbox_conv ON inbox(user_id, conversation_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_inbox_undelivered ON inbox(user_id, created_at) WHERE delivered = FALSE;
`

// Store manages inbox entries in PostgreSQL.
type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func (s *Store) Migrate(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, migrateSQL)
	return err
}

// Insert inserts a message entry using ON CONFLICT DO NOTHING for deduplication.
func (s *Store) Insert(ctx context.Context, e *Entry) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO inbox (user_id, message_id, conversation_id, sender_id, payload)
		 VALUES ($1, $2, $3, $4, $5)
		 ON CONFLICT (user_id, message_id) DO NOTHING`,
		e.UserID, e.MessageID, e.ConversationID, e.SenderID, e.Payload)
	return err
}

// GetUndelivered returns undelivered messages for a user, ordered by creation time.
func (s *Store) GetUndelivered(ctx context.Context, userID []byte, limit int) ([]Entry, error) {
	return s.query(ctx,
		`SELECT id, user_id, message_id, conversation_id, sender_id, payload, delivered, read, created_at
		 FROM inbox WHERE user_id = $1 AND delivered = FALSE
		 ORDER BY created_at LIMIT $2`, userID, limit)
}

// GetHistory returns message history for a user-conversation pair with cursor-based pagination.
func (s *Store) GetHistory(ctx context.Context, userID, conversationID []byte, afterID int64, limit int) ([]Entry, error) {
	return s.query(ctx,
		`SELECT id, user_id, message_id, conversation_id, sender_id, payload, delivered, read, created_at
		 FROM inbox WHERE user_id = $1 AND conversation_id = $2 AND id > $3
		 ORDER BY created_at DESC LIMIT $4`, userID, conversationID, afterID, limit)
}

// GetAfterMessageID returns all inbox entries for a user created after the given message_id.
func (s *Store) GetAfterMessageID(ctx context.Context, userID, lastSeenMessageID []byte, limit int) ([]Entry, error) {
	return s.query(ctx,
		`SELECT i.id, i.user_id, i.message_id, i.conversation_id, i.sender_id, i.payload, i.delivered, i.read, i.created_at
		 FROM inbox i
		 WHERE i.user_id = $1 AND i.created_at > (
		   SELECT created_at FROM inbox WHERE user_id = $1 AND message_id = $2 LIMIT 1
		 )
		 ORDER BY i.created_at LIMIT $3`, userID, lastSeenMessageID, limit)
}

// MarkDelivered marks a message as delivered.
func (s *Store) MarkDelivered(ctx context.Context, id int64) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE inbox SET delivered = TRUE WHERE id = $1`, id)
	return err
}

// MarkRead marks a message as read.
func (s *Store) MarkRead(ctx context.Context, id int64) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE inbox SET read = TRUE WHERE id = $1`, id)
	return err
}

// Count returns the total number of entries for a user (for testing).
func (s *Store) Count(ctx context.Context, userID []byte) (int64, error) {
	var c int64
	err := s.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM inbox WHERE user_id = $1`, userID).Scan(&c)
	return c, err
}

func (s *Store) query(ctx context.Context, sql string, args ...interface{}) ([]Entry, error) {
	rows, err := s.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("inbox query: %w", err)
	}
	defer rows.Close()

	var entries []Entry
	for rows.Next() {
		var e Entry
		if err := rows.Scan(&e.ID, &e.UserID, &e.MessageID, &e.ConversationID,
			&e.SenderID, &e.Payload, &e.Delivered, &e.Read, &e.CreatedAt); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}
