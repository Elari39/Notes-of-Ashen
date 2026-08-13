package middleware

import (
	"net/http"

	apperrors "notes-of-ashen/internal/errors"
	"notes-of-ashen/internal/response"
	"notes-of-ashen/model"
)

// RAGChatPageMiddleware 在认证、限流和并发控制之前检查公开问答页开关。
//
// 页面关闭时 API 必须与 /ask 路由一样表现为不存在；若把该检查放在 Handler
// 内部，未登录请求会先得到 401，受限请求会先得到 429/503，从而泄露功能状态。
type RAGChatPageMiddleware struct {
	store *model.Store
}

func NewRAGChatPageMiddleware(store *model.Store) *RAGChatPageMiddleware {
	return &RAGChatPageMiddleware{store: store}
}

func (m *RAGChatPageMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if m == nil || m.store == nil {
			response.ErrorCtx(r.Context(), w, apperrors.ServiceUnavailable("rag chat page check is unavailable"))
			return
		}
		settings, err := m.store.SiteSettings(r.Context())
		if err != nil {
			response.ErrorCtx(r.Context(), w, err)
			return
		}
		if !settings.RAGChatPageEnabled {
			response.ErrorCtx(r.Context(), w, apperrors.NotFound("rag chat not found"))
			return
		}
		next(w, r)
	}
}
