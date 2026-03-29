package media

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

const migrateSQL = `
CREATE TABLE IF NOT EXISTS media_uploads (
    id           BIGSERIAL PRIMARY KEY,
    stream_id    BYTEA UNIQUE NOT NULL,
    user_id      BYTEA NOT NULL,
    mime_type    VARCHAR(255) NOT NULL,
    total_size   BIGINT NOT NULL,
    status       VARCHAR(20) NOT NULL DEFAULT 'uploading'
                 CHECK (status IN ('uploading','completed','failed','expired')),
    s3_key       TEXT NOT NULL,
    s3_url       TEXT,
    file_hash    BYTEA,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_media_user ON media_uploads(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_media_status ON media_uploads(status) WHERE status = 'uploading';
`

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

func (s *Store) Insert(ctx context.Context, u *Upload) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO media_uploads (stream_id, user_id, mime_type, total_size, status, s3_key, file_hash)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		u.StreamID, u.UserID, u.MimeType, u.TotalSize, string(StatusUploading), u.S3Key, u.FileHash)
	return err
}

func (s *Store) GetByStreamID(ctx context.Context, streamID []byte) (*Upload, error) {
	row := s.pool.QueryRow(ctx,
		`SELECT id, stream_id, user_id, mime_type, total_size, status, s3_key, s3_url, file_hash, created_at, updated_at
		 FROM media_uploads WHERE stream_id = $1`, streamID)

	var u Upload
	var status string
	err := row.Scan(&u.ID, &u.StreamID, &u.UserID, &u.MimeType, &u.TotalSize,
		&status, &u.S3Key, &u.S3URL, &u.FileHash, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("media upload not found: %w", err)
	}
	u.Status = Status(status)
	return &u, nil
}

func (s *Store) MarkCompleted(ctx context.Context, streamID []byte, s3URL string) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE media_uploads SET status = 'completed', s3_url = $2, updated_at = NOW() WHERE stream_id = $1`,
		streamID, s3URL)
	return err
}

func (s *Store) MarkFailed(ctx context.Context, streamID []byte) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE media_uploads SET status = 'failed', updated_at = NOW() WHERE stream_id = $1`,
		streamID)
	return err
}
