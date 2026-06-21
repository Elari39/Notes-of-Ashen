package svc

import (
	"context"
	"time"

	"notes-of-ashen/internal/authutil"
	appcache "notes-of-ashen/internal/cache"
	"notes-of-ashen/internal/config"
	"notes-of-ashen/internal/emailer"
	"notes-of-ashen/internal/middleware"
	"notes-of-ashen/internal/mq"
	"notes-of-ashen/internal/search"
	"notes-of-ashen/model"

	"github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/core/logx"
)

// startupRedisTimeout 控制启动期 Redis 拨号与 PING 的超时。
// 远程/VPN 场景下 3s 偏紧，放宽到 10s；仅影响启动探测与重连拨号，
// 不影响稳态的 Read/Write/Pool 超时。
const startupRedisTimeout = 10 * time.Second

type ServiceContext struct {
	Config        config.Config
	Store         *model.Store
	Redis         *redis.Client
	Cache         *appcache.JSONCache
	Search        *search.Client
	Tokens        *authutil.Manager
	Events        *mq.Publisher
	Mailer        *emailer.Sender
	AuthUserCache middleware.AuthUserCache
}

func NewServiceContext(c config.Config) *ServiceContext {
	db, err := model.Open(c.Database.DataSource, c.Database.MaxOpenConns, c.Database.MaxIdleConns)
	logx.Must(err)

	store := model.NewStore(db)
	redisClient := redis.NewClient(&redis.Options{
		Addr:            c.Redis.Addr,
		Password:        c.Redis.Password,
		DB:              c.Redis.DB,
		DialTimeout:     startupRedisTimeout,
		ReadTimeout:     1 * time.Second,
		WriteTimeout:    1 * time.Second,
		PoolTimeout:     2 * time.Second,
		MaxRetries:      2,
		MinRetryBackoff: 50 * time.Millisecond,
		MaxRetryBackoff: 300 * time.Millisecond,
	})
	// Redis 是限流、缓存、refresh token 的关键依赖，启动时 PING 校验，失败 fail-fast。
	// 打印实际使用的 Addr（host:port，不含密码），便于确认 APP_REDIS_ADDR 是否注入正确。
	logx.Infof("[startup] pinging redis at %s (db=%d)", c.Redis.Addr, c.Redis.DB)
	pingCtx, cancel := context.WithTimeout(context.Background(), startupRedisTimeout)
	if err := redisClient.Ping(pingCtx).Err(); err != nil {
		cancel()
		logx.Must(err)
	}
	cancel()
	events := mq.NewPublisher(c.RabbitMQ, db)
	mq.StartConsumer(c.RabbitMQ, db)

	return &ServiceContext{
		Config:        c,
		Store:         store,
		Redis:         redisClient,
		Cache:         appcache.NewJSONCache(redisClient),
		Search:        search.NewClient(c.Search),
		Tokens:        authutil.NewManager(c.Auth.AccessSecret, c.Auth.AccessExpire, c.Auth.RefreshExpire),
		Events:        events,
		Mailer:        emailer.NewSender(c.Email),
		AuthUserCache: middleware.NewAuthUserCache(redisClient),
	}
}

func (s *ServiceContext) Close() {
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
