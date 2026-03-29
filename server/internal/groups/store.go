package groups

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

const migrateSQL = `
CREATE TABLE IF NOT EXISTS groups (
    id         BYTEA PRIMARY KEY,
    name       TEXT NOT NULL,
    creator_id BYTEA NOT NULL,
    settings   JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS group_members (
    group_id  BYTEA NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    user_id   BYTEA NOT NULL,
    role      VARCHAR(20) NOT NULL DEFAULT 'member'
              CHECK (role IN ('admin','moderator','member')),
    joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (group_id, user_id)
);
CREATE INDEX IF NOT EXISTS idx_group_members_user ON group_members(user_id);
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

func (s *Store) CreateGroup(ctx context.Context, g *Group) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx,
		`INSERT INTO groups (id, name, creator_id, settings) VALUES ($1, $2, $3, $4)`,
		g.ID, g.Name, g.CreatorID, g.Settings)
	if err != nil {
		return fmt.Errorf("insert group: %w", err)
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO group_members (group_id, user_id, role) VALUES ($1, $2, 'admin')`,
		g.ID, g.CreatorID)
	if err != nil {
		return fmt.Errorf("insert creator as admin: %w", err)
	}

	return tx.Commit(ctx)
}

func (s *Store) GetGroup(ctx context.Context, groupID []byte) (*Group, error) {
	row := s.pool.QueryRow(ctx,
		`SELECT id, name, creator_id, settings, created_at, updated_at FROM groups WHERE id = $1`, groupID)
	var g Group
	if err := row.Scan(&g.ID, &g.Name, &g.CreatorID, &g.Settings, &g.CreatedAt, &g.UpdatedAt); err != nil {
		return nil, fmt.Errorf("group not found: %w", err)
	}
	return &g, nil
}

func (s *Store) AddMember(ctx context.Context, groupID, userID []byte, role MemberRole) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO group_members (group_id, user_id, role) VALUES ($1, $2, $3)
		 ON CONFLICT (group_id, user_id) DO NOTHING`,
		groupID, userID, string(role))
	return err
}

func (s *Store) RemoveMember(ctx context.Context, groupID, userID []byte) error {
	_, err := s.pool.Exec(ctx,
		`DELETE FROM group_members WHERE group_id = $1 AND user_id = $2`, groupID, userID)
	return err
}

func (s *Store) UpdateSettings(ctx context.Context, groupID []byte, settings []byte) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE groups SET settings = $2, updated_at = NOW() WHERE id = $1`, groupID, settings)
	return err
}

func (s *Store) BanMember(ctx context.Context, groupID, userID []byte) error {
	return s.RemoveMember(ctx, groupID, userID)
}

func (s *Store) GetMembers(ctx context.Context, groupID []byte) ([]GroupMember, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT group_id, user_id, role, joined_at FROM group_members WHERE group_id = $1`, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []GroupMember
	for rows.Next() {
		var m GroupMember
		var role string
		if err := rows.Scan(&m.GroupID, &m.UserID, &role, &m.JoinedAt); err != nil {
			return nil, err
		}
		m.Role = MemberRole(role)
		members = append(members, m)
	}
	return members, rows.Err()
}

func (s *Store) IsMember(ctx context.Context, groupID, userID []byte) (bool, error) {
	var exists bool
	err := s.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM group_members WHERE group_id = $1 AND user_id = $2)`,
		groupID, userID).Scan(&exists)
	return exists, err
}

func (s *Store) AdminCount(ctx context.Context, groupID []byte) (int, error) {
	var count int
	err := s.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM group_members WHERE group_id = $1 AND role = 'admin'`,
		groupID).Scan(&count)
	return count, err
}
