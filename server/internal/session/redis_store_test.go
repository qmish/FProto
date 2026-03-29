package session

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

func newTestRedisStore(t *testing.T) *RedisStore {
	t.Helper()
	client := redis.NewClient(&redis.Options{Addr: "localhost:6379", DB: 15})
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Skipf("Redis not available: %v", err)
	}
	client.FlushDB(ctx)
	t.Cleanup(func() { client.FlushDB(ctx); client.Close() })
	return NewRedisStore(client, 24*time.Hour)
}

func TestRedisStore_CreateAndGet(t *testing.T) {
	store := newTestRedisStore(t)
	ctx := context.Background()

	s := &Session{
		SessionID:   []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
		UserID:      []byte{10, 20, 30, 40, 50, 60, 70, 80, 90, 100, 110, 120, 130, 140, 150, 160},
		DeviceID:    []byte{20, 30, 40, 50, 60, 70, 80, 90, 100, 110, 120, 130, 140, 150, 160, 170},
		Status:      StatusConnecting,
		CryptoState: []byte("test-crypto-state"),
		CreatedAt:   time.Now().Truncate(time.Second),
		LastActive:  time.Now().Truncate(time.Second),
		ExpiresAt:   time.Now().Add(24 * time.Hour).Truncate(time.Second),
	}

	if err := store.Create(ctx, s); err != nil {
		t.Fatalf("create: %v", err)
	}

	got, err := store.Get(ctx, s.SessionID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}

	if got.Status != StatusConnecting {
		t.Fatalf("expected connecting, got %s", got.Status)
	}
	if string(got.CryptoState) != "test-crypto-state" {
		t.Fatalf("crypto state mismatch: %q", got.CryptoState)
	}
}

func TestRedisStore_UpdateStatus(t *testing.T) {
	store := newTestRedisStore(t)
	ctx := context.Background()

	s := &Session{
		SessionID: []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
		UserID:    []byte{10, 20},
		DeviceID:  []byte{30, 40},
		Status:    StatusConnecting,
		CreatedAt: time.Now(),
		LastActive: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	_ = store.Create(ctx, s)

	if err := store.UpdateStatus(ctx, s.SessionID, StatusActive); err != nil {
		t.Fatalf("update status: %v", err)
	}

	got, _ := store.Get(ctx, s.SessionID)
	if got.Status != StatusActive {
		t.Fatalf("expected active, got %s", got.Status)
	}
}

func TestRedisStore_Connections(t *testing.T) {
	store := newTestRedisStore(t)
	ctx := context.Background()

	sid := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}
	s := &Session{
		SessionID: sid, UserID: []byte{10, 20}, DeviceID: []byte{30, 40},
		Status: StatusActive, CreatedAt: time.Now(), LastActive: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	_ = store.Create(ctx, s)

	conn := &Connection{
		ConnectionID: "conn-1",
		Transport:    TransportWebSocket,
		GatewayID:    "gw-pod-1",
		RemoteAddr:   "1.2.3.4:5678",
		ConnectedAt:  time.Now(),
	}
	if err := store.AddConnection(ctx, sid, conn); err != nil {
		t.Fatalf("add connection: %v", err)
	}

	conns, err := store.GetConnections(ctx, sid)
	if err != nil {
		t.Fatalf("get connections: %v", err)
	}
	if len(conns) != 1 {
		t.Fatalf("expected 1 connection, got %d", len(conns))
	}
	if conns[0].ConnectionID != "conn-1" {
		t.Fatalf("unexpected connection id: %s", conns[0].ConnectionID)
	}
	if conns[0].Transport != TransportWebSocket {
		t.Fatalf("unexpected transport: %s", conns[0].Transport)
	}

	if err := store.RemoveConnection(ctx, sid, "conn-1"); err != nil {
		t.Fatalf("remove connection: %v", err)
	}
	conns, _ = store.GetConnections(ctx, sid)
	if len(conns) != 0 {
		t.Fatalf("expected 0 connections after removal, got %d", len(conns))
	}
}

func TestRedisStore_UserSessions(t *testing.T) {
	store := newTestRedisStore(t)
	ctx := context.Background()

	uid := []byte{10, 20, 30, 40}

	s1 := &Session{
		SessionID: []byte{1, 1, 1, 1}, UserID: uid, DeviceID: []byte{1},
		Status: StatusActive, CreatedAt: time.Now(), LastActive: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	s2 := &Session{
		SessionID: []byte{2, 2, 2, 2}, UserID: uid, DeviceID: []byte{2},
		Status: StatusActive, CreatedAt: time.Now(), LastActive: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	_ = store.Create(ctx, s1)
	_ = store.Create(ctx, s2)

	sessions, err := store.GetUserSessions(ctx, uid)
	if err != nil {
		t.Fatalf("get user sessions: %v", err)
	}
	if len(sessions) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(sessions))
	}
}

func TestRedisStore_Delete(t *testing.T) {
	store := newTestRedisStore(t)
	ctx := context.Background()

	s := &Session{
		SessionID: []byte{5, 5, 5, 5}, UserID: []byte{10, 20}, DeviceID: []byte{1},
		Status: StatusActive, CreatedAt: time.Now(), LastActive: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	_ = store.Create(ctx, s)

	if err := store.Delete(ctx, s.SessionID); err != nil {
		t.Fatalf("delete: %v", err)
	}

	_, err := store.Get(ctx, s.SessionID)
	if err == nil {
		t.Fatal("expected error after delete")
	}
}
