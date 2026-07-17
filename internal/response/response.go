package response

import (
	"context"
	"errors"
	"net/http"

	apperrors "notes-of-ashen/internal/errors"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest/httpx"
)

type Body struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func Ok(w http.ResponseWriter, data interface{}) {
	httpx.OkJson(w, Body{
		Code:    0,
		Message: "success",
		Data:    data,
	})
}

func NoData(w http.ResponseWriter) {
	httpx.OkJson(w, Body{
		Code:    0,
		Message: "success",
	})
}

// Error 保持原有签名兼容存量调用；不带请求上下文，无法记录 request id / path / method，
// 仅对非 CodeError 的 5xx 错误记录 err。新代码应优先使用 ErrorCtx。
func Error(w http.ResponseWriter, err error) {
	ErrorCtx(context.Background(), w, err)
}

// ErrorCtx 写入错误响应，并在 5xx（非 CodeError 或 CodeError.StatusCode>=500）时
// 记录 request id / method / path / err，便于排查内部错误。
func ErrorCtx(ctx context.Context, w http.ResponseWriter, err error) {
	var codeErr *apperrors.CodeError
	if errors.As(err, &codeErr) {
		if codeErr.StatusCode >= http.StatusInternalServerError {
			logError(ctx, err, codeErr.StatusCode)
		}
		httpx.WriteJson(w, codeErr.StatusCode, Body{
			Code:    codeErr.Code,
			Message: codeErr.Message,
		})
		return
	}

	logError(ctx, err, http.StatusInternalServerError)
	httpx.WriteJson(w, http.StatusInternalServerError, Body{
		Code:    50000,
		Message: "internal server error",
	})
}

func logError(ctx context.Context, err error, statusCode int) {
	logx.WithContext(ctx).Errorf("response error: status=%d method=%s path=%s request_id=%s err=%v",
		statusCode,
		ctx.Value(requestMetaKeyMethod{}),
		ctx.Value(requestMetaKeyPath{}),
		RequestIDFromContext(ctx),
		err,
	)
}

// --- request id & request meta context ---

type requestIDKey struct{}
type requestIDSourceKey struct{}
type requestMetaKeyMethod struct{}
type requestMetaKeyPath struct{}

// WithRequestID 将 request id 存入 context。
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey{}, id)
}

// RequestIDFromContext 取出 request id，缺失返回空串。
func RequestIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(requestIDKey{}).(string); ok {
		return v
	}
	return ""
}

// WithRequestIDSource 标记请求 ID 是客户端透传还是服务端生成，供访问审计区分。
func WithRequestIDSource(ctx context.Context, source string) context.Context {
	return context.WithValue(ctx, requestIDSourceKey{}, source)
}

// RequestIDSourceFromContext 返回 request ID 来源，缺失时返回空串。
func RequestIDSourceFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(requestIDSourceKey{}).(string); ok {
		return v
	}
	return ""
}

// WithRequestMeta 记录 method/path 供错误日志使用。
func WithRequestMeta(ctx context.Context, method, path string) context.Context {
	ctx = context.WithValue(ctx, requestMetaKeyMethod{}, method)
	return context.WithValue(ctx, requestMetaKeyPath{}, path)
}
