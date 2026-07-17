package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"sync/atomic"
	"time"

	"notes-of-ashen/internal/response"
)

const maxRequestIDLength = 128

// RequestID 为每个请求生成或校验后透传 X-Request-Id，写入响应头并存入 context，
// 供 response.ErrorCtx 记录带上下文的错误日志。
func RequestID(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-Id")
		source := "client"
		if !validRequestID(id) {
			id = newRequestID()
			source = "server"
		}
		w.Header().Set("X-Request-Id", id)
		ctx := response.WithRequestID(r.Context(), id)
		ctx = response.WithRequestIDSource(ctx, source)
		ctx = response.WithRequestMeta(ctx, r.Method, r.URL.Path)
		next(w, r.WithContext(ctx))
	}
}

func validRequestID(id string) bool {
	if len(id) == 0 || len(id) > maxRequestIDLength {
		return false
	}
	for _, char := range id {
		if (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') || char == '-' || char == '_' {
			continue
		}
		return false
	}
	return true
}

// fallbackCounter 在 rand.Read 失败时为退化 ID 提供递增计数，配合纳秒时间戳保证唯一，
// 避免所有请求共用同一占位串导致日志聚合无法区分链路。
var fallbackCounter uint64

func newRequestID() string {
	var b [12]byte
	if _, err := rand.Read(b[:]); err != nil {
		// rand.Read 极少失败；退化为纳秒时间戳 + 原子计数，保证唯一且非空。
		seq := atomic.AddUint64(&fallbackCounter, 1)
		return fmt.Sprintf("%016x-%012d", time.Now().UnixNano(), seq)
	}
	return hex.EncodeToString(b[:])
}
