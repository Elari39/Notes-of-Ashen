package httphelper

import (
	"crypto/tls"
	"net"
	"net/http"
	"strconv"
	"strings"

	apperrors "notes-of-ashen/internal/errors"
	"notes-of-ashen/internal/types"

	"github.com/zeromicro/go-zero/rest/httpx"
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
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 10
	}
	if size > 100 {
		size = 100
	}
	return page, size
}

func Query(r *http.Request, key string) string {
	return strings.TrimSpace(r.URL.Query().Get(key))
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

func Meta(r *http.Request, options ...ForwardedOptions) types.RequestMeta {
	ip := remoteIP(r.RemoteAddr)
	host := strings.TrimSpace(r.Host)
	if trustedProxy(r.RemoteAddr, forwardedOptions(options).TrustedProxyCIDRs) {
		if forwardedIP := forwardedClientIP(r); forwardedIP != "" {
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
	}
}

func forwardedOptions(options []ForwardedOptions) ForwardedOptions {
	if len(options) == 0 {
		return ForwardedOptions{}
	}
	return options[0]
}

func forwardedClientIP(r *http.Request) string {
	forwardedFor := strings.TrimSpace(r.Header.Get("X-Forwarded-For"))
	if forwardedFor != "" {
		for _, part := range strings.Split(forwardedFor, ",") {
			if ip := validIP(part); ip != "" {
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
