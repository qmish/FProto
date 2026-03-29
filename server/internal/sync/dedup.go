package sync

import (
	"context"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const dedupTTL = 48 * time.Hour

// Dedup checks message uniqueness using Redis SET NX with TTL.
type Dedup struct {
	rdb *redis.Client
}

func NewDedup(rdb *redis.Client) *Dedup {
	return &Dedup{rdb: rdb}
}

func dedupKey(userID, messageID []byte) string {
	return fmt.Sprintf("dedup:%s:%s", hex.EncodeToString(userID), hex.EncodeToString(messageID))
}

// IsDuplicate returns true if the message was already processed.
// If not a duplicate, it marks it as seen atomically.
func (d *Dedup) IsDuplicate(ctx context.Context, userID, messageID []byte) (bool, error) {
	key := dedupKey(userID, messageID)
	set, err := d.rdb.SetNX(ctx, key, "1", dedupTTL).Result()
	if err != nil {
		return false, fmt.Errorf("dedup setnx: %w", err)
	}
	return !set, nil // if SetNX returned false, key already existed -> duplicate
}
