package httphelper

import (
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

func RequestBaseURL(r *http.Request) string {
	proto := strings.TrimSpace(r.Header.Get("X-Forwarded-Proto"))
	if proto == "" {
		proto = "http"
		if r.TLS != nil {
			proto = "https"
		}
	}
	host := strings.TrimSpace(r.Header.Get("X-Forwarded-Host"))
	if host == "" {
		host = r.Host
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

func Meta(r *http.Request) types.RequestMeta {
	ip := strings.TrimSpace(r.Header.Get("X-Forwarded-For"))
	if ip != "" {
		parts := strings.Split(ip, ",")
		ip = strings.TrimSpace(parts[0])
	} else {
		ip = strings.TrimSpace(r.Header.Get("X-Real-IP"))
	}
	if ip == "" {
		host, _, err := net.SplitHostPort(r.RemoteAddr)
		if err == nil {
			ip = host
		} else {
			ip = r.RemoteAddr
		}
	}
	return types.RequestMeta{
		IP:        ip,
		UserAgent: r.UserAgent(),
	}
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
