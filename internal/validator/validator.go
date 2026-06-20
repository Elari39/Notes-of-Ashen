package validator

import (
	"net"
	"net/mail"
	"net/url"
	"strings"
	"unicode/utf8"

	apperrors "notes-of-ashen/internal/errors"
)

func Required(value, field string) error {
	if strings.TrimSpace(value) == "" {
		return apperrors.BadRequest(field + " is required")
	}
	return nil
}

func Length(value, field string, min, max int) error {
	n := utf8.RuneCountInString(value)
	if n < min || n > max {
		return apperrors.BadRequest(field + " length is invalid")
	}
	return nil
}

func Email(value string) error {
	value = strings.TrimSpace(value)
	parsed, err := mail.ParseAddress(value)
	if err != nil || parsed.Address != value {
		return apperrors.BadRequest("email format is invalid")
	}
	return nil
}

func OptionalHTTPURL(value, field string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	parsed, err := url.ParseRequestURI(value)
	if err != nil || parsed.Host == "" {
		return apperrors.BadRequest(field + " format is invalid")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return apperrors.BadRequest(field + " format is invalid")
	}
	// 阻止 SSRF / 内网地址：avatarUrl/coverUrl/projects 等用户可控 URL 不得指向本机或保留地址段。
	host := parsed.Hostname()
	if host == "localhost" {
		return apperrors.BadRequest(field + " must not point to a local address")
	}
	// 先尝试直接当作 IP 解析（含 IPv6），命中即校验；否则对域名做 DNS 解析后逐一校验。
	if ip := net.ParseIP(host); ip != nil {
		if isBlockedHostIP(ip) {
			return apperrors.BadRequest(field + " must not point to a local address")
		}
		return nil
	}
	// 域名场景：解析所有 A/AAAA 记录，任一命中内网/保留段即拒绝（防御 DNS rebinding）。
	ips, lookupErr := net.LookupIP(host)
	if lookupErr != nil {
		// 解析失败不在此拦截（可能是离线/临时错误），留给实际请求处理，但记录为可疑。
		return nil
	}
	for _, ip := range ips {
		if isBlockedHostIP(ip) {
			return apperrors.BadRequest(field + " must not point to a local address")
		}
	}
	return nil
}

// isBlockedHostIP 判断 IP 是否属于本机/内网/保留/链路本地/CGNAT 等不应被用户可控 URL 访问的地址段。
func isBlockedHostIP(ip net.IP) bool {
	if ip == nil {
		return true
	}
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() ||
		ip.IsMulticast() || ip.IsUnspecified() {
		return true
	}
	// CGNAT 100.64.0.0/10（net.IP.IsPrivate 不覆盖）。
	if v4 := ip.To4(); v4 != nil {
		if v4[0] == 100 && v4[1] >= 64 && v4[1] <= 127 {
			return true
		}
	}
	return false
}

func Status(value string, allowed map[string]struct{}, field string) error {
	if _, ok := allowed[value]; !ok {
		return apperrors.BadRequest(field + " is invalid")
	}
	return nil
}
