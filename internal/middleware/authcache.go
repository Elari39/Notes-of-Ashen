package middleware

import (
	"context"
	"fmt"

	appcache "notes-of-ashen/internal/cache"

	"github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/core/logx"
)

// redisUserCache 基于 appcache.JSONCache 实现认证用户快照缓存。
// Redis 故障时 Get 返回 (nil, false) 触发中间件降级直查 DB，不产生错误。
type redisUserCache struct {
	cache *appcache.JSONCache
}

func authUserCacheKey(userID uint64) string {
	return fmt.Sprintf("auth:user:%d", userID)
}

// NewAuthUserCache 构造一个基于 Redis 的认证用户缓存；redisClient 为 nil 时返回 nil，
// 中间件据此退化为每请求直查 DB。
func NewAuthUserCache(redisClient *redis.Client) AuthUserCache {
	c := appcache.NewJSONCache(redisClient)
	if c == nil {
		return nil
	}
	return &redisUserCache{cache: c}
}

func (c *redisUserCache) Get(ctx context.Context, userID uint64) (*authUserSnapshot, bool) {
	var snapshot authUserSnapshot
	hit, err := c.cache.Get(ctx, authUserCacheKey(userID), &snapshot)
	if err != nil {
		logx.Errorf("auth user cache read failed, fallback to db: userID=%d, err=%v", userID, err)
		return nil, false
	}
	if !hit {
		return nil, false
	}
	return &snapshot, true
}

func (c *redisUserCache) Set(ctx context.Context, userID uint64, snapshot authUserSnapshot) {
	if err := c.cache.Set(ctx, authUserCacheKey(userID), snapshot, authUserCacheTTL); err != nil {
		logx.Errorf("auth user cache write failed: userID=%d, err=%v", userID, err)
	}
}

func (c *redisUserCache) Delete(ctx context.Context, userID uint64) {
	if err := c.cache.Delete(ctx, authUserCacheKey(userID)); err != nil {
		logx.Errorf("auth user cache delete failed: userID=%d, err=%v", userID, err)
	}
}
