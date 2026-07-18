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

// ByteLength 按 UTF-8 编码后的字节数校验文本长度。数据库 TEXT/MEDIUMTEXT
// 的容量以字节计，不能仅使用 RuneCountInString 作为写入边界。
func ByteLength(value, field string, min, max int) error {
	n := len(value)
	if n < min || n > max {
		return apperrors.BadRequest(field + " size is invalid")
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
	// 保存表单时只校验显式主机，不对域名做 DNS 解析。域名可能被本机代理解析为
	// 198.18.0.0/15 Fake-IP，保存阶段解析会把正常公网 HTTPS URL 误判为本地地址。
	// 真正由服务端发起的请求必须在建连时重新解析并执行 SSRF 校验。
	host := parsed.Hostname()
	if isLocalHostname(host) {
		return apperrors.BadRequest(field + " must not point to a local address")
	}
	// 显式 IP（含 IPv6）仍在保存阶段拦截，避免直接写入本机、私网或保留地址。
	if ip := net.ParseIP(host); ip != nil {
		if IsBlockedHostIP(ip) {
			return apperrors.BadRequest(field + " must not point to a local address")
		}
	}
	return nil
}

func isLocalHostname(host string) bool {
	host = strings.TrimSuffix(strings.ToLower(strings.TrimSpace(host)), ".")
	return host == "localhost" || strings.HasSuffix(host, ".localhost")
}

func OptionalImageURL(value, field string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	if strings.HasPrefix(value, "/media/") {
		name := strings.TrimPrefix(value, "/media/")
		parts := strings.Split(name, ".")
		if len(parts) == 2 && len(parts[0]) == 64 && !strings.ContainsAny(name, "/\\") {
			for _, r := range parts[0] {
				if !strings.ContainsRune("0123456789abcdef", r) {
					return apperrors.BadRequest(field + " format is invalid")
				}
			}
			switch parts[1] {
			case "jpg", "png", "gif", "webp":
				return nil
			}
		}
		return apperrors.BadRequest(field + " format is invalid")
	}
	return OptionalHTTPURL(value, field)
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

// IsProxyFakeIP 判断 IP 是否属于常被 Clash 等透明代理用于 Fake-IP DNS 的
// 基准测试网段。该网段本身仍属于受限地址，只允许 AI 客户端在域名解析全部落入
// 此范围时保留原域名交给透明代理连接；显式 IP 地址始终拒绝。
func IsProxyFakeIP(ip net.IP) bool {
	return ip != nil && proxyFakeIPNetwork.Contains(ip)
}

var proxyFakeIPNetwork = mustParseCIDRs("198.18.0.0/15")[0]

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
