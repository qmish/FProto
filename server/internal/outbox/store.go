package outbox

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const migrateSQL = `
CREATE TABLE IF NOT EXISTS outbox (
    id            BIGSERIAL PRIMARY KEY,
    message_id    BYTEA UNIQUE NOT NULL,
    topic         TEXT NOT NULL,
    partition_key BYTEA NOT NULL,
    payload       BYTEA NOT NULL,
    status        VARCHAR(20) NOT NULL DEFAULT 'pending'
                  CHECK (status IN ('pending','published','failed')),
    retry_count   INT NOT NULL DEFAULT 0,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    published_at  TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_outbox_pending ON outbox(status, created_at) WHERE status = 'pending';
`

// Store manages outbox entries in PostgreSQL.
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

func (s *Store) Insert(ctx context.Context, e *Entry) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO outbox (message_id, topic, partition_key, payload, status)
		 VALUES ($1, $2, $3, $4, $5)`,
		e.MessageID, e.Topic, e.PartitionKey, e.Payload, string(StatusPending))
	return err
}

func (s *Store) FetchPending(ctx context.Context, limit int) ([]Entry, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, message_id, topic, partition_key, payload, status, retry_count, created_at, published_at
		 FROM outbox WHERE status = 'pending' ORDER BY created_at LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []Entry
	for rows.Next() {
		var e Entry
		var status string
		err := rows.Scan(&e.ID, &e.MessageID, &e.Topic, &e.PartitionKey, &e.Payload,
			&status, &e.RetryCount, &e.CreatedAt, &e.PublishedAt)
		if err != nil {
			return nil, err
		}
		e.Status = Status(status)
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

func (s *Store) MarkPublished(ctx context.Context, id int64) error {
	now := time.Now()
	_, err := s.pool.Exec(ctx,
		`UPDATE outbox SET status = 'published', published_at = $2 WHERE id = $1`,
		id, now)
	return err
}

func (s *Store) MarkFailed(ctx context.Context, id int64) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE outbox SET status = 'failed' WHERE id = $1`, id)
	return err
}

func (s *Store) IncrementRetry(ctx context.Context, id int64) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE outbox SET retry_count = retry_count + 1 WHERE id = $1`, id)
	return err
}

func (s *Store) GetByMessageID(ctx context.Context, messageID []byte) (*Entry, error) {
	row := s.pool.QueryRow(ctx,
		`SELECT id, message_id, topic, partition_key, payload, status, retry_count, created_at, published_at
		 FROM outbox WHERE message_id = $1`, messageID)

	var e Entry
	var status string
	err := row.Scan(&e.ID, &e.MessageID, &e.Topic, &e.PartitionKey, &e.Payload,
		&status, &e.RetryCount, &e.CreatedAt, &e.PublishedAt)
	if err != nil {
		return nil, fmt.Errorf("outbox entry not found: %w", err)
	}
	e.Status = Status(status)
	return &e, nil
}
