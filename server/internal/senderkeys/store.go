package senderkeys

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const cacheTTL = 1 * time.Hour

// CachedKey is the serialized form stored in Redis.
type CachedKey struct {
	KeyID      uint32 `json:"key_id"`
	ChainKey   []byte `json:"chain_key"`
	SigningKey []byte `json:"signing_key"`
}

// Cache stores sender keys in Redis for fast lookups.
type Cache struct {
	rdb *redis.Client
}

func NewCache(rdb *redis.Client) *Cache {
	return &Cache{rdb: rdb}
}

func cacheKey(groupID []byte) string {
	return "group:" + hex.EncodeToString(groupID) + ":sender_keys"
}

// Set stores a sender key for a user in a group.
func (c *Cache) Set(ctx context.Context, groupID, userID []byte, ck CachedKey) error {
	data, err := json.Marshal(ck)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	key := cacheKey(groupID)
	field := hex.EncodeToString(userID)
	pipe := c.rdb.Pipeline()
	pipe.HSet(ctx, key, field, data)
	pipe.Expire(ctx, key, cacheTTL)
	_, err = pipe.Exec(ctx)
	return err
}

// Get retrieves a sender key for a user in a group.
func (c *Cache) Get(ctx context.Context, groupID, userID []byte) (*CachedKey, error) {
	key := cacheKey(groupID)
	field := hex.EncodeToString(userID)
	data, err := c.rdb.HGet(ctx, key, field).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var ck CachedKey
	if err := json.Unmarshal(data, &ck); err != nil {
		return nil, err
	}
	return &ck, nil
}

// Delete removes a sender key for a user in a group.
func (c *Cache) Delete(ctx context.Context, groupID, userID []byte) error {
	key := cacheKey(groupID)
	field := hex.EncodeToString(userID)
	return c.rdb.HDel(ctx, key, field).Err()
}

// DeleteAll removes all sender keys for a group (used during full rotation).
func (c *Cache) DeleteAll(ctx context.Context, groupID []byte) error {
	return c.rdb.Del(ctx, cacheKey(groupID)).Err()
}
