package session

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisStore implements Store using Redis hashes, sets, and TTLs.
type RedisStore struct {
	client *redis.Client
	ttl    time.Duration
}

// NewRedisStore creates a Redis-backed session store.
func NewRedisStore(client *redis.Client, ttl time.Duration) *RedisStore {
	if ttl == 0 {
		ttl = 24 * time.Hour
	}
	return &RedisStore{client: client, ttl: ttl}
}

func sessionKey(id []byte) string       { return "session:" + hex.EncodeToString(id) }
func connKey(id []byte) string          { return sessionKey(id) + ":connections" }
func userSessionsKey(uid []byte) string { return "user:" + hex.EncodeToString(uid) + ":sessions" }

func (r *RedisStore) Create(ctx context.Context, s *Session) error {
	key := sessionKey(s.SessionID)
	fields := map[string]interface{}{
		"user_id":      hex.EncodeToString(s.UserID),
		"device_id":    hex.EncodeToString(s.DeviceID),
		"status":       string(s.Status),
		"crypto_state": hex.EncodeToString(s.CryptoState),
		"created_at":   s.CreatedAt.Unix(),
		"last_active":  s.LastActive.Unix(),
		"expires_at":   s.ExpiresAt.Unix(),
	}

	pipe := r.client.Pipeline()
	pipe.HSet(ctx, key, fields)
	pipe.Expire(ctx, key, r.ttl)
	pipe.SAdd(ctx, userSessionsKey(s.UserID), hex.EncodeToString(s.SessionID))
	_, err := pipe.Exec(ctx)
	return err
}

func (r *RedisStore) Get(ctx context.Context, sessionID []byte) (*Session, error) {
	key := sessionKey(sessionID)
	vals, err := r.client.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("redis hgetall: %w", err)
	}
	if len(vals) == 0 {
		return nil, fmt.Errorf("session not found: %s", hex.EncodeToString(sessionID))
	}

	userID, _ := hex.DecodeString(vals["user_id"])
	deviceID, _ := hex.DecodeString(vals["device_id"])
	cryptoState, _ := hex.DecodeString(vals["crypto_state"])
	createdAt, _ := strconv.ParseInt(vals["created_at"], 10, 64)
	lastActive, _ := strconv.ParseInt(vals["last_active"], 10, 64)
	expiresAt, _ := strconv.ParseInt(vals["expires_at"], 10, 64)

	conns, _ := r.GetConnections(ctx, sessionID)

	return &Session{
		SessionID:   sessionID,
		UserID:      userID,
		DeviceID:    deviceID,
		Status:      Status(vals["status"]),
		CryptoState: cryptoState,
		CreatedAt:   time.Unix(createdAt, 0),
		LastActive:  time.Unix(lastActive, 0),
		ExpiresAt:   time.Unix(expiresAt, 0),
		Connections: conns,
	}, nil
}

func (r *RedisStore) UpdateStatus(ctx context.Context, sessionID []byte, status Status) error {
	return r.client.HSet(ctx, sessionKey(sessionID), "status", string(status)).Err()
}

func (r *RedisStore) UpdateCryptoState(ctx context.Context, sessionID []byte, cryptoState []byte) error {
	return r.client.HSet(ctx, sessionKey(sessionID), "crypto_state", hex.EncodeToString(cryptoState)).Err()
}

func (r *RedisStore) UpdateLastActive(ctx context.Context, sessionID []byte) error {
	pipe := r.client.Pipeline()
	pipe.HSet(ctx, sessionKey(sessionID), "last_active", time.Now().Unix())
	pipe.Expire(ctx, sessionKey(sessionID), r.ttl)
	_, err := pipe.Exec(ctx)
	return err
}

func (r *RedisStore) Delete(ctx context.Context, sessionID []byte) error {
	s, err := r.Get(ctx, sessionID)
	if err != nil {
		return err
	}
	pipe := r.client.Pipeline()
	pipe.Del(ctx, sessionKey(sessionID))
	pipe.Del(ctx, connKey(sessionID))
	pipe.SRem(ctx, userSessionsKey(s.UserID), hex.EncodeToString(sessionID))
	_, err = pipe.Exec(ctx)
	return err
}

type connJSON struct {
	Transport     string `json:"transport"`
	GatewayID     string `json:"gateway_id"`
	RemoteAddr    string `json:"remote_addr"`
	ConnectedAt   int64  `json:"connected_at"`
	LastPacketSeq uint64 `json:"last_packet_seq"`
}

func (r *RedisStore) AddConnection(ctx context.Context, sessionID []byte, conn *Connection) error {
	data, err := json.Marshal(connJSON{
		Transport:     string(conn.Transport),
		GatewayID:     conn.GatewayID,
		RemoteAddr:    conn.RemoteAddr,
		ConnectedAt:   conn.ConnectedAt.Unix(),
		LastPacketSeq: conn.LastPacketSeq,
	})
	if err != nil {
		return err
	}
	return r.client.HSet(ctx, connKey(sessionID), conn.ConnectionID, string(data)).Err()
}

func (r *RedisStore) RemoveConnection(ctx context.Context, sessionID []byte, connectionID string) error {
	return r.client.HDel(ctx, connKey(sessionID), connectionID).Err()
}

func (r *RedisStore) GetConnections(ctx context.Context, sessionID []byte) ([]Connection, error) {
	vals, err := r.client.HGetAll(ctx, connKey(sessionID)).Result()
	if err != nil {
		return nil, err
	}
	conns := make([]Connection, 0, len(vals))
	for connID, raw := range vals {
		var cj connJSON
		if err := json.Unmarshal([]byte(raw), &cj); err != nil {
			continue
		}
		conns = append(conns, Connection{
			ConnectionID:  connID,
			Transport:     TransportType(cj.Transport),
			GatewayID:     cj.GatewayID,
			RemoteAddr:    cj.RemoteAddr,
			ConnectedAt:   time.Unix(cj.ConnectedAt, 0),
			LastPacketSeq: cj.LastPacketSeq,
		})
	}
	return conns, nil
}

func (r *RedisStore) GetUserSessions(ctx context.Context, userID []byte) ([][]byte, error) {
	members, err := r.client.SMembers(ctx, userSessionsKey(userID)).Result()
	if err != nil {
		return nil, err
	}
	result := make([][]byte, 0, len(members))
	for _, m := range members {
		id, _ := hex.DecodeString(m)
		result = append(result, id)
	}
	return result, nil
}
