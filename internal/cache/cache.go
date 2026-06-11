package cache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
)

type JSONCache struct {
	client *redis.Client
}

const (
	cacheOperationTimeout    = 200 * time.Millisecond
	cacheDeletePrefixTimeout = 500 * time.Millisecond
)

func NewJSONCache(client *redis.Client) *JSONCache {
	if client == nil {
		return nil
	}
	return &JSONCache{client: client}
}

func (c *JSONCache) Get(ctx context.Context, key string, target interface{}) (bool, error) {
	if c == nil || c.client == nil {
		return false, nil
	}
	cacheCtx, cancel := context.WithTimeout(ctx, cacheOperationTimeout)
	defer cancel()
	raw, err := c.client.Get(cacheCtx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return false, nil
		}
		return false, err
	}
	if err := json.Unmarshal(raw, target); err != nil {
		return false, err
	}
	return true, nil
}

func (c *JSONCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	if c == nil || c.client == nil {
		return nil
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	cacheCtx, cancel := context.WithTimeout(ctx, cacheOperationTimeout)
	defer cancel()
	return c.client.Set(cacheCtx, key, raw, ttl).Err()
}

func (c *JSONCache) Delete(ctx context.Context, keys ...string) error {
	if c == nil || c.client == nil || len(keys) == 0 {
		return nil
	}
	cacheCtx, cancel := context.WithTimeout(ctx, cacheOperationTimeout)
	defer cancel()
	return c.client.Del(cacheCtx, keys...).Err()
}

func (c *JSONCache) DeletePrefix(ctx context.Context, prefix string) error {
	if c == nil || c.client == nil || prefix == "" {
		return nil
	}
	cacheCtx, cancel := context.WithTimeout(ctx, cacheDeletePrefixTimeout)
	defer cancel()
	var cursor uint64
	for {
		keys, nextCursor, err := c.client.Scan(cacheCtx, cursor, prefix+"*", 100).Result()
		if err != nil {
			return err
		}
		if len(keys) > 0 {
			if err := c.client.Del(cacheCtx, keys...).Err(); err != nil {
				return err
			}
		}
		cursor = nextCursor
		if cursor == 0 {
			return nil
		}
	}
}

func HashKey(prefix string, values ...interface{}) string {
	raw, _ := json.Marshal(values)
	sum := sha256.Sum256(raw)
	return prefix + hex.EncodeToString(sum[:])
}
