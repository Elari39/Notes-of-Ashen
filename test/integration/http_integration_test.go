//go:build integration

package integration

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/redis/go-redis/v9"
)

const (
	webBaseURLEnv       = "E2E_WEB_BASE_URL"
	apiBaseURLEnv       = "E2E_API_BASE_URL"
	redisURLEnv         = "E2E_REDIS_URL"
	redisContainerIDEnv = "E2E_REDIS_CONTAINER_ID"
	mysqlDSNEnv         = "E2E_MYSQL_DSN"
	mysqlRootDSNEnv     = "E2E_MYSQL_ROOT_DSN"
	refreshCookie       = "noa_refresh_token"
	verificationCode    = "917263"
	backupPassphrase    = "integration-backup-passphrase"
)

var uniqueSequence atomic.Uint64

type testEnvironment struct {
	webBaseURL       string
	apiBaseURL       string
	redisURL         string
	redisContainerID string
	mysqlDSN         string
	mysqlRootDSN     string
}

type harness struct {
	webBaseURL       string
	apiBaseURL       string
	client           *http.Client
	redis            *redis.Client
	redisContainerID string
	appDB            *sql.DB
	rootDB           *sql.DB
}

type apiEnvelope struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

type apiResult struct {
	status   int
	body     []byte
	header   http.Header
	cookies  []*http.Cookie
	envelope apiEnvelope
}

type credentials struct {
	account  string
	email    string
	password string
}

type session struct {
	credentials
	accessToken  string
	refreshToken string
}

type tokenPair struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	TokenType    string `json:"tokenType"`
	ExpiresIn    int64  `json:"expiresIn"`
}

type user struct {
	ID      uint64 `json:"id"`
	Account string `json:"account"`
	Role    string `json:"role"`
}

type mediaAsset struct {
	ID         uint64 `json:"id"`
	StorageKey string `json:"storageKey"`
	URL        string `json:"url"`
	MIMEType   string `json:"mimeType"`
	SHA256     string `json:"sha256"`
}

type article struct {
	ID       uint64 `json:"id"`
	Title    string `json:"title"`
	Slug     string `json:"slug"`
	Content  string `json:"content"`
	CoverURL string `json:"coverUrl"`
	Status   string `json:"status"`
}

type backupRestoreResponse struct {
	Users    int      `json:"users"`
	Articles int      `json:"articles"`
	Media    int      `json:"media"`
	Warnings []string `json:"warnings"`
}

type healthReport struct {
	Status string `json:"status"`
	Checks map[string]struct {
		Status string `json:"status"`
		Error  string `json:"error"`
	} `json:"checks"`
}

func TestCoreHealthAndSchema(t *testing.T) {
	h := newHarness(t)

	web, err := h.request(context.Background(), http.MethodGet, h.webURL("/"), nil, nil, "", "")
	if err != nil {
		t.Fatalf("请求前端首页失败: %v", err)
	}
	if web.status != http.StatusOK {
		t.Fatalf("前端首页状态码 = %d，期望 %d", web.status, http.StatusOK)
	}
	proxiedAPI, err := h.request(context.Background(), http.MethodGet, h.webURL("/api/v1/site/settings"), nil, nil, "", "")
	if err != nil {
		t.Fatalf("通过前端 Nginx 代理访问 API 失败: %v", err)
	}
	expectAPISuccess(t, proxiedAPI)

	webReady, err := h.request(context.Background(), http.MethodGet, h.webURL("/healthz"), nil, nil, "", "")
	if err != nil {
		t.Fatalf("通过前端 Nginx 请求 readiness 失败: %v", err)
	}
	if webReady.status != http.StatusOK || !strings.Contains(webReady.header.Get("Content-Type"), "application/json") {
		t.Fatalf("Web readiness 状态或类型错误: status=%d content-type=%q", webReady.status, webReady.header.Get("Content-Type"))
	}
	var webHealth healthReport
	if err := json.Unmarshal(webReady.body, &webHealth); err != nil || webHealth.Status != "ok" {
		t.Fatalf("Web readiness 未返回后端健康报告: error=%v body=%s", err, webReady.body)
	}
	webLive, err := h.request(context.Background(), http.MethodGet, h.webURL("/livez"), nil, nil, "", "")
	if err != nil {
		t.Fatalf("请求 Web liveness 失败: %v", err)
	}
	if webLive.status != http.StatusNoContent || len(webLive.body) != 0 {
		t.Fatalf("Web liveness 响应错误: status=%d body=%q", webLive.status, webLive.body)
	}

	result, err := h.request(context.Background(), http.MethodGet, h.apiURL("/healthz"), nil, nil, "", "")
	if err != nil {
		t.Fatalf("请求健康检查失败: %v", err)
	}
	if result.status != http.StatusOK {
		t.Fatalf("健康检查状态码 = %d，期望 %d", result.status, http.StatusOK)
	}

	var report healthReport
	if err := json.Unmarshal(result.body, &report); err != nil {
		t.Fatalf("解析健康检查响应失败: %v", err)
	}
	if report.Status != "ok" {
		t.Fatalf("健康检查整体状态 = %q，期望 ok", report.Status)
	}
	for _, name := range []string{"db", "redis", "schema"} {
		check, ok := report.Checks[name]
		if !ok {
			t.Fatalf("健康检查缺少 %s 检查项", name)
		}
		if check.Status != "up" {
			t.Fatalf("健康检查 %s 状态 = %q，错误 = %q", name, check.Status, check.Error)
		}
	}

	const restoreColumn = "ALTER TABLE media_assets ADD COLUMN alt_text VARCHAR(255) NOT NULL DEFAULT '' AFTER height"
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if _, err := h.rootDB.ExecContext(ctx, "ALTER TABLE media_assets DROP COLUMN alt_text"); err != nil {
		t.Fatalf("移除必需 schema 字段失败: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		if _, err := h.rootDB.ExecContext(cleanupCtx, restoreColumn); err != nil {
			t.Errorf("恢复必需 schema 字段失败: %v", err)
		}
	})

	degraded, err := h.request(context.Background(), http.MethodGet, h.apiURL("/healthz"), nil, nil, "", "")
	if err != nil {
		t.Fatalf("schema 缺失后的健康检查请求失败: %v", err)
	}
	if degraded.status != http.StatusServiceUnavailable {
		t.Fatalf("schema 缺失后的健康检查状态码 = %d，期望 %d", degraded.status, http.StatusServiceUnavailable)
	}
	var degradedReport healthReport
	if err := json.Unmarshal(degraded.body, &degradedReport); err != nil {
		t.Fatalf("解析 schema 缺失后的健康检查响应失败: %v", err)
	}
	if degradedReport.Status != "degraded" || degradedReport.Checks["schema"].Status != "down" {
		t.Fatalf("schema 缺失后的健康检查未降级: %#v", degradedReport)
	}
}

func TestCoreAuthRefreshRotationAndPermission(t *testing.T) {
	h := newHarness(t)
	admin := h.registerUser(t, "admin")

	unauthorized, err := h.jsonRequest(context.Background(), http.MethodPost, "/api/v1/articles", map[string]any{
		"title":   "未授权文章",
		"slug":    h.uniqueSlug("unauthorized"),
		"content": "这次请求不应通过鉴权。",
		"status":  "draft",
	}, "", "")
	if err != nil {
		t.Fatalf("发送未授权文章创建请求失败: %v", err)
	}
	expectAPIStatus(t, unauthorized, http.StatusUnauthorized)

	regular := h.registerUser(t, "user")
	adminOnly, err := h.jsonRequest(context.Background(), http.MethodGet, "/api/v1/admin/users", nil, regular.accessToken, "")
	if err != nil {
		t.Fatalf("普通用户访问管理员接口失败: %v", err)
	}
	expectAPIStatus(t, adminOnly, http.StatusForbidden)

	contentManagerOnly, err := h.jsonRequest(context.Background(), http.MethodPost, "/api/v1/articles", map[string]any{
		"title":   "普通用户文章",
		"slug":    h.uniqueSlug("regular-user"),
		"content": "普通用户不应有文章创建权限。",
		"status":  "draft",
	}, regular.accessToken, "")
	if err != nil {
		t.Fatalf("普通用户创建文章请求失败: %v", err)
	}
	expectAPIStatus(t, contentManagerOnly, http.StatusForbidden)

	rotated, err := h.jsonRequest(context.Background(), http.MethodPost, "/api/v1/auth/refresh", map[string]any{}, "", admin.refreshToken)
	if err != nil {
		t.Fatalf("刷新令牌请求失败: %v", err)
	}
	expectAPISuccess(t, rotated)
	rotatedPair := decodeData[tokenPair](t, rotated)
	if rotatedPair.AccessToken == "" {
		t.Fatal("刷新响应未返回 accessToken")
	}
	if rotatedPair.RefreshToken != "" {
		t.Fatal("刷新响应不应在 JSON 中暴露 refreshToken")
	}
	newRefreshToken := cookieValue(rotated.cookies, refreshCookie)
	if newRefreshToken == "" {
		t.Fatal("刷新响应未设置新的 HttpOnly refresh cookie")
	}
	assertRefreshCookie(t, findCookie(rotated.cookies, refreshCookie))
	if newRefreshToken == admin.refreshToken {
		t.Fatal("刷新令牌未完成轮换")
	}

	reused, err := h.jsonRequest(context.Background(), http.MethodPost, "/api/v1/auth/refresh", map[string]any{}, "", admin.refreshToken)
	if err != nil {
		t.Fatalf("复用旧刷新令牌请求失败: %v", err)
	}
	expectAPIStatus(t, reused, http.StatusUnauthorized)

	adminAfterRotation, err := h.jsonRequest(context.Background(), http.MethodGet, "/api/v1/admin/users", nil, rotatedPair.AccessToken, "")
	if err != nil {
		t.Fatalf("使用新访问令牌访问管理员接口失败: %v", err)
	}
	expectAPISuccess(t, adminAfterRotation)
}

func TestCoreArticleAndRealMedia(t *testing.T) {
	h := newHarness(t)
	admin := h.registerUser(t, "mediaadmin")
	image := onePixelPNG(t)
	asset := h.uploadMedia(t, admin.accessToken, image)
	h.assertMedia(t, asset, image)

	content := fmt.Sprintf("# 真实媒体\n\n![像素图](%s)\n", asset.URL)
	created := h.createPublishedArticle(t, admin.accessToken, "真实媒体文章", content, asset.URL)
	if created.CoverURL != asset.URL {
		t.Fatalf("文章封面地址 = %q，期望 %q", created.CoverURL, asset.URL)
	}
	if !strings.Contains(created.Content, asset.URL) {
		t.Fatalf("文章内容未保留真实媒体地址 %q", asset.URL)
	}

	detail := h.publicArticle(t, created.ID)
	if detail.Title != created.Title || detail.CoverURL != asset.URL || !strings.Contains(detail.Content, asset.URL) {
		t.Fatalf("公开文章详情未包含预期文章和媒体关联: %#v", detail)
	}
}

func TestCoreBackupRestore(t *testing.T) {
	h := newHarness(t)
	admin := h.registerUser(t, "backupadmin")
	image := onePixelPNG(t)
	asset := h.uploadMedia(t, admin.accessToken, image)
	backupArticle := h.createPublishedArticle(t, admin.accessToken, "备份中的文章", "备份恢复应保留这篇文章。", asset.URL)

	archive := h.exportBackup(t, admin.accessToken, admin.password)
	if len(archive) == 0 {
		t.Fatal("备份导出内容为空")
	}

	transient := h.createPublishedArticle(t, admin.accessToken, "恢复前临时文章", "这篇文章不应出现在恢复后的数据中。", "")
	result := h.restoreBackup(t, admin.accessToken, archive, admin.password)
	expectAPISuccess(t, result)
	restored := decodeData[backupRestoreResponse](t, result)
	if restored.Users != 1 || restored.Articles != 1 || restored.Media != 1 {
		t.Fatalf("备份恢复计数 = %+v，期望 users=1 articles=1 media=1", restored)
	}

	oldToken, err := h.jsonRequest(context.Background(), http.MethodGet, "/api/v1/users/me", nil, admin.accessToken, "")
	if err != nil {
		t.Fatalf("验证恢复后旧访问令牌失败: %v", err)
	}
	expectAPIStatus(t, oldToken, http.StatusUnauthorized)

	loginCtx, loginCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer loginCancel()
	h.waitPastAccessTokenCutoff(t, loginCtx)
	newSession := h.login(t, admin.credentials)
	meAfterLogin, err := h.jsonRequest(context.Background(), http.MethodGet, "/api/v1/users/me", nil, newSession.accessToken, "")
	if err != nil {
		t.Fatalf("恢复后重新登录用户访问失败: %v", err)
	}
	expectAPISuccess(t, meAfterLogin)
	if got := decodeData[user](t, meAfterLogin); got.Account != admin.account || got.Role != "admin" {
		t.Fatalf("恢复后重新登录用户不匹配: %#v", got)
	}

	articleAfterRestore := h.publicArticle(t, backupArticle.ID)
	if articleAfterRestore.Title != backupArticle.Title || articleAfterRestore.CoverURL != asset.URL {
		t.Fatalf("恢复后的文章不匹配: %#v", articleAfterRestore)
	}
	h.assertMedia(t, asset, image)

	missing, err := h.jsonRequest(context.Background(), http.MethodGet, fmt.Sprintf("/api/v1/articles/%d", transient.ID), nil, "", "")
	if err != nil {
		t.Fatalf("验证临时文章不存在失败: %v", err)
	}
	expectAPIStatus(t, missing, http.StatusNotFound)
}

func TestExtendedConcurrentRegistrationAndRefreshRotation(t *testing.T) {
	h := newHarness(t)
	first := h.newCredentials("concurrentone")
	second := h.newCredentials("concurrenttwo")
	h.seedRegistrationCode(t, first.email)
	h.seedRegistrationCode(t, second.email)

	type registerOutcome struct {
		session session
		result  apiResult
		err     error
	}
	start := make(chan struct{})
	outcomes := make(chan registerOutcome, 2)
	var registrations sync.WaitGroup
	for _, item := range []credentials{first, second} {
		credentials := item
		registrations.Add(1)
		go func() {
			defer registrations.Done()
			<-start
			s, result, err := h.registerRequest(context.Background(), credentials)
			outcomes <- registerOutcome{session: s, result: result, err: err}
		}()
	}
	close(start)
	registrations.Wait()
	close(outcomes)

	sessions := make([]session, 0, 2)
	for outcome := range outcomes {
		if outcome.err != nil {
			t.Fatalf("并发注册请求失败: %v", outcome.err)
		}
		expectAPISuccess(t, outcome.result)
		h.assertSession(t, outcome.session)
		sessions = append(sessions, outcome.session)
	}
	if len(sessions) != 2 {
		t.Fatalf("并发注册成功会话数 = %d，期望 2", len(sessions))
	}

	var admins int
	if err := h.appDB.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM users WHERE role = 'admin'").Scan(&admins); err != nil {
		t.Fatalf("查询管理员数量失败: %v", err)
	}
	if admins != 1 {
		t.Fatalf("并发首注册后的管理员数量 = %d，期望 1", admins)
	}

	var adminSession session
	for _, candidate := range sessions {
		me, err := h.jsonRequest(context.Background(), http.MethodGet, "/api/v1/users/me", nil, candidate.accessToken, "")
		if err != nil {
			t.Fatalf("查询并发注册用户资料失败: %v", err)
		}
		expectAPISuccess(t, me)
		if decodeData[user](t, me).Role == "admin" {
			adminSession = candidate
		}
	}
	if adminSession.accessToken == "" {
		t.Fatal("未找到并发首注册产生的管理员会话")
	}

	type refreshOutcome struct {
		result apiResult
		err    error
	}
	refreshStart := make(chan struct{})
	refreshes := make(chan refreshOutcome, 2)
	var refreshGroup sync.WaitGroup
	for range 2 {
		refreshGroup.Add(1)
		go func() {
			defer refreshGroup.Done()
			<-refreshStart
			result, err := h.jsonRequest(context.Background(), http.MethodPost, "/api/v1/auth/refresh", map[string]any{}, "", adminSession.refreshToken)
			refreshes <- refreshOutcome{result: result, err: err}
		}()
	}
	close(refreshStart)
	refreshGroup.Wait()
	close(refreshes)

	successes := 0
	unauthorized := 0
	for outcome := range refreshes {
		if outcome.err != nil {
			t.Fatalf("并发刷新请求失败: %v", outcome.err)
		}
		switch outcome.result.status {
		case http.StatusOK:
			expectAPISuccess(t, outcome.result)
			successes++
		case http.StatusUnauthorized:
			unauthorized++
		default:
			t.Fatalf("并发刷新返回非预期状态码 %d（code=%d, message=%q）", outcome.result.status, outcome.result.envelope.Code, outcome.result.envelope.Message)
		}
	}
	if successes != 1 || unauthorized != 1 {
		t.Fatalf("并发刷新结果 success=%d unauthorized=%d，期望各为 1", successes, unauthorized)
	}
}

func TestExtendedRedisFailClosed(t *testing.T) {
	if strings.TrimSpace(os.Getenv(redisContainerIDEnv)) == "" {
		t.Skip("未设置 E2E_REDIS_CONTAINER_ID，跳过 Redis 停止故障注入测试")
	}
	h := newHarness(t)
	credentials := h.newCredentials("redispaused")
	h.seedRegistrationCode(t, credentials.email)

	stopCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := h.dockerRedis(stopCtx, "stop"); err != nil {
		t.Fatalf("停止 Redis 以验证 fail-closed 失败: %v", err)
	}
	redisStopped := true
	t.Cleanup(func() {
		if !redisStopped {
			return
		}
		if err := h.startRedisAndWait(); err != nil {
			t.Errorf("测试失败后恢复 Redis 失败: %v", err)
		}
	})

	requestCtx, requestCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer requestCancel()
	result, requestErr := h.jsonRequest(requestCtx, http.MethodPost, "/api/v1/auth/register", registrationPayload(credentials), "", "")
	if err := h.startRedisAndWait(); err != nil {
		t.Fatalf("恢复 Redis 失败: %v", err)
	}
	redisStopped = false
	if requestErr != nil {
		t.Fatalf("Redis 停止期间注册请求失败: %v", requestErr)
	}
	expectAPIStatus(t, result, http.StatusServiceUnavailable)
	if result.envelope.Code != 50300 {
		t.Fatalf("Redis 不可用时注册错误码 = %d，期望 50300", result.envelope.Code)
	}
	var users int
	if err := h.appDB.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM users WHERE account = ?", credentials.account).Scan(&users); err != nil {
		t.Fatalf("查询 Redis 停止期间注册结果失败: %v", err)
	}
	if users != 0 {
		t.Fatalf("Redis 限流不可用时仍创建了 %d 个用户，期望 0", users)
	}
}

func TestExtendedBackupDatabaseStageFailure(t *testing.T) {
	h := newHarness(t)
	admin := h.registerUser(t, "restorefailure")
	backupArticle := h.createPublishedArticle(t, admin.accessToken, "数据库失败备份文章", "这篇文章进入备份。", "")
	archive := h.exportBackup(t, admin.accessToken, admin.password)
	sentinel := h.createPublishedArticle(t, admin.accessToken, "数据库失败哨兵文章", "恢复事务失败后这篇文章必须保留。", "")

	const trigger = "e2e_fail_restore_users"
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if _, err := h.rootDB.ExecContext(ctx, "DROP TRIGGER IF EXISTS "+trigger); err != nil {
		t.Fatalf("清理旧的恢复失败触发器失败: %v", err)
	}
	if _, err := h.rootDB.ExecContext(ctx, "CREATE TRIGGER "+trigger+" BEFORE INSERT ON users FOR EACH ROW SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'e2e restore database stage failure'"); err != nil {
		t.Fatalf("创建恢复数据库阶段失败触发器失败: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		if _, err := h.rootDB.ExecContext(cleanupCtx, "DROP TRIGGER IF EXISTS "+trigger); err != nil {
			t.Errorf("清理恢复失败触发器失败: %v", err)
		}
	})

	result := h.restoreBackup(t, admin.accessToken, archive, admin.password)
	expectAPIStatus(t, result, http.StatusInternalServerError)
	if result.envelope.Code != 50000 {
		t.Fatalf("数据库阶段失败响应错误码 = %d，期望 50000", result.envelope.Code)
	}

	if got := h.publicArticle(t, backupArticle.ID); got.Title != backupArticle.Title {
		t.Fatalf("数据库恢复失败后备份文章未保留: %#v", got)
	}
	if got := h.publicArticle(t, sentinel.ID); got.Title != sentinel.Title {
		t.Fatalf("数据库恢复失败后哨兵文章未保留: %#v", got)
	}
	var articleCount int
	if err := h.appDB.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM articles").Scan(&articleCount); err != nil {
		t.Fatalf("查询数据库阶段失败后的文章数量失败: %v", err)
	}
	if articleCount != 2 {
		t.Fatalf("数据库阶段失败后文章数量 = %d，期望 2（事务应回滚）", articleCount)
	}
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	env := loadEnvironment(t)
	rootDB := openDatabase(t, env.mysqlRootDSN, "root")
	appDB := openDatabase(t, env.mysqlDSN, "application")
	options, err := redis.ParseURL(env.redisURL)
	if err != nil {
		t.Fatalf("解析 E2E_REDIS_URL 失败: %v", err)
	}
	redisClient := redis.NewClient(options)
	t.Cleanup(func() { _ = redisClient.Close() })
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		t.Fatalf("连接 E2E Redis 失败: %v", err)
	}

	h := &harness{
		webBaseURL:       env.webBaseURL,
		apiBaseURL:       env.apiBaseURL,
		client:           &http.Client{Timeout: 45 * time.Second},
		redis:            redisClient,
		redisContainerID: env.redisContainerID,
		appDB:            appDB,
		rootDB:           rootDB,
	}
	h.resetState(t)
	return h
}

func loadEnvironment(t *testing.T) testEnvironment {
	t.Helper()
	values := map[string]string{
		webBaseURLEnv:   strings.TrimRight(strings.TrimSpace(os.Getenv(webBaseURLEnv)), "/"),
		apiBaseURLEnv:   strings.TrimRight(strings.TrimSpace(os.Getenv(apiBaseURLEnv)), "/"),
		redisURLEnv:     strings.TrimSpace(os.Getenv(redisURLEnv)),
		mysqlDSNEnv:     strings.TrimSpace(os.Getenv(mysqlDSNEnv)),
		mysqlRootDSNEnv: strings.TrimSpace(os.Getenv(mysqlRootDSNEnv)),
	}
	for name, value := range values {
		if value == "" {
			t.Skipf("未设置 %s，跳过真实 HTTP 集成测试", name)
		}
	}
	return testEnvironment{
		webBaseURL:       values[webBaseURLEnv],
		apiBaseURL:       values[apiBaseURLEnv],
		redisURL:         values[redisURLEnv],
		redisContainerID: strings.TrimSpace(os.Getenv(redisContainerIDEnv)),
		mysqlDSN:         values[mysqlDSNEnv],
		mysqlRootDSN:     values[mysqlRootDSNEnv],
	}
}

func openDatabase(t *testing.T, dsn, label string) *sql.DB {
	t.Helper()
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatalf("打开 %s MySQL 连接失败: %v", label, err)
	}
	t.Cleanup(func() { _ = db.Close() })
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("连接 %s MySQL 失败: %v", label, err)
	}
	return db
}

func (h *harness) resetState(t *testing.T) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	h.waitPastAccessTokenCutoff(t, ctx)
	if err := h.redis.FlushDB(ctx).Err(); err != nil {
		t.Fatalf("清空 E2E Redis 数据失败: %v", err)
	}
	for _, statement := range []string{
		"DELETE FROM refresh_tokens",
		"DELETE FROM operation_logs",
		"DELETE FROM traffic_content_daily_visitors",
		"DELETE FROM traffic_content_daily_stats",
		"DELETE FROM traffic_referer_stats",
		"DELETE FROM traffic_daily_visitors",
		"DELETE FROM traffic_daily_stats",
		"DELETE FROM article_likes",
		"DELETE FROM project_tags",
		"DELETE FROM article_tags",
		"DELETE FROM article_versions",
		"DELETE FROM projects",
		"DELETE FROM media_assets",
		"DELETE FROM articles",
		"DELETE FROM categories",
		"DELETE FROM tags",
		"DELETE FROM site_settings",
		"DELETE FROM users",
	} {
		if _, err := h.rootDB.ExecContext(ctx, statement); err != nil {
			t.Fatalf("重置 E2E MySQL 数据失败: %v", err)
		}
	}
}

func (h *harness) waitPastAccessTokenCutoff(t *testing.T, ctx context.Context) {
	t.Helper()
	raw, err := h.redis.Get(ctx, "auth:access:not-before").Result()
	if err == redis.Nil {
		return
	}
	if err != nil {
		t.Fatalf("读取访问令牌失效时间失败: %v", err)
	}
	cutoff, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		t.Fatalf("访问令牌失效时间格式无效")
	}
	until := time.Until(time.Unix(cutoff+1, 0))
	if until <= 0 {
		return
	}
	timer := time.NewTimer(until)
	defer timer.Stop()
	select {
	case <-timer.C:
	case <-ctx.Done():
		t.Fatal("等待访问令牌失效时间结束超时")
	}
}

func (h *harness) dockerRedis(ctx context.Context, operation string) error {
	command := exec.CommandContext(ctx, "docker", operation, h.redisContainerID)
	output, err := command.CombinedOutput()
	if err == nil {
		return nil
	}
	if ctx.Err() != nil {
		return fmt.Errorf("docker %s Redis 超时: %w", operation, ctx.Err())
	}
	message := strings.TrimSpace(string(output))
	if operation == "start" && (strings.Contains(message, "already started") || strings.Contains(message, "already running")) {
		return nil
	}
	if message == "" {
		return fmt.Errorf("docker %s Redis 失败: %w", operation, err)
	}
	return fmt.Errorf("docker %s Redis 失败: %s", operation, message)
}

func (h *harness) startRedisAndWait() error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	if err := h.dockerRedis(ctx, "start"); err != nil {
		return err
	}
	return h.waitForRedis(ctx)
}

func (h *harness) waitForRedis(ctx context.Context) error {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	var lastErr error
	for {
		if err := h.pingRedisContainer(ctx); err != nil {
			lastErr = err
		} else if err := h.apiRedisHealthy(ctx); err != nil {
			lastErr = err
		} else {
			return nil
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("等待 Redis 恢复超时: %w", lastErr)
		case <-ticker.C:
		}
	}
}

func (h *harness) pingRedisContainer(ctx context.Context) error {
	pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	command := exec.CommandContext(pingCtx, "docker", "exec", h.redisContainerID, "redis-cli", "ping")
	output, err := command.CombinedOutput()
	if err != nil {
		return fmt.Errorf("容器内 Redis ping 失败: %w", err)
	}
	if strings.TrimSpace(string(output)) != "PONG" {
		return fmt.Errorf("容器内 Redis ping 返回 %q", strings.TrimSpace(string(output)))
	}
	return nil
}

func (h *harness) apiRedisHealthy(ctx context.Context) error {
	healthCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	result, err := h.request(healthCtx, http.MethodGet, h.apiURL("/healthz"), nil, nil, "", "")
	if err != nil {
		return fmt.Errorf("API 健康检查失败: %w", err)
	}
	if result.status != http.StatusOK {
		return fmt.Errorf("API 健康检查状态码 = %d", result.status)
	}
	var report healthReport
	if err := json.Unmarshal(result.body, &report); err != nil {
		return fmt.Errorf("解析 API 健康检查失败: %w", err)
	}
	if report.Status != "ok" || report.Checks["redis"].Status != "up" {
		return fmt.Errorf("API Redis 健康状态未恢复")
	}
	return nil
}

func (h *harness) apiURL(path string) string {
	return h.apiBaseURL + path
}

func (h *harness) webURL(path string) string {
	return h.webBaseURL + path
}

func (h *harness) request(ctx context.Context, method, target string, body io.Reader, headers http.Header, accessToken, refreshToken string) (apiResult, error) {
	req, err := http.NewRequestWithContext(ctx, method, target, body)
	if err != nil {
		return apiResult{}, err
	}
	for name, values := range headers {
		for _, value := range values {
			req.Header.Add(name, value)
		}
	}
	if accessToken != "" {
		req.Header.Set("Authorization", "Bearer "+accessToken)
	}
	if refreshToken != "" {
		req.AddCookie(&http.Cookie{Name: refreshCookie, Value: refreshToken})
	}
	resp, err := h.client.Do(req)
	if err != nil {
		return apiResult{}, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return apiResult{}, err
	}
	result := apiResult{status: resp.StatusCode, body: raw, header: resp.Header.Clone(), cookies: resp.Cookies()}
	if len(raw) > 0 && json.Unmarshal(raw, &result.envelope) == nil {
		return result, nil
	}
	return result, nil
}

func (h *harness) jsonRequest(ctx context.Context, method, path string, payload any, accessToken, refreshToken string) (apiResult, error) {
	var body io.Reader
	if payload != nil {
		raw, err := json.Marshal(payload)
		if err != nil {
			return apiResult{}, err
		}
		body = bytes.NewReader(raw)
	}
	headers := make(http.Header)
	headers.Set("Accept", "application/json")
	if payload != nil {
		headers.Set("Content-Type", "application/json")
	}
	return h.request(ctx, method, h.apiURL(path), body, headers, accessToken, refreshToken)
}

func (h *harness) newCredentials(prefix string) credentials {
	id := uniqueSequence.Add(1)
	account := fmt.Sprintf("e2e%s%d", prefix, id)
	return credentials{
		account:  account,
		email:    account + "@example.test",
		password: "E2E-password-2026!",
	}
}

func (h *harness) uniqueSlug(prefix string) string {
	return fmt.Sprintf("e2e-%s-%d", prefix, uniqueSequence.Add(1))
}

func (h *harness) seedRegistrationCode(t *testing.T, email string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	key := "verify_code:register:" + strings.ToLower(strings.TrimSpace(email))
	if err := h.redis.Set(ctx, key, verificationCode, 5*time.Minute).Err(); err != nil {
		t.Fatalf("写入注册验证码测试数据失败: %v", err)
	}
}

func registrationPayload(credentials credentials) map[string]string {
	return map[string]string{
		"account":   credentials.account,
		"password":  credentials.password,
		"email":     credentials.email,
		"nickname":  "E2E 集成测试",
		"emailCode": verificationCode,
	}
}

func (h *harness) registerRequest(ctx context.Context, credentials credentials) (session, apiResult, error) {
	result, err := h.jsonRequest(ctx, http.MethodPost, "/api/v1/auth/register", registrationPayload(credentials), "", "")
	if err != nil {
		return session{}, apiResult{}, err
	}
	if result.status != http.StatusOK || result.envelope.Code != 0 {
		return session{}, result, nil
	}
	var pair tokenPair
	if err := json.Unmarshal(result.envelope.Data, &pair); err != nil {
		return session{}, apiResult{}, err
	}
	return session{credentials: credentials, accessToken: pair.AccessToken, refreshToken: cookieValue(result.cookies, refreshCookie)}, result, nil
}

func (h *harness) registerUser(t *testing.T, prefix string) session {
	t.Helper()
	credentials := h.newCredentials(prefix)
	h.seedRegistrationCode(t, credentials.email)
	created, result, err := h.registerRequest(context.Background(), credentials)
	if err != nil {
		t.Fatalf("注册用户请求失败: %v", err)
	}
	expectAPISuccess(t, result)
	h.assertSession(t, created)
	var pair tokenPair
	if err := json.Unmarshal(result.envelope.Data, &pair); err != nil {
		t.Fatalf("解析注册令牌响应失败: %v", err)
	}
	if pair.RefreshToken != "" {
		t.Fatal("注册响应不应在 JSON 中暴露 refreshToken")
	}
	assertRefreshCookie(t, findCookie(result.cookies, refreshCookie))
	return created
}

func (h *harness) login(t *testing.T, credentials credentials) session {
	t.Helper()
	captchaID := fmt.Sprintf("e2e-login-%d", uniqueSequence.Add(1))
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := h.redis.Set(ctx, "captcha:login:"+captchaID, "4815", 5*time.Minute).Err(); err != nil {
		t.Fatalf("写入登录验证码测试数据失败: %v", err)
	}
	result, err := h.jsonRequest(context.Background(), http.MethodPost, "/api/v1/auth/login", map[string]string{
		"account":     credentials.account,
		"password":    credentials.password,
		"captchaId":   captchaID,
		"captchaCode": "4815",
	}, "", "")
	if err != nil {
		t.Fatalf("登录请求失败: %v", err)
	}
	expectAPISuccess(t, result)
	pair := decodeData[tokenPair](t, result)
	if pair.RefreshToken != "" {
		t.Fatal("登录响应不应在 JSON 中暴露 refreshToken")
	}
	created := session{credentials: credentials, accessToken: pair.AccessToken, refreshToken: cookieValue(result.cookies, refreshCookie)}
	h.assertSession(t, created)
	assertRefreshCookie(t, findCookie(result.cookies, refreshCookie))
	return created
}

func (h *harness) assertSession(t *testing.T, session session) {
	t.Helper()
	if session.accessToken == "" {
		t.Fatal("认证响应未返回 accessToken")
	}
	if session.refreshToken == "" {
		t.Fatal("认证响应未设置 HttpOnly refresh cookie")
	}
}

func (h *harness) uploadMedia(t *testing.T, accessToken string, data []byte) mediaAsset {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("altText", "集成测试真实像素图"); err != nil {
		t.Fatalf("写入媒体 altText 失败: %v", err)
	}
	part, err := writer.CreateFormFile("file", "e2e-pixel.png")
	if err != nil {
		t.Fatalf("创建媒体 multipart 文件字段失败: %v", err)
	}
	if _, err := part.Write(data); err != nil {
		t.Fatalf("写入媒体文件失败: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("关闭媒体 multipart 请求失败: %v", err)
	}
	headers := make(http.Header)
	headers.Set("Content-Type", writer.FormDataContentType())
	headers.Set("Accept", "application/json")
	result, err := h.request(context.Background(), http.MethodPost, h.apiURL("/api/v1/admin/media"), &body, headers, accessToken, "")
	if err != nil {
		t.Fatalf("上传真实媒体失败: %v", err)
	}
	expectAPISuccess(t, result)
	asset := decodeData[mediaAsset](t, result)
	if asset.ID == 0 || asset.StorageKey == "" || asset.URL == "" || asset.MIMEType != "image/png" {
		t.Fatalf("媒体上传响应不完整: %#v", asset)
	}
	return asset
}

func (h *harness) assertMedia(t *testing.T, asset mediaAsset, want []byte) {
	t.Helper()
	result, err := h.request(context.Background(), http.MethodGet, h.webURL(asset.URL), nil, nil, "", "")
	if err != nil {
		t.Fatalf("读取真实媒体失败: %v", err)
	}
	if result.status != http.StatusOK {
		t.Fatalf("真实媒体状态码 = %d，期望 %d", result.status, http.StatusOK)
	}
	if contentType := result.header.Get("Content-Type"); !strings.HasPrefix(contentType, "image/png") {
		t.Fatalf("真实媒体 Content-Type = %q，期望 image/png", contentType)
	}
	if !bytes.Equal(result.body, want) {
		t.Fatal("真实媒体内容与上传数据不一致")
	}
}

func (h *harness) createPublishedArticle(t *testing.T, accessToken, title, content, coverURL string) article {
	t.Helper()
	result, err := h.jsonRequest(context.Background(), http.MethodPost, "/api/v1/articles", map[string]any{
		"title":    title,
		"slug":     h.uniqueSlug("article"),
		"summary":  "E2E 集成测试文章摘要",
		"content":  content,
		"coverUrl": coverURL,
		"status":   "published",
	}, accessToken, "")
	if err != nil {
		t.Fatalf("创建文章失败: %v", err)
	}
	expectAPISuccess(t, result)
	created := decodeData[article](t, result)
	if created.ID == 0 || created.Status != "published" {
		t.Fatalf("创建文章响应不完整: %#v", created)
	}
	return created
}

func (h *harness) publicArticle(t *testing.T, id uint64) article {
	t.Helper()
	result, err := h.jsonRequest(context.Background(), http.MethodGet, fmt.Sprintf("/api/v1/articles/%d", id), nil, "", "")
	if err != nil {
		t.Fatalf("读取公开文章失败: %v", err)
	}
	expectAPISuccess(t, result)
	return decodeData[article](t, result)
}

func (h *harness) exportBackup(t *testing.T, accessToken, password string) []byte {
	t.Helper()
	result, err := h.jsonRequest(context.Background(), http.MethodPost, "/api/v1/admin/backups/export", map[string]string{
		"currentPassword": password,
		"passphrase":      backupPassphrase,
	}, accessToken, "")
	if err != nil {
		t.Fatalf("导出备份失败: %v", err)
	}
	if result.status != http.StatusOK {
		t.Fatalf("导出备份状态码 = %d（code=%d, message=%q）", result.status, result.envelope.Code, result.envelope.Message)
	}
	if !strings.HasPrefix(result.header.Get("Content-Type"), "application/octet-stream") {
		t.Fatalf("备份导出 Content-Type = %q，期望 application/octet-stream", result.header.Get("Content-Type"))
	}
	return result.body
}

func (h *harness) restoreBackup(t *testing.T, accessToken string, archive []byte, password string) apiResult {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for name, value := range map[string]string{
		"currentPassword": password,
		"passphrase":      backupPassphrase,
		"confirmation":    "REPLACE",
	} {
		if err := writer.WriteField(name, value); err != nil {
			t.Fatalf("写入备份恢复字段 %s 失败: %v", name, err)
		}
	}
	part, err := writer.CreateFormFile("file", "integration.noa-backup")
	if err != nil {
		t.Fatalf("创建备份恢复文件字段失败: %v", err)
	}
	if _, err := part.Write(archive); err != nil {
		t.Fatalf("写入备份恢复文件失败: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("关闭备份恢复 multipart 请求失败: %v", err)
	}
	headers := make(http.Header)
	headers.Set("Content-Type", writer.FormDataContentType())
	headers.Set("Accept", "application/json")
	result, err := h.request(context.Background(), http.MethodPost, h.apiURL("/api/v1/admin/backups/restore"), &body, headers, accessToken, "")
	if err != nil {
		t.Fatalf("恢复备份请求失败: %v", err)
	}
	return result
}

func onePixelPNG(t *testing.T) []byte {
	t.Helper()
	const encoded = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVQIHWP4z8DwHwAFgAI/ScL0JwAAAABJRU5ErkJggg=="
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatalf("解码真实 PNG 测试数据失败: %v", err)
	}
	return data
}

func expectAPISuccess(t *testing.T, result apiResult) {
	t.Helper()
	if result.status != http.StatusOK || result.envelope.Code != 0 {
		t.Fatalf("API 请求失败: status=%d code=%d message=%q", result.status, result.envelope.Code, result.envelope.Message)
	}
}

func expectAPIStatus(t *testing.T, result apiResult, want int) {
	t.Helper()
	if result.status != want {
		t.Fatalf("API 状态码 = %d，期望 %d（code=%d, message=%q）", result.status, want, result.envelope.Code, result.envelope.Message)
	}
}

func decodeData[T any](t *testing.T, result apiResult) T {
	t.Helper()
	var value T
	if len(result.envelope.Data) == 0 {
		t.Fatal("API 成功响应缺少 data")
	}
	if err := json.Unmarshal(result.envelope.Data, &value); err != nil {
		t.Fatalf("解析 API data 失败: %v", err)
	}
	return value
}

func cookieValue(cookies []*http.Cookie, name string) string {
	cookie := findCookie(cookies, name)
	if cookie == nil {
		return ""
	}
	return cookie.Value
}

func findCookie(cookies []*http.Cookie, name string) *http.Cookie {
	for _, cookie := range cookies {
		if cookie.Name == name {
			return cookie
		}
	}
	return nil
}

func assertRefreshCookie(t *testing.T, cookie *http.Cookie) {
	t.Helper()
	if cookie == nil {
		t.Fatal("认证响应未设置 refresh cookie")
	}
	if !cookie.HttpOnly || cookie.Path != "/" || cookie.SameSite != http.SameSiteStrictMode || cookie.Secure || cookie.MaxAge <= 0 {
		t.Fatalf("refresh cookie 属性不符合本地 HTTP 集成环境要求: HttpOnly=%t Path=%q SameSite=%d Secure=%t MaxAge=%d", cookie.HttpOnly, cookie.Path, cookie.SameSite, cookie.Secure, cookie.MaxAge)
	}
}
