package svc

import (
	"time"

	"notes-of-ashen/internal/authutil"
	appcache "notes-of-ashen/internal/cache"
	"notes-of-ashen/internal/config"
	"notes-of-ashen/internal/emailer"
	"notes-of-ashen/internal/mq"
	"notes-of-ashen/internal/search"
	"notes-of-ashen/model"

	"github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/core/logx"
)

type ServiceContext struct {
	Config config.Config
	Store  *model.Store
	Redis  *redis.Client
	Cache  *appcache.JSONCache
	Search *search.Client
	Tokens *authutil.Manager
	Events *mq.Publisher
	Mailer *emailer.Sender
}

func NewServiceContext(c config.Config) *ServiceContext {
	db, err := model.Open(c.Database.DataSource, c.Database.MaxOpenConns, c.Database.MaxIdleConns)
	logx.Must(err)

	store := model.NewStore(db)
	redisClient := redis.NewClient(&redis.Options{
		Addr:            c.Redis.Addr,
		Password:        c.Redis.Password,
		DB:              c.Redis.DB,
		DialTimeout:     300 * time.Millisecond,
		ReadTimeout:     300 * time.Millisecond,
		WriteTimeout:    300 * time.Millisecond,
		PoolTimeout:     300 * time.Millisecond,
		MaxRetries:      -1,
		MinRetryBackoff: 50 * time.Millisecond,
		MaxRetryBackoff: 100 * time.Millisecond,
	})
	events := mq.NewPublisher(c.RabbitMQ)
	mq.StartConsumer(c.RabbitMQ, db)

	return &ServiceContext{
		Config: c,
		Store:  store,
		Redis:  redisClient,
		Cache:  appcache.NewJSONCache(redisClient),
		Search: search.NewClient(c.Search),
		Tokens: authutil.NewManager(c.Auth.AccessSecret, c.Auth.AccessExpire, c.Auth.RefreshExpire),
		Events: events,
		Mailer: emailer.NewSender(c.Email),
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
