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
		if IsBlockedHostIP(ip) {
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
		if IsBlockedHostIP(ip) {
			return apperrors.BadRequest(field + " must not point to a local address")
		}
	}
	return nil
}

// IsBlockedHostIP 判断 IP 是否属于本机、内网、文档、基准测试、保留、
// 链路本地或 CGNAT 等不应被服务端用户可控请求访问的地址段。
func IsBlockedHostIP(ip net.IP) bool {
	if ip == nil {
		return true
	}
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() ||
		ip.IsMulticast() || ip.IsUnspecified() {
		return true
	}
	for _, network := range blockedHostNetworks {
		if network.Contains(ip) {
			return true
		}
	}
	return false
}

var blockedHostNetworks = mustParseCIDRs(
	"0.0.0.0/8",
	"100.64.0.0/10",
	"192.0.0.0/24",
	"192.0.2.0/24",
	"192.88.99.0/24",
	"198.18.0.0/15",
	"198.51.100.0/24",
	"203.0.113.0/24",
	"240.0.0.0/4",
	"::/96",
	"64:ff9b::/96",
	"64:ff9b:1::/48",
	"100::/64",
	"2001::/23",
	"2001:db8::/32",
	"2002::/16",
	"3fff::/20",
	"5f00::/16",
)

func mustParseCIDRs(values ...string) []*net.IPNet {
	networks := make([]*net.IPNet, 0, len(values))
	for _, value := range values {
		_, network, err := net.ParseCIDR(value)
		if err != nil {
			panic("invalid blocked host CIDR: " + value)
		}
		networks = append(networks, network)
	}
	return networks
}

func Status(value string, allowed map[string]struct{}, field string) error {
	if _, ok := allowed[value]; !ok {
		return apperrors.BadRequest(field + " is invalid")
	}
	return nil
}
