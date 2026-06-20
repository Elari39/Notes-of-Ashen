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

// RequestID 为每个请求生成或透传 X-Request-Id，写入响应头并存入 context，
// 供 response.ErrorCtx 记录带上下文的错误日志。
func RequestID(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-Id")
		if id == "" {
			id = newRequestID()
		}
		w.Header().Set("X-Request-Id", id)
		ctx := response.WithRequestID(r.Context(), id)
		ctx = response.WithRequestMeta(ctx, r.Method, r.URL.Path)
		next(w, r.WithContext(ctx))
	}
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
