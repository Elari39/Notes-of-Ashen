package middleware

import (
	"net/http"
	"time"

	apperrors "notes-of-ashen/internal/errors"
	basehandler "notes-of-ashen/internal/httphelper"
	"notes-of-ashen/internal/response"
	"notes-of-ashen/internal/security"

	"github.com/redis/go-redis/v9"
)

type RateLimitMiddleware struct {
	redisClient *redis.Client
	name        string
	limit       int64
	window      time.Duration
}

func NewRateLimitMiddleware(redisClient *redis.Client, name string, limit int64, window time.Duration) *RateLimitMiddleware {
	return &RateLimitMiddleware{
		redisClient: redisClient,
		name:        name,
		limit:       limit,
		window:      window,
	}
}

func (m *RateLimitMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ip := basehandler.Meta(r).IP
		key := security.RateLimitKey(m.name, ip)
		count, err := m.redisClient.Incr(r.Context(), key).Result()
		if err != nil {
			response.Error(w, err)
			return
		}
		if count == 1 {
			if err := m.redisClient.Expire(r.Context(), key, m.window).Err(); err != nil {
				response.Error(w, err)
				return
			}
		}
		if count > m.limit {
			response.Error(w, apperrors.TooManyRequests("too many requests"))
			return
		}
		next(w, r)
	}
}
