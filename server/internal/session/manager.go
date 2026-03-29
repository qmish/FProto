package session

import (
	"context"
	"crypto/rand"
	"fmt"
	"time"
)

// Manager orchestrates session lifecycle using Redis (hot) and PG (durable) stores.
type Manager struct {
	redis  *RedisStore
	pg     *PGStore
	config Config
}

func NewManager(redis *RedisStore, pg *PGStore, config Config) *Manager {
	return &Manager{redis: redis, pg: pg, config: config}
}

func (m *Manager) CreateSession(ctx context.Context, userID, deviceID []byte) (*Session, error) {
	sessionID := make([]byte, 16)
	if _, err := rand.Read(sessionID); err != nil {
		return nil, fmt.Errorf("generate session id: %w", err)
	}

	now := time.Now()
	s := &Session{
		SessionID:   sessionID,
		UserID:      userID,
		DeviceID:    deviceID,
		Status:      StatusConnecting,
		CryptoState: nil,
		CreatedAt:   now,
		LastActive:  now,
		ExpiresAt:   now.Add(m.config.ExpiryTimeout),
	}

	if err := m.redis.Create(ctx, s); err != nil {
		return nil, fmt.Errorf("redis create: %w", err)
	}

	if m.pg != nil {
		if err := m.pg.Create(ctx, s); err != nil {
			return nil, fmt.Errorf("pg create: %w", err)
		}
	}

	return s, nil
}

func (m *Manager) GetSession(ctx context.Context, sessionID []byte) (*Session, error) {
	s, err := m.redis.Get(ctx, sessionID)
	if err != nil && m.pg != nil {
		s, err = m.pg.Get(ctx, sessionID)
	}
	return s, err
}

func (m *Manager) TransitionSession(ctx context.Context, sessionID []byte, event Event) (*Session, error) {
	s, err := m.redis.Get(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	next, destroyed, err := Transition(s.Status, event)
	if err != nil {
		return nil, err
	}

	if destroyed {
		if delErr := m.redis.Delete(ctx, sessionID); delErr != nil {
			return nil, fmt.Errorf("delete session: %w", delErr)
		}
		if m.pg != nil {
			_ = m.pg.Delete(ctx, sessionID)
		}
		return nil, nil
	}

	if err := m.redis.UpdateStatus(ctx, sessionID, next); err != nil {
		return nil, fmt.Errorf("redis update status: %w", err)
	}

	if event == EventPacketReceived {
		_ = m.redis.UpdateLastActive(ctx, sessionID)
	}

	if m.pg != nil {
		_ = m.pg.UpdateStatus(ctx, sessionID, next)
	}

	s.Status = next
	return s, nil
}

func (m *Manager) UpdateCryptoState(ctx context.Context, sessionID, cryptoState []byte) error {
	if err := m.redis.UpdateCryptoState(ctx, sessionID, cryptoState); err != nil {
		return err
	}
	if m.pg != nil {
		return m.pg.UpdateCryptoState(ctx, sessionID, cryptoState)
	}
	return nil
}

func (m *Manager) RegisterConnection(ctx context.Context, sessionID []byte, conn *Connection) (int, error) {
	existing, err := m.redis.GetConnections(ctx, sessionID)
	if err != nil {
		return 0, err
	}
	if len(existing) >= m.config.MaxConnections {
		return len(existing), fmt.Errorf("max connections (%d) reached", m.config.MaxConnections)
	}

	if err := m.redis.AddConnection(ctx, sessionID, conn); err != nil {
		return 0, err
	}
	return len(existing) + 1, nil
}

func (m *Manager) UnregisterConnection(ctx context.Context, sessionID []byte, connectionID string) error {
	return m.redis.RemoveConnection(ctx, sessionID, connectionID)
}

func (m *Manager) GetUserSessions(ctx context.Context, userID []byte) ([][]byte, error) {
	return m.redis.GetUserSessions(ctx, userID)
}
