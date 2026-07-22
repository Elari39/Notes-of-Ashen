package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/zeromicro/go-zero/rest"
)

type Config struct {
	rest.RestConf
	Database DatabaseConf
	Auth     AuthConf
	Redis    RedisConf
	Search   SearchConf
	RabbitMQ RabbitMQConf
	Email    EmailConf
	Media    MediaConf
	Backup   BackupConf
	Proxy    ProxyConf
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
	CookieSecure  bool
}

type RedisConf struct {
	Addr     string
	Password string
	DB       int
}

type SearchConf struct {
	Enabled           bool
	MeilisearchHost   string
	MeilisearchAPIKey string
	MeilisearchIndex  string
}

type RabbitMQConf struct {
	Enabled    bool
	User       string
	Password   string
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
	TLSMode      string // implicit(默认,465) | starttls(587) | none(明文,仅内网测试)
}

type MediaConf struct {
	RootDir        string
	MaxUploadBytes int64
}

type BackupConf struct {
	MaxUploadBytes int64
}

const (
	DefaultMediaRootDir         = "./data/media"
	DefaultMediaMaxUploadBytes  = int64(10 << 20)
	DefaultBackupMaxUploadBytes = int64(1 << 30)
)

func (c MediaConf) EffectiveRootDir() string {
	if strings.TrimSpace(c.RootDir) == "" {
		return DefaultMediaRootDir
	}
	return strings.TrimSpace(c.RootDir)
}

func (c MediaConf) EffectiveMaxUploadBytes() int64 {
	if c.MaxUploadBytes == 0 {
		return DefaultMediaMaxUploadBytes
	}
	return c.MaxUploadBytes
}

func (c BackupConf) EffectiveMaxUploadBytes() int64 {
	if c.MaxUploadBytes == 0 {
		return DefaultBackupMaxUploadBytes
	}
	return c.MaxUploadBytes
}

type ProxyConf struct {
	TrustedCIDRs string
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
		// compose 用 ${VAR:-} 在未设置时注入空串，空串视为未设置跳过，
		// 避免 strconv 解析空串报错导致 ApplyEnv panic（P4-1 回归）。
		if strings.TrimSpace(value) == "" {
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
		if strings.TrimSpace(value) == "" {
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
		if strings.TrimSpace(value) == "" {
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
	if err := setInt64("APP_TIMEOUT", &c.Timeout); err != nil {
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
	if err := setBool("APP_AUTH_COOKIE_SECURE", &c.Auth.CookieSecure); err != nil {
		return err
	}
	setString("APP_REDIS_ADDR", &c.Redis.Addr)
	setString("APP_REDIS_PASSWORD", &c.Redis.Password)
	if err := setInt("APP_REDIS_DB", &c.Redis.DB); err != nil {
		return err
	}
	if err := setBool("APP_SEARCH_ENABLED", &c.Search.Enabled); err != nil {
		return err
	}
	setString("APP_MEILISEARCH_HOST", &c.Search.MeilisearchHost)
	setString("APP_MEILISEARCH_API_KEY", &c.Search.MeilisearchAPIKey)
	setString("APP_MEILISEARCH_INDEX", &c.Search.MeilisearchIndex)
	if err := setBool("APP_RABBITMQ_ENABLED", &c.RabbitMQ.Enabled); err != nil {
		return err
	}
	setString("APP_RABBITMQ_USER", &c.RabbitMQ.User)
	setString("APP_RABBITMQ_PASSWORD", &c.RabbitMQ.Password)
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
	setString("APP_EMAIL_TLS_MODE", &c.Email.TLSMode)
	setString("APP_MEDIA_ROOT", &c.Media.RootDir)
	if err := setInt64("APP_MEDIA_MAX_UPLOAD_BYTES", &c.Media.MaxUploadBytes); err != nil {
		return err
	}
	if err := setInt64("APP_BACKUP_MAX_UPLOAD_BYTES", &c.Backup.MaxUploadBytes); err != nil {
		return err
	}
	setString("APP_TRUSTED_PROXY_CIDRS", &c.Proxy.TrustedCIDRs)

	return nil
}

// ValidateOptionalDependencyProfiles validates the Compose-only contract between
// optional dependency switches and the profiles that create their containers.
// COMPOSE_PROFILES is intentionally optional so non-Compose deployments can use
// externally managed RabbitMQ/Meilisearch instances without inheriting Compose
// assumptions.
func (c Config) ValidateOptionalDependencyProfiles() error {
	profiles, ok := os.LookupEnv("COMPOSE_PROFILES")
	if !ok {
		return nil
	}
	active := composeProfiles(profiles)

	if c.Search.Enabled != active["search"] {
		return fmt.Errorf("search configuration mismatch: APP_SEARCH_ENABLED=%t requires COMPOSE_PROFILES to %sinclude search", c.Search.Enabled, ternary(c.Search.Enabled, "", "not "))
	}
	if c.Search.Enabled {
		if err := validateComposeURL("APP_MEILISEARCH_HOST", c.Search.MeilisearchHost, "http", "https"); err != nil {
			return err
		}
		if err := validateRequiredSecret("APP_MEILISEARCH_API_KEY", c.Search.MeilisearchAPIKey, 8); err != nil {
			return err
		}
	} else if strings.TrimSpace(c.Search.MeilisearchAPIKey) != "" {
		return errors.New("APP_MEILISEARCH_API_KEY must be empty when APP_SEARCH_ENABLED=false")
	}

	if c.RabbitMQ.Enabled != active["messaging"] {
		return fmt.Errorf("RabbitMQ configuration mismatch: APP_RABBITMQ_ENABLED=%t requires COMPOSE_PROFILES to %sinclude messaging", c.RabbitMQ.Enabled, ternary(c.RabbitMQ.Enabled, "", "not "))
	}
	if c.RabbitMQ.Enabled {
		if strings.TrimSpace(c.RabbitMQ.User) == "" {
			return errors.New("APP_RABBITMQ_USER is required when messaging profile is enabled")
		}
		if err := validateRequiredSecret("APP_RABBITMQ_PASSWORD", c.RabbitMQ.Password, 12); err != nil {
			return err
		}
		parsed, err := url.Parse(strings.TrimSpace(c.RabbitMQ.URL))
		if err != nil || (parsed.Scheme != "amqp" && parsed.Scheme != "amqps") || parsed.Host == "" || parsed.User == nil {
			return errors.New("APP_RABBITMQ_URL must be a valid amqp(s) URL with user and password")
		}
		password, hasPassword := parsed.User.Password()
		if !hasPassword || password == "" || parsed.User.Username() != strings.TrimSpace(c.RabbitMQ.User) || password != c.RabbitMQ.Password {
			return errors.New("APP_RABBITMQ_URL credentials must match APP_RABBITMQ_USER and APP_RABBITMQ_PASSWORD")
		}
	} else if strings.TrimSpace(c.RabbitMQ.URL) != "" || strings.TrimSpace(c.RabbitMQ.Password) != "" {
		return errors.New("APP_RABBITMQ_URL and APP_RABBITMQ_PASSWORD must be empty when APP_RABBITMQ_ENABLED=false")
	}

	return nil
}

func composeProfiles(value string) map[string]bool {
	profiles := make(map[string]bool)
	for _, item := range strings.FieldsFunc(value, func(r rune) bool { return r == ',' || r == ' ' || r == '\t' || r == '\r' || r == '\n' }) {
		profiles[strings.TrimSpace(item)] = true
	}
	return profiles
}

func validateComposeURL(name, value string, schemes ...string) error {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil || parsed.Host == "" {
		return fmt.Errorf("%s must be a valid URL", name)
	}
	for _, scheme := range schemes {
		if parsed.Scheme == scheme {
			return nil
		}
	}
	return fmt.Errorf("%s must use one of %s", name, strings.Join(schemes, "|"))
}

func ternary(condition bool, whenTrue, whenFalse string) string {
	if condition {
		return whenTrue
	}
	return whenFalse
}

const minAccessSecretLength = 16

var insecureConfigMarkers = []string{
	"please-change",
	"replace-with",
	"change-me",
	"<replace",
	"notes_of_ashen_meili_master_key",
}

// ValidateConfig validates the resolved configuration. It is intentionally NOT
// named Validate so go-zero's conf.MustLoad does not auto-invoke it before
// .env / ApplyEnv have run (MustLoad calls any Validate() error method
// immediately after unmarshalling the YAML, which would fail on the empty
// placeholders that are meant to be filled from the environment).
func (c Config) ValidateConfig() error {
	if strings.TrimSpace(c.Database.DataSource) == "" {
		return fmt.Errorf("APP_DATABASE_DSN is required")
	}
	if err := validateRequiredSecret("APP_AUTH_ACCESS_SECRET", c.Auth.AccessSecret, minAccessSecretLength); err != nil {
		return err
	}
	if c.Auth.AccessExpire <= 0 {
		return fmt.Errorf("APP_AUTH_ACCESS_EXPIRE must be a positive integer")
	}
	if c.Auth.RefreshExpire <= 0 {
		return fmt.Errorf("APP_AUTH_REFRESH_EXPIRE must be a positive integer")
	}
	if c.Search.MeilisearchAPIKey != "" && containsInsecureMarker(c.Search.MeilisearchAPIKey) {
		return fmt.Errorf("APP_MEILISEARCH_API_KEY contains an insecure placeholder value")
	}
	if c.Search.Enabled {
		if err := validateRequiredSecret("APP_MEILISEARCH_API_KEY", c.Search.MeilisearchAPIKey, 8); err != nil {
			return err
		}
	}
	if c.RabbitMQ.Enabled {
		if strings.TrimSpace(c.RabbitMQ.URL) == "" {
			return fmt.Errorf("APP_RABBITMQ_URL is required when RabbitMQ is enabled")
		}
		if containsInsecureMarker(c.RabbitMQ.URL) {
			return fmt.Errorf("APP_RABBITMQ_URL contains an insecure placeholder value")
		}
	}
	if containsInsecureMarker(c.Database.DataSource) {
		return fmt.Errorf("APP_DATABASE_DSN contains an insecure placeholder value")
	}
	if c.Redis.Password != "" && containsInsecureMarker(c.Redis.Password) {
		return fmt.Errorf("APP_REDIS_PASSWORD contains an insecure placeholder value")
	}
	if strings.TrimSpace(c.Redis.Addr) == "" {
		return fmt.Errorf("APP_REDIS_ADDR is required")
	}
	if containsInsecureMarker(c.Redis.Addr) {
		return fmt.Errorf("APP_REDIS_ADDR contains an insecure placeholder value")
	}
	if c.Email.Enabled {
		switch strings.ToLower(strings.TrimSpace(c.Email.TLSMode)) {
		case "", "implicit", "starttls", "none":
		default:
			return fmt.Errorf("APP_EMAIL_TLS_MODE must be one of implicit|starttls|none, got %q", c.Email.TLSMode)
		}
		if err := validateRequiredSecret("APP_EMAIL_SMTP_HOST", c.Email.SMTPHost, 1); err != nil {
			return err
		}
		if c.Email.SMTPPort <= 0 {
			return fmt.Errorf("APP_EMAIL_SMTP_PORT must be a positive integer when Email is enabled")
		}
		if err := validateRequiredSecret("APP_EMAIL_SMTP_USERNAME", c.Email.SMTPUsername, 1); err != nil {
			return err
		}
		if err := validateRequiredSecret("APP_EMAIL_SMTP_PASSWORD", c.Email.SMTPPassword, 1); err != nil {
			return err
		}
		if err := validateRequiredSecret("APP_EMAIL_FROM", c.Email.From, 1); err != nil {
			return err
		}
	}
	if c.Media.MaxUploadBytes < 0 {
		return fmt.Errorf("APP_MEDIA_MAX_UPLOAD_BYTES must be a positive integer")
	}
	if c.Backup.MaxUploadBytes < 0 {
		return fmt.Errorf("APP_BACKUP_MAX_UPLOAD_BYTES must be a positive integer")
	}
	return nil
}

func validateRequiredSecret(name, value string, minLen int) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return fmt.Errorf("%s is required", name)
	}
	if len(value) < minLen {
		return fmt.Errorf("%s must be at least %d characters", name, minLen)
	}
	if containsInsecureMarker(value) {
		return fmt.Errorf("%s contains an insecure placeholder value", name)
	}
	return nil
}

func containsInsecureMarker(value string) bool {
	lower := strings.ToLower(value)
	for _, marker := range insecureConfigMarkers {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}
