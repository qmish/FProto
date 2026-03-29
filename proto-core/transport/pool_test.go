package transport

import (
	"testing"
)

type mockConn struct {
	tp   Type
	addr string
}

func (m *mockConn) ReadMessage() ([]byte, error) { return nil, nil }
func (m *mockConn) WriteMessage([]byte) error     { return nil }
func (m *mockConn) Close() error                   { return nil }
func (m *mockConn) Type() Type                     { return m.tp }
func (m *mockConn) RemoteAddr() string             { return m.addr }

func TestConnPool_AddAndCount(t *testing.T) {
	pool := NewConnPool(5)

	if err := pool.Add("c1", &mockConn{tp: TypeWebSocket}); err != nil {
		t.Fatalf("add: %v", err)
	}
	if pool.Count() != 1 {
		t.Fatalf("expected 1, got %d", pool.Count())
	}

	if err := pool.Add("c2", &mockConn{tp: TypeQUIC}); err != nil {
		t.Fatalf("add: %v", err)
	}
	if pool.Count() != 2 {
		t.Fatalf("expected 2, got %d", pool.Count())
	}
}

func TestConnPool_MaxConns(t *testing.T) {
	pool := NewConnPool(2)

	_ = pool.Add("c1", &mockConn{tp: TypeWebSocket})
	_ = pool.Add("c2", &mockConn{tp: TypeQUIC})

	err := pool.Add("c3", &mockConn{tp: TypeGRPC})
	if err == nil {
		t.Fatal("expected error when pool is full")
	}
}

func TestConnPool_Remove(t *testing.T) {
	pool := NewConnPool(5)
	_ = pool.Add("c1", &mockConn{tp: TypeWebSocket})
	pool.Remove("c1")

	if pool.Count() != 0 {
		t.Fatalf("expected 0 after remove, got %d", pool.Count())
	}
}

func TestConnPool_BestBySeq(t *testing.T) {
	pool := NewConnPool(5)
	_ = pool.Add("c1", &mockConn{tp: TypeWebSocket, addr: "1.1.1.1"})
	_ = pool.Add("c2", &mockConn{tp: TypeWebSocket, addr: "2.2.2.2"})

	pool.UpdateSeq("c1", 10)
	pool.UpdateSeq("c2", 20)

	conn, id, ok := pool.Best()
	if !ok {
		t.Fatal("expected a best connection")
	}
	if id != "c2" {
		t.Fatalf("expected c2 (higher seq), got %s", id)
	}
	if conn.RemoteAddr() != "2.2.2.2" {
		t.Fatalf("unexpected addr: %s", conn.RemoteAddr())
	}
}

func TestConnPool_BestByTransportPriority(t *testing.T) {
	pool := NewConnPool(5)
	_ = pool.Add("ws", &mockConn{tp: TypeWebSocket})
	_ = pool.Add("quic", &mockConn{tp: TypeQUIC})
	_ = pool.Add("grpc", &mockConn{tp: TypeGRPC})

	_, id, ok := pool.Best()
	if !ok {
		t.Fatal("expected a best connection")
	}
	if id != "quic" {
		t.Fatalf("expected quic (highest priority), got %s", id)
	}
}

func TestConnPool_BestEmpty(t *testing.T) {
	pool := NewConnPool(5)
	_, _, ok := pool.Best()
	if ok {
		t.Fatal("expected no best for empty pool")
	}
}

func TestConnPool_All(t *testing.T) {
	pool := NewConnPool(5)
	_ = pool.Add("a", &mockConn{tp: TypeWebSocket})
	_ = pool.Add("b", &mockConn{tp: TypeQUIC})

	ids := pool.All()
	if len(ids) != 2 {
		t.Fatalf("expected 2, got %d", len(ids))
	}
}
