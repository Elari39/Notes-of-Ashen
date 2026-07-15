package middleware

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"time"

	basehandler "notes-of-ashen/internal/httphelper"
	"notes-of-ashen/internal/response"

	"github.com/zeromicro/go-zero/core/logx"
)

// AccessLogMiddleware 只记录请求元数据，不读取或输出查询参数、Header、Cookie 和请求正文。
// 可信客户端 IP 的解析规则与限流、流量统计保持一致。
type AccessLogMiddleware struct {
	forwarded basehandler.ForwardedOptions
}

func NewAccessLogMiddleware(forwarded ...basehandler.ForwardedOptions) *AccessLogMiddleware {
	options := basehandler.ForwardedOptions{}
	if len(forwarded) > 0 {
		options = forwarded[0]
	}
	return &AccessLogMiddleware{forwarded: options}
}

func (m *AccessLogMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		startedAt := time.Now()
		loggedWriter := &accessLogResponseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}
		next(loggedWriter, r)

		path := r.URL.EscapedPath()
		if path == "" {
			path = "/"
		}
		message := fmt.Sprintf(
			"[HTTP] %d - %s %s - %s - request_id=%s - duration=%s",
			loggedWriter.statusCode,
			r.Method,
			path,
			basehandler.Meta(r, m.forwarded).IP,
			response.RequestIDFromContext(r.Context()),
			time.Since(startedAt).Round(time.Microsecond),
		)
		logger := logx.WithContext(r.Context())
		if loggedWriter.statusCode >= http.StatusInternalServerError {
			logger.Error(message)
			return
		}
		logger.Info(message)
	}
}

type accessLogResponseWriter struct {
	http.ResponseWriter
	statusCode  int
	wroteHeader bool
}

func (w *accessLogResponseWriter) WriteHeader(statusCode int) {
	if w.wroteHeader {
		return
	}
	w.wroteHeader = true
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *accessLogResponseWriter) Write(body []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(body)
}

// Unwrap 允许 http.ResponseController 访问底层 writer 的流式和连接控制能力。
func (w *accessLogResponseWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

func (w *accessLogResponseWriter) Flush() {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	_ = http.NewResponseController(w.ResponseWriter).Flush()
}

func (w *accessLogResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return http.NewResponseController(w.ResponseWriter).Hijack()
}

func (w *accessLogResponseWriter) Push(target string, options *http.PushOptions) error {
	pusher, ok := w.ResponseWriter.(http.Pusher)
	if !ok {
		return http.ErrNotSupported
	}
	return pusher.Push(target, options)
}
