package svc

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"notes-of-ashen/internal/authutil"
	appcache "notes-of-ashen/internal/cache"
	"notes-of-ashen/internal/config"
	"notes-of-ashen/internal/emailer"
	"notes-of-ashen/internal/middleware"
	"notes-of-ashen/internal/mq"
	"notes-of-ashen/internal/rag"
	"notes-of-ashen/internal/search"
	"notes-of-ashen/model"

	"github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/core/logx"
)

// startupRedisTimeout 控制启动期 Redis 拨号与 PING 的超时。
// 远程/VPN 场景下 3s 偏紧，放宽到 10s；仅影响启动探测与重连拨号，
// 不影响稳态的 Read/Write/Pool 超时。
const startupRedisTimeout = 10 * time.Second

const searchIndexRetryInterval = 30 * time.Second

const (
	refreshTokenCleanupInterval  = 24 * time.Hour
	refreshTokenRevokedRetention = 30 * 24 * time.Hour
	refreshTokenCleanupTimeout   = 30 * time.Second
	ragChatCleanupInterval       = 24 * time.Hour
	ragChatCleanupTimeout        = 30 * time.Second
)

func isRedisAuthenticationError(err error) bool {
	if err == nil {
		return false
	}

	message := strings.ToLower(err.Error())
	return strings.Contains(message, "wrongpass") ||
		strings.Contains(message, "noauth") ||
		strings.Contains(message, "authentication required") ||
		strings.Contains(message, "invalid username-password pair") ||
		strings.Contains(message, "auth <password> called without any password configured")
}

// redisStartupPingFailure 生成供 logx.Must 使用的安全启动错误。认证失败时不透传
// Redis 服务端原始错误，避免将意外包含的凭据写入启动日志。
func redisStartupPingFailure(err error) error {
	if isRedisAuthenticationError(err) {
		return errors.New("redis PING authentication failed; verify APP_REDIS_PASSWORD matches the target Redis credentials")
	}
	return fmt.Errorf("redis PING failed: %w", err)
}

type ServiceContext struct {
	Config               config.Config
	Store                *model.Store
	Redis                *redis.Client
	Cache                *appcache.JSONCache
	Search               *search.Client
	Tokens               *authutil.Manager
	Events               *mq.Publisher
	Mailer               *emailer.Sender
	RAGWorker            *rag.Worker
	AuthUserCache        middleware.AuthUserCache
	searchCancel         context.CancelFunc
	tokenCleanupCancel   context.CancelFunc
	ragChatCleanupCancel context.CancelFunc
}

func NewServiceContext(c config.Config) *ServiceContext {
	db, err := model.Open(c.Database.DataSource, c.Database.MaxOpenConns, c.Database.MaxIdleConns)
	logx.Must(err)

	store := model.NewStore(db)
	redisClient := redis.NewClient(&redis.Options{
		Addr:         c.Redis.Addr,
		Password:     c.Redis.Password,
		DB:           c.Redis.DB,
		DialTimeout:  startupRedisTimeout,
		ReadTimeout:  1 * time.Second,
		WriteTimeout: 1 * time.Second,
		// 让调用方传入的 context deadline 生效。敏感接口的限流中间件依赖
		// 短超时在 Redis 不可用时 fail-closed，而不是被客户端重试拖延。
		ContextTimeoutEnabled: true,
		PoolTimeout:           2 * time.Second,
		PoolSize:              20,
		MinIdleConns:          3,
		MaxRetries:            2,
		MinRetryBackoff:       50 * time.Millisecond,
		MaxRetryBackoff:       300 * time.Millisecond,
	})
	// Redis 是限流、缓存、refresh token 的关键依赖，启动时 PING 校验，失败 fail-fast。
	// 打印实际使用的 Addr（host:port，不含密码），便于确认 APP_REDIS_ADDR 是否注入正确。
	logx.Infof("[startup] pinging redis at %s (db=%d)", c.Redis.Addr, c.Redis.DB)
	pingCtx, cancel := context.WithTimeout(context.Background(), startupRedisTimeout)
	if err := redisClient.Ping(pingCtx).Err(); err != nil {
		cancel()
		if isRedisAuthenticationError(err) {
			logx.Error("[startup] redis PING authentication failed; API will exit. Verify APP_REDIS_PASSWORD matches the target Redis credentials.")
		} else {
			logx.Errorf("[startup] redis PING failed; API will exit: %v", err)
		}
		logx.Must(redisStartupPingFailure(err))
	}
	cancel()
	events := mq.NewPublisher(c.RabbitMQ, db)
	mq.StartConsumer(c.RabbitMQ, db)

	searchClient := search.NewClient(c.Search)
	searchCancel := initializeSearch(searchClient)
	tokenCleanupCancel := startRefreshTokenCleanup(store)
	// 私有会话的留存策略独立于 RAG 引擎开关。恢复后或临时停用 RAG 时，
	// 仍需按期删除过期数据，不能依赖 Worker 恰好运行。
	ragChatCleanupCancel := startRAGChatCleanup(store)
	var ragWorker *rag.Worker
	if c.RAG.Enabled {
		worker, workerErr := rag.NewWorker(c.RAG, c.Auth.AccessSecret, store)
		if workerErr != nil {
			logx.Errorf("[startup] RAG worker initialization failed; chat stays unavailable: %v", workerErr)
		} else {
			ragWorker = worker
			ragWorker.Start()
		}
	}

	return &ServiceContext{
		Config:               c,
		Store:                store,
		Redis:                redisClient,
		Cache:                appcache.NewJSONCache(redisClient),
		Search:               searchClient,
		Tokens:               authutil.NewManager(c.Auth.AccessSecret, c.Auth.AccessExpire, c.Auth.RefreshExpire),
		Events:               events,
		Mailer:               emailer.NewSender(c.Email),
		RAGWorker:            ragWorker,
		AuthUserCache:        middleware.NewAuthUserCache(redisClient),
		searchCancel:         searchCancel,
		tokenCleanupCancel:   tokenCleanupCancel,
		ragChatCleanupCancel: ragChatCleanupCancel,
	}
}

func startRefreshTokenCleanup(store *model.Store) context.CancelFunc {
	if store == nil {
		return nil
	}

	cleanup := func(parent context.Context) {
		ctx, cancel := context.WithTimeout(parent, refreshTokenCleanupTimeout)
		defer cancel()
		now := time.Now().UTC()
		deleted, err := store.CleanupRefreshTokens(ctx, now, now.Add(-refreshTokenRevokedRetention))
		if err != nil {
			logx.Errorf("[auth] refresh token cleanup failed: %v", err)
			return
		}
		if deleted > 0 {
			logx.Infof("[auth] refresh token cleanup removed %d historical records", deleted)
		}
	}

	// 启动时先清理一次，之后按固定周期运行；清理失败不阻断 API 服务。
	cleanup(context.Background())
	cleanupCtx, cancel := context.WithCancel(context.Background())
	go func() {
		ticker := time.NewTicker(refreshTokenCleanupInterval)
		defer ticker.Stop()
		for {
			select {
			case <-cleanupCtx.Done():
				return
			case <-ticker.C:
				cleanup(cleanupCtx)
			}
		}
	}()
	return cancel
}

func startRAGChatCleanup(store *model.Store) context.CancelFunc {
	if store == nil {
		return nil
	}

	cleanup := func(parent context.Context) {
		ctx, cancel := context.WithTimeout(parent, ragChatCleanupTimeout)
		defer cancel()
		deleted, err := store.CleanupExpiredRAGChatSessions(ctx, time.Now().UTC())
		if err != nil {
			logx.Errorf("[rag] chat session cleanup failed: %v", err)
			return
		}
		if deleted > 0 {
			logx.Infof("[rag] chat session cleanup removed %d expired records", deleted)
		}
	}

	cleanup(context.Background())
	cleanupCtx, cancel := context.WithCancel(context.Background())
	go func() {
		ticker := time.NewTicker(ragChatCleanupInterval)
		defer ticker.Stop()
		for {
			select {
			case <-cleanupCtx.Done():
				return
			case <-ticker.C:
				cleanup(cleanupCtx)
			}
		}
	}()
	return cancel
}

// initializeSearch 尝试在启动阶段创建并配置 Meilisearch 索引。Meilisearch 是可选依赖，
// 初始化失败时保留客户端，让现有搜索调用按错误回退 MySQL，并在后台持续重试直至恢复。
func initializeSearch(client *search.Client) context.CancelFunc {
	if !client.Enabled() {
		return nil
	}

	ensure := func(ctx context.Context) error {
		ensureCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
		return client.EnsureIndex(ensureCtx)
	}
	return initializeSearchWithEnsure(ensure, searchIndexRetryInterval)
}

// initializeSearchWithEnsure 将重试调度与具体搜索客户端解耦，便于覆盖恢复和取消路径。
func initializeSearchWithEnsure(ensure func(context.Context) error, retryInterval time.Duration) context.CancelFunc {
	if err := ensure(context.Background()); err == nil {
		return nil
	} else {
		logx.Errorf("[startup] meilisearch index initialization failed; API will continue with MySQL search fallback and retry in background: %v", err)
	}

	retryCtx, cancel := context.WithCancel(context.Background())
	go func() {
		ticker := time.NewTicker(retryInterval)
		defer ticker.Stop()
		for {
			select {
			case <-retryCtx.Done():
				return
			case <-ticker.C:
				if err := ensure(retryCtx); err != nil {
					logx.Errorf("[search] meilisearch index initialization retry failed; continuing with MySQL fallback: %v", err)
					continue
				}
				logx.Info("[search] meilisearch index initialized successfully; full-text search is available")
				return
			}
		}
	}()
	return cancel
}

func (s *ServiceContext) Close() {
	if s.RAGWorker != nil {
		s.RAGWorker.Close()
	}
	if s.searchCancel != nil {
		s.searchCancel()
	}
	if s.tokenCleanupCancel != nil {
		s.tokenCleanupCancel()
	}
	if s.ragChatCleanupCancel != nil {
		s.ragChatCleanupCancel()
	}
	if s.Events != nil {
		s.Events.Close()
	}
	if s.Redis != nil {
		_ = s.Redis.Close()
	}
	if s.Store != nil && s.Store.DB() != nil {
		_ = s.Store.DB().Close()
	}
}
