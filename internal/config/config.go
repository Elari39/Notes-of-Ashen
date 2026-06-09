package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/zeromicro/go-zero/rest"
)

type Config struct {
	rest.RestConf
	Database DatabaseConf
	Auth     AuthConf
	Redis    RedisConf
	RabbitMQ RabbitMQConf
	Email    EmailConf
	AI       AIConf
	GeoIP    GeoIPConf
}

type DatabaseConf struct {
	DataSource   string
	MaxOpenConns int
	MaxIdleConns int
}

type AuthConf struct {
	AccessSecret  string
	AccessExpire  int64
	RefreshExpire int64
}

type RedisConf struct {
	Addr     string
	Password string
	DB       int
}

type RabbitMQConf struct {
	Enabled    bool
	URL        string
	Exchange   string
	Queue      string
	RoutingKey string
}

type EmailConf struct {
	Enabled      bool
	SMTPHost     string
	SMTPPort     int
	SMTPUsername string
	SMTPPassword string
	From         string
	FromName     string
}

type AIConf struct {
	Enabled        bool
	BaseURL        string
	APIKey         string
	Model          string
	TimeoutSeconds int
}

type GeoIPConf struct {
	DatabasePath string
}

func (c *Config) ApplyEnv() error {
	setString := func(key string, target *string) {
		if value, ok := os.LookupEnv(key); ok {
			*target = value
		}
	}

	setInt := func(key string, target *int) error {
		value, ok := os.LookupEnv(key)
		if !ok {
			return nil
		}
		parsed, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("invalid integer env %s: %w", key, err)
		}
		*target = parsed
		return nil
	}

	setInt64 := func(key string, target *int64) error {
		value, ok := os.LookupEnv(key)
		if !ok {
			return nil
		}
		parsed, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid int64 env %s: %w", key, err)
		}
		*target = parsed
		return nil
	}

	setBool := func(key string, target *bool) error {
		value, ok := os.LookupEnv(key)
		if !ok {
			return nil
		}
		parsed, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("invalid boolean env %s: %w", key, err)
		}
		*target = parsed
		return nil
	}

	setString("APP_HOST", &c.Host)
	if err := setInt("APP_PORT", &c.Port); err != nil {
		return err
	}
	setString("APP_DATABASE_DSN", &c.Database.DataSource)
	if err := setInt("APP_DATABASE_MAX_OPEN_CONNS", &c.Database.MaxOpenConns); err != nil {
		return err
	}
	if err := setInt("APP_DATABASE_MAX_IDLE_CONNS", &c.Database.MaxIdleConns); err != nil {
		return err
	}
	setString("APP_AUTH_ACCESS_SECRET", &c.Auth.AccessSecret)
	if err := setInt64("APP_AUTH_ACCESS_EXPIRE", &c.Auth.AccessExpire); err != nil {
		return err
	}
	if err := setInt64("APP_AUTH_REFRESH_EXPIRE", &c.Auth.RefreshExpire); err != nil {
		return err
	}
	setString("APP_REDIS_ADDR", &c.Redis.Addr)
	setString("APP_REDIS_PASSWORD", &c.Redis.Password)
	if err := setInt("APP_REDIS_DB", &c.Redis.DB); err != nil {
		return err
	}
	if err := setBool("APP_RABBITMQ_ENABLED", &c.RabbitMQ.Enabled); err != nil {
		return err
	}
	setString("APP_RABBITMQ_URL", &c.RabbitMQ.URL)
	setString("APP_RABBITMQ_EXCHANGE", &c.RabbitMQ.Exchange)
	setString("APP_RABBITMQ_QUEUE", &c.RabbitMQ.Queue)
	setString("APP_RABBITMQ_ROUTING_KEY", &c.RabbitMQ.RoutingKey)
	if err := setBool("APP_EMAIL_ENABLED", &c.Email.Enabled); err != nil {
		return err
	}
	setString("APP_EMAIL_SMTP_HOST", &c.Email.SMTPHost)
	if err := setInt("APP_EMAIL_SMTP_PORT", &c.Email.SMTPPort); err != nil {
		return err
	}
	setString("APP_EMAIL_SMTP_USERNAME", &c.Email.SMTPUsername)
	setString("APP_EMAIL_SMTP_PASSWORD", &c.Email.SMTPPassword)
	setString("APP_EMAIL_FROM", &c.Email.From)
	setString("APP_EMAIL_FROM_NAME", &c.Email.FromName)
	if err := setBool("APP_AI_ENABLED", &c.AI.Enabled); err != nil {
		return err
	}
	setString("APP_AI_BASE_URL", &c.AI.BaseURL)
	setString("APP_AI_API_KEY", &c.AI.APIKey)
	setString("APP_AI_MODEL", &c.AI.Model)
	if err := setInt("APP_AI_TIMEOUT_SECONDS", &c.AI.TimeoutSeconds); err != nil {
		return err
	}
	setString("APP_GEOIP_DATABASE_PATH", &c.GeoIP.DatabasePath)

	return nil
}
