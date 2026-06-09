package traffic

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/url"
	"strings"
	"time"

	apperrors "notes-of-ashen/internal/errors"
	"notes-of-ashen/internal/security"
	"notes-of-ashen/internal/svc"
	"notes-of-ashen/internal/types"
	"notes-of-ashen/internal/validator"
	"notes-of-ashen/model"

	"github.com/redis/go-redis/v9"
)

const trafficRetention = 45 * 24 * time.Hour

func Visit(ctx context.Context, svcCtx *svc.ServiceContext, req types.TrafficVisitReq, meta types.RequestMeta) error {
	req.Path = strings.TrimSpace(req.Path)
	req.RouteType = strings.TrimSpace(req.RouteType)
	req.Referrer = strings.TrimSpace(req.Referrer)
	if req.Referrer == "" {
		req.Referrer = strings.TrimSpace(meta.Referrer)
	}
	if err := validator.Length(req.Path, "path", 1, 255); err != nil {
		return err
	}
	if err := validator.Length(req.RouteType, "routeType", 1, 64); err != nil {
		return err
	}
	if len(req.Referrer) > 512 {
		return apperrors.BadRequest("referrer length is invalid")
	}
	if !isPublicTrafficPath(req.Path) {
		return nil
	}

	now := time.Now()
	date := now.Format("2006-01-02")
	sourceType, sourceName := classifyReferer(req.Referrer, meta.Host)
	visitorHash := visitorDailyHash(date, meta.IP, meta.UserAgent)
	geo := lookupGeo(svcCtx, meta.IP)

	if err := svcCtx.Store.RecordTraffic(ctx, model.TrafficRecord{
		Date:        date,
		VisitorHash: visitorHash,
		ArticleID:   req.ArticleID,
		SourceType:  sourceType,
		SourceName:  sourceName,
		Geo: model.GeoLocation{
			CountryCode: geo.CountryCode,
			CountryName: geo.CountryName,
			RegionName:  geo.RegionName,
			CityName:    geo.CityName,
		},
	}); err != nil {
		return err
	}
	recordRedisTraffic(ctx, svcCtx.Redis, date, visitorHash, sourceType, sourceName)
	return nil
}

func lookupGeo(svcCtx *svc.ServiceContext, ip string) model.GeoLocation {
	if svcCtx == nil || svcCtx.GeoIP == nil {
		return model.GeoLocation{
			CountryCode: "Unknown",
			CountryName: "Unknown",
			RegionName:  "Unknown",
			CityName:    "Unknown",
		}
	}
	location := svcCtx.GeoIP.Lookup(ip)
	return model.GeoLocation{
		CountryCode: location.CountryCode,
		CountryName: location.CountryName,
		RegionName:  location.RegionName,
		CityName:    location.CityName,
	}
}

func recordRedisTraffic(ctx context.Context, redisClient *redis.Client, date, visitorHash, sourceType, sourceName string) {
	if redisClient == nil {
		return
	}
	sourceKey := sourceType + ":" + sourceName
	pipe := redisClient.Pipeline()
	pvKey := security.TrafficPVKey(date)
	uvKey := security.TrafficUVKey(date)
	refererKey := security.TrafficRefererKey(date)
	pipe.Incr(ctx, pvKey)
	pipe.PFAdd(ctx, uvKey, visitorHash)
	pipe.ZIncrBy(ctx, refererKey, 1, sourceKey)
	pipe.Expire(ctx, pvKey, trafficRetention)
	pipe.Expire(ctx, uvKey, trafficRetention)
	pipe.Expire(ctx, refererKey, trafficRetention)
	_, _ = pipe.Exec(ctx)
}

func isPublicTrafficPath(path string) bool {
	path = strings.TrimSpace(path)
	if path == "" {
		return false
	}
	blockedPrefixes := []string{"/admin", "/login", "/register", "/profile", "/forgot-password"}
	for _, prefix := range blockedPrefixes {
		if path == prefix || strings.HasPrefix(path, prefix+"/") || strings.HasPrefix(path, prefix+"?") {
			return false
		}
	}
	return true
}

func visitorDailyHash(date, ip, userAgent string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(date) + "|" + strings.TrimSpace(ip) + "|" + strings.TrimSpace(userAgent)))
	return hex.EncodeToString(sum[:])
}

func classifyReferer(referrer, currentHost string) (string, string) {
	referrer = strings.TrimSpace(referrer)
	if referrer == "" {
		return "direct", "direct"
	}
	if strings.HasPrefix(referrer, "/") {
		return "internal", "site"
	}
	parsed, err := url.Parse(referrer)
	if err != nil || parsed.Host == "" {
		return "external", "unknown"
	}
	host := normalizeHost(parsed.Host)
	if host == "" {
		return "external", "unknown"
	}
	if sameHost(host, normalizeHost(currentHost)) {
		return "internal", "site"
	}
	if engine := searchEngineName(host); engine != "" {
		return "search", engine
	}
	return "external", host
}

func normalizeHost(host string) string {
	host = strings.ToLower(strings.TrimSpace(host))
	if strings.Contains(host, "@") {
		if parsed, err := url.Parse("//" + host); err == nil {
			host = parsed.Host
		}
	}
	if parsed, err := url.Parse("//" + host); err == nil && parsed.Hostname() != "" {
		host = parsed.Hostname()
	}
	return strings.TrimPrefix(host, "www.")
}

func sameHost(a, b string) bool {
	return a != "" && b != "" && a == b
}

func searchEngineName(host string) string {
	switch {
	case containsLabelDomain(host, "google"):
		return "google"
	case hasDomain(host, "bing.com"):
		return "bing"
	case hasDomain(host, "baidu.com"):
		return "baidu"
	case hasDomain(host, "sogou.com"):
		return "sogou"
	case hasDomain(host, "so.com") || hasDomain(host, "360.cn"):
		return "360"
	case hasDomain(host, "duckduckgo.com"):
		return "duckduckgo"
	case hasDomain(host, "yahoo.com"):
		return "yahoo"
	case containsLabelDomain(host, "yandex"):
		return "yandex"
	default:
		return ""
	}
}

func hasDomain(host, domain string) bool {
	return host == domain || strings.HasSuffix(host, "."+domain)
}

func containsLabelDomain(host, label string) bool {
	return host == label || strings.HasPrefix(host, label+".") || strings.Contains(host, "."+label+".")
}
