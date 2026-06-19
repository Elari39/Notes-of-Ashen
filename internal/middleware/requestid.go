package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"

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

func newRequestID() string {
	var b [12]byte
	if _, err := rand.Read(b[:]); err != nil {
		// rand.Read 极少失败；退化为占位串，保证非空。
		return "000000000000000000000000"
	}
	return hex.EncodeToString(b[:])
}
