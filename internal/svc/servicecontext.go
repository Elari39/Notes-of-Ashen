package svc

import (
	"notes-of-ashen/internal/authutil"
	"notes-of-ashen/internal/config"
	"notes-of-ashen/internal/mq"
	"notes-of-ashen/model"

	"github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/core/logx"
)

type ServiceContext struct {
	Config config.Config
	Store  *model.Store
	Redis  *redis.Client
	Tokens *authutil.Manager
	Events *mq.Publisher
}

func NewServiceContext(c config.Config) *ServiceContext {
	db, err := model.Open(c.Database.DataSource, c.Database.MaxOpenConns, c.Database.MaxIdleConns)
	logx.Must(err)

	store := model.NewStore(db)
	redisClient := redis.NewClient(&redis.Options{
		Addr:     c.Redis.Addr,
		Password: c.Redis.Password,
		DB:       c.Redis.DB,
	})
	events := mq.NewPublisher(c.RabbitMQ)
	mq.StartConsumer(c.RabbitMQ, db)

	return &ServiceContext{
		Config: c,
		Store:  store,
		Redis:  redisClient,
		Tokens: authutil.NewManager(c.Auth.AccessSecret, c.Auth.AccessExpire, c.Auth.RefreshExpire),
		Events: events,
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
