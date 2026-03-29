package transport

import (
	"fmt"
	"sync"
)

// DefaultMaxConns is the default maximum connections per pool.
const DefaultMaxConns = 5

var transportPriority = map[Type]int{
	TypeQUIC:      3,
	TypeWebSocket: 2,
	TypeGRPC:      1,
}

// ConnPool manages multiple transport connections for a single session.
type ConnPool struct {
	mu       sync.RWMutex
	conns    map[string]poolEntry
	maxConns int
}

type poolEntry struct {
	conn          Conn
	lastPacketSeq uint64
}

// NewConnPool creates a connection pool with the given max capacity.
func NewConnPool(maxConns int) *ConnPool {
	if maxConns <= 0 {
		maxConns = DefaultMaxConns
	}
	return &ConnPool{
		conns:    make(map[string]poolEntry),
		maxConns: maxConns,
	}
}

// Add registers a connection in the pool. Returns error if pool is full.
func (p *ConnPool) Add(id string, conn Conn) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.conns) >= p.maxConns {
		return fmt.Errorf("connection pool full: max %d", p.maxConns)
	}
	p.conns[id] = poolEntry{conn: conn}
	return nil
}

// Remove unregisters a connection from the pool.
func (p *ConnPool) Remove(id string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.conns, id)
}

// UpdateSeq updates the last packet sequence number for a connection.
func (p *ConnPool) UpdateSeq(id string, seq uint64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if e, ok := p.conns[id]; ok {
		e.lastPacketSeq = seq
		p.conns[id] = e
	}
}

// Best selects the best connection for push delivery.
// Priority: highest lastPacketSeq, tie-break by transport (QUIC > WS > gRPC).
func (p *ConnPool) Best() (Conn, string, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var bestID string
	var bestEntry poolEntry
	found := false

	for id, entry := range p.conns {
		if !found {
			bestID = id
			bestEntry = entry
			found = true
			continue
		}
		if entry.lastPacketSeq > bestEntry.lastPacketSeq {
			bestID = id
			bestEntry = entry
		} else if entry.lastPacketSeq == bestEntry.lastPacketSeq {
			if transportPriority[entry.conn.Type()] > transportPriority[bestEntry.conn.Type()] {
				bestID = id
				bestEntry = entry
			}
		}
	}

	if !found {
		return nil, "", false
	}
	return bestEntry.conn, bestID, true
}

// Count returns the number of active connections.
func (p *ConnPool) Count() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.conns)
}

// All returns all connection IDs.
func (p *ConnPool) All() []string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	ids := make([]string, 0, len(p.conns))
	for id := range p.conns {
		ids = append(ids, id)
	}
	return ids
}
