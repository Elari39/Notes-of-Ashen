package traffic

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/url"
	"strconv"
	"strings"
	"time"

	apperrors "notes-of-ashen/internal/errors"
	"notes-of-ashen/internal/security"
	"notes-of-ashen/internal/svc"
	"notes-of-ashen/internal/types"
	"notes-of-ashen/internal/validator"

	"github.com/zeromicro/go-zero/core/logx"
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
	// 规范化 path，避免 "/admin/../articles" 之类绕过黑名单。
	normalizedPath := normalizeTrafficPath(req.Path)
	routeType, contentPath, ok := canonicalContentRoute(normalizedPath, req.ArticleID)
	if !ok || !isPublicTrafficPath(contentPath) {
		return nil
	}
	// articleId 来自 body 任意值，校验其指向真实已发布文章，防止伪造刷统计。
	if req.ArticleID > 0 {
		article, err := svcCtx.Store.FindArticle(ctx, req.ArticleID)
		if err != nil {
			if errors.Is(err, model.ErrNotFound) {
				return nil
			}
			return err
		}
		if !model.IsArticlePubliclyVisible(*article, time.Now()) {
			return nil
		}
	}

	now := time.Now()
	date := now.Format("2006-01-02")
	sourceType, sourceName := classifyReferer(req.Referrer, meta.Host)
	// UV 使用服务端观察到的 IP/UA 与前端匿名 visitor ID 的组合哈希：
	// 同一 NAT 下不同浏览器不会被强行合并，清 Cookie 后仍会回退到 IP/UA，
	// 且数据库和 Redis 不保存原始 visitor ID、IP 或 User-Agent。
	visitorHash := visitorDailyHash(date, meta.IP, meta.UserAgent, meta.VisitorID)

	if err := svcCtx.Store.RecordTraffic(ctx, model.TrafficRecord{
		Date:        date,
		VisitorHash: visitorHash,
		ArticleID:   req.ArticleID,
		RouteType:   routeType,
		Path:        contentPath,
		SourceType:  sourceType,
		SourceName:  sourceName,
	}); err != nil {
		return err
	}
	recordRedisTraffic(ctx, svcCtx.Redis, date, visitorHash, sourceType, sourceName)
	cleanupVisitorRows(ctx, svcCtx, now)
	return nil
}

func cleanupVisitorRows(ctx context.Context, svcCtx *svc.ServiceContext, now time.Time) {
	if svcCtx.Redis == nil {
		return
	}
	key := "traffic:visitor-cleanup:" + now.Format("2006-01-02")
	ok, err := svcCtx.Redis.SetNX(ctx, key, 1, 48*time.Hour).Result()
	if err != nil || !ok {
		return
	}
	before := now.AddDate(0, 0, -7).Format("2006-01-02")
	if err := svcCtx.Store.CleanupTrafficVisitors(ctx, before); err != nil {
		logx.Errorf("cleanup traffic visitor rows failed: %v", err)
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
	if _, err := pipe.Exec(ctx); err != nil {
		logx.Errorf("traffic redis pipeline exec failed: %v", err)
	}
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

// normalizeTrafficPath 规范化 path：去除首尾空白与斜杠后补回单个前导斜杠，
// 避免 "/admin/" "/admin/../articles" 等变体绕过黑名单前缀匹配。
func normalizeTrafficPath(path string) string {
	path = strings.TrimSpace(path)
	if index := strings.IndexByte(path, '?'); index >= 0 {
		path = path[:index]
	}
	path = strings.Trim(path, "/")
	if path == "" {
		return "/"
	}
	return "/" + path
}

func canonicalContentRoute(path string, articleID uint64) (string, string, bool) {
	if articleID > 0 {
		return "article", "/article/" + strconv.FormatUint(articleID, 10), true
	}
	switch path {
	case "/", "/archive", "/search", "/projects":
		return "page", path, true
	default:
		return "", "", false
	}
}

func visitorDailyHash(date, ip, userAgent string, visitorIDs ...string) string {
	input := strings.TrimSpace(date) + "|" + strings.TrimSpace(ip) + "|" + strings.TrimSpace(userAgent)
	if len(visitorIDs) > 0 && strings.TrimSpace(visitorIDs[0]) != "" {
		input += "|" + strings.TrimSpace(visitorIDs[0])
	}
	sum := sha256.Sum256([]byte(input))
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
	// 限制 sourceName 长度，避免超长 host 撑大 traffic_referer_stats 与 Redis ZSet member。
	if len(host) > 128 {
		host = host[:128]
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
