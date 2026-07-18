package httphelper

import (
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"strconv"
	"strings"

	apperrors "notes-of-ashen/internal/errors"
	"notes-of-ashen/internal/logicutil"
	"notes-of-ashen/internal/types"

	"github.com/zeromicro/go-zero/rest/httpx"
)

const (
	SmallJSONBodyLimit    int64 = 16 << 10
	StandardJSONBodyLimit int64 = 64 << 10
	ProjectsJSONBodyLimit int64 = 12 << 20
)

type idPath struct {
	ID uint64 `path:"id"`
}

type versionPath struct {
	VersionNo int `path:"versionNo"`
}

func PathID(r *http.Request) (uint64, error) {
	var path idPath
	if err := httpx.ParsePath(r, &path); err != nil || path.ID == 0 {
		return 0, apperrors.BadRequest("invalid id")
	}
	return path.ID, nil
}

func PathVersionNo(r *http.Request) (int, error) {
	var path versionPath
	if err := httpx.ParsePath(r, &path); err != nil || path.VersionNo <= 0 {
		return 0, apperrors.BadRequest("versionNo is invalid")
	}
	return path.VersionNo, nil
}

type ForwardedOptions struct {
	TrustedProxyCIDRs string
}

func RequestBaseURL(r *http.Request, options ...ForwardedOptions) string {
	proto := requestProto(r.TLS)
	host := strings.TrimSpace(r.Host)
	if trustedProxy(r.RemoteAddr, forwardedOptions(options).TrustedProxyCIDRs) {
		if forwardedProto := forwardedProto(r); forwardedProto != "" {
			proto = forwardedProto
		}
		if forwardedHost := forwardedHost(r); forwardedHost != "" {
			host = forwardedHost
		}
	}
	if host == "" {
		host = "localhost"
	}
	return strings.TrimRight(proto+"://"+host, "/")
}

func PageSize(r *http.Request) (int, int) {
	page := queryInt(r, "page", 1)
	size := queryInt(r, "size", 10)
	return logicutil.Page(page, size)
}

func Query(r *http.Request, key string) string {
	return strings.TrimSpace(r.URL.Query().Get(key))
}

func QueryInt(r *http.Request, key string, fallback int) int {
	return queryInt(r, key, fallback)
}

func QueryUint64(r *http.Request, key string) (uint64, error) {
	raw := strings.TrimSpace(r.URL.Query().Get(key))
	if raw == "" {
		return 0, nil
	}
	value, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || value == 0 {
		return 0, apperrors.BadRequest(key + " is invalid")
	}
	return value, nil
}

func Parse(r *http.Request, v interface{}) error {
	if err := httpx.Parse(r, v); err != nil {
		return apperrors.BadRequest("invalid request body or parameters")
	}
	return nil
}

// ParseLimited 为 JSON/表单类小请求提供统一的读取上限，避免在绑定结构体前
// 接收超出业务容量的大请求体。
func ParseLimited(w http.ResponseWriter, r *http.Request, v interface{}, maxBytes int64) error {
	if maxBytes > 0 {
		if r.ContentLength > maxBytes {
			return apperrors.BadRequest("request body is too large")
		}
		r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
	}
	if err := httpx.Parse(r, v); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			return apperrors.BadRequest("request body is too large")
		}
		return apperrors.BadRequest("invalid request body or parameters")
	}
	return nil
}

func Meta(r *http.Request, options ...ForwardedOptions) types.RequestMeta {
	opts := forwardedOptions(options)
	ip := remoteIP(r.RemoteAddr)
	host := strings.TrimSpace(r.Host)
	if trustedProxy(r.RemoteAddr, opts.TrustedProxyCIDRs) {
		if forwardedIP := forwardedClientIP(r, opts.TrustedProxyCIDRs); forwardedIP != "" {
			ip = forwardedIP
		}
		if forwardedHost := forwardedHost(r); forwardedHost != "" {
			host = forwardedHost
		}
	}
	if host == "" {
		host = "localhost"
	}
	return types.RequestMeta{
		IP:        ip,
		UserAgent: r.UserAgent(),
		Referrer:  r.Referer(),
		Host:      host,
		VisitorID: strings.TrimSpace(r.Header.Get("X-Visitor-Id")),
	}
}

func forwardedOptions(options []ForwardedOptions) ForwardedOptions {
	if len(options) == 0 {
		return ForwardedOptions{}
	}
	return options[0]
}

// forwardedClientIP 从 XFF 链中取最右侧“不可信”来源 IP：从右向左遍历，
// 跳过所有属于可信代理 CIDR 的分段，返回第一个不可信 IP。
// XFF 全部为可信代理时回退到 X-Real-IP。
func forwardedClientIP(r *http.Request, trustedCIDRs string) string {
	forwardedFor := strings.TrimSpace(r.Header.Get("X-Forwarded-For"))
	if forwardedFor != "" {
		parts := strings.Split(forwardedFor, ",")
		for i := len(parts) - 1; i >= 0; i-- {
			ip := validIP(parts[i])
			if ip == "" {
				continue
			}
			if !trustedProxy(ip, trustedCIDRs) {
				return ip
			}
		}
	}
	return validIP(r.Header.Get("X-Real-IP"))
}

func forwardedProto(r *http.Request) string {
	proto := strings.ToLower(strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")))
	if proto == "http" || proto == "https" {
		return proto
	}
	return ""
}

func forwardedHost(r *http.Request) string {
	host := strings.TrimSpace(r.Header.Get("X-Forwarded-Host"))
	if host == "" || strings.ContainsAny(host, " \t\r\n") {
		return ""
	}
	return host
}

func requestProto(tlsState *tls.ConnectionState) string {
	if tlsState != nil {
		return "https"
	}
	return "http"
}

func trustedProxy(remoteAddr, cidrs string) bool {
	ip := net.ParseIP(remoteIP(remoteAddr))
	if ip == nil {
		return false
	}
	for _, raw := range strings.Split(cidrs, ",") {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		_, network, err := net.ParseCIDR(raw)
		if err == nil && network.Contains(ip) {
			return true
		}
		if trustedIP := net.ParseIP(raw); trustedIP != nil && trustedIP.Equal(ip) {
			return true
		}
	}
	return false
}

func remoteIP(remoteAddr string) string {
	host, _, err := net.SplitHostPort(strings.TrimSpace(remoteAddr))
	if err == nil {
		return host
	}
	return strings.TrimSpace(remoteAddr)
}

func validIP(value string) string {
	value = strings.TrimSpace(value)
	ip := net.ParseIP(value)
	if ip == nil {
		return ""
	}
	return ip.String()
}

func queryInt(r *http.Request, key string, fallback int) int {
	raw := strings.TrimSpace(r.URL.Query().Get(key))
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return value
}
