package config

import (
	"strings"
	"testing"
)

func TestApplyEnvOverridesConfig(t *testing.T) {
	c := Config{}
	c.Database.DataSource = "root:123456@tcp(127.0.0.1:3306)/notes_of_ashen"
	c.Redis.Addr = "127.0.0.1:6379"
	c.RabbitMQ.Enabled = true

	t.Setenv("APP_PORT", "19001")
	t.Setenv("APP_DATABASE_DSN", "notes:secret@tcp(mysql:3306)/notes_of_ashen")
	t.Setenv("APP_REDIS_ADDR", "redis:6379")
	t.Setenv("APP_RABBITMQ_ENABLED", "false")
	t.Setenv("APP_AUTH_ACCESS_SECRET", "production-secret")
	t.Setenv("APP_REQUIRE_PUBLIC_SITE_URL", "true")
	t.Setenv("APP_EMAIL_ENABLED", "true")
	t.Setenv("APP_EMAIL_SMTP_HOST", "smtp.qq.com")
	t.Setenv("APP_EMAIL_SMTP_PORT", "465")
	t.Setenv("APP_EMAIL_SMTP_USERNAME", "user@qq.com")
	t.Setenv("APP_EMAIL_SMTP_PASSWORD", "mail-auth-code")
	t.Setenv("APP_EMAIL_FROM", "user@qq.com")
	t.Setenv("APP_EMAIL_FROM_NAME", "Notes of Ashen")
	t.Setenv("APP_MEDIA_ROOT", "/data/media")
	t.Setenv("APP_MEDIA_MAX_UPLOAD_BYTES", "10485760")
	t.Setenv("APP_BACKUP_MAX_UPLOAD_BYTES", "1073741824")
	t.Setenv("APP_TRUSTED_PROXY_CIDRS", "172.18.0.0/16")

	if err := c.ApplyEnv(); err != nil {
		t.Fatalf("ApplyEnv() error = %v", err)
	}

	if c.Port != 19001 {
		t.Fatalf("Port = %d, want 19001", c.Port)
	}
	if c.Database.DataSource != "notes:secret@tcp(mysql:3306)/notes_of_ashen" {
		t.Fatalf("Database.DataSource = %q", c.Database.DataSource)
	}
	if c.Redis.Addr != "redis:6379" {
		t.Fatalf("Redis.Addr = %q", c.Redis.Addr)
	}
	if c.RabbitMQ.Enabled {
		t.Fatal("RabbitMQ.Enabled = true, want false")
	}
	if c.Auth.AccessSecret != "production-secret" {
		t.Fatalf("Auth.AccessSecret = %q", c.Auth.AccessSecret)
	}
	if !c.RequirePublicSiteURL {
		t.Fatal("RequirePublicSiteURL = false, want true")
	}
	if !c.Email.Enabled {
		t.Fatal("Email.Enabled = false, want true")
	}
	if c.Email.SMTPHost != "smtp.qq.com" || c.Email.SMTPPort != 465 {
		t.Fatalf("Email SMTP = %s:%d, want smtp.qq.com:465", c.Email.SMTPHost, c.Email.SMTPPort)
	}
	if c.Email.SMTPUsername != "user@qq.com" || c.Email.SMTPPassword != "mail-auth-code" {
		t.Fatalf("Email credentials were not overridden")
	}
	if c.Media.RootDir != "/data/media" || c.Media.MaxUploadBytes != 10485760 || c.Backup.MaxUploadBytes != 1073741824 {
		t.Fatalf("media/backup configuration was not overridden: %+v %+v", c.Media, c.Backup)
	}
	if c.Proxy.TrustedCIDRs != "172.18.0.0/16" {
		t.Fatalf("Proxy.TrustedCIDRs = %q", c.Proxy.TrustedCIDRs)
	}
}

func TestApplyEnvRejectsInvalidInteger(t *testing.T) {
	c := Config{}
	t.Setenv("APP_PORT", "not-a-port")

	if err := c.ApplyEnv(); err == nil {
		t.Fatal("ApplyEnv() error = nil, want error")
	}
}

func TestValidateRejectsDefaultOrMissingSecrets(t *testing.T) {
	tests := []struct {
		name string
		conf Config
	}{
		{
			name: "missing database dsn",
			conf: Config{Auth: AuthConf{AccessSecret: "a-long-enough-secret-value", AccessExpire: 1, RefreshExpire: 1}},
		},
		{
			name: "missing access secret",
			conf: Config{Database: DatabaseConf{DataSource: "user:pass@tcp(mysql:3306)/notes_of_ashen"}},
		},
		{
			name: "default access secret",
			conf: Config{Database: DatabaseConf{DataSource: "user:pass@tcp(mysql:3306)/notes_of_ashen"}, Auth: AuthConf{AccessSecret: "please-change-this-secret-in-production", AccessExpire: 1, RefreshExpire: 1}},
		},
		{
			name: "short access secret",
			conf: Config{Database: DatabaseConf{DataSource: "user:pass@tcp(mysql:3306)/notes_of_ashen"}, Auth: AuthConf{AccessSecret: "short", AccessExpire: 1, RefreshExpire: 1}},
		},
		{
			name: "enabled search with placeholder api key",
			conf: Config{Database: DatabaseConf{DataSource: "user:pass@tcp(mysql:3306)/notes_of_ashen"}, Auth: AuthConf{AccessSecret: "a-long-enough-secret-value", AccessExpire: 1, RefreshExpire: 1}, Search: SearchConf{Enabled: true, MeilisearchAPIKey: "notes_of_ashen_meili_master_key"}},
		},
		{
			name: "enabled rabbitmq with placeholder password",
			conf: Config{Database: DatabaseConf{DataSource: "user:pass@tcp(mysql:3306)/notes_of_ashen"}, Auth: AuthConf{AccessSecret: "a-long-enough-secret-value", AccessExpire: 1, RefreshExpire: 1}, RabbitMQ: RabbitMQConf{Enabled: true, URL: "amqp://user:replace-with-password@rabbitmq:5672/"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.conf.ValidateConfig(); err == nil {
				t.Fatal("ValidateConfig() error = nil, want error")
			}
		})
	}
}

func TestValidateAllowsEnvBackedConfig(t *testing.T) {
	c := Config{
		Database: DatabaseConf{DataSource: "notes_user:strong-db-password@tcp(mysql:3306)/notes_of_ashen"},
		Auth:     AuthConf{AccessSecret: "a-long-enough-secret-value", AccessExpire: 7200, RefreshExpire: 604800},
		Redis:    RedisConf{Addr: "redis:6379", Password: "strong-redis-password"},
		Search:   SearchConf{Enabled: false},
		RabbitMQ: RabbitMQConf{Enabled: true, URL: "amqp://rabbit_user:strong-rabbitmq-password@rabbitmq:5672/"},
	}

	if err := c.ValidateConfig(); err != nil {
		t.Fatalf("ValidateConfig() error = %v, want nil", err)
	}
}

func TestValidateRejectsNonPositiveTokenExpirations(t *testing.T) {
	base := Config{
		Database: DatabaseConf{DataSource: "user:pass@tcp(mysql:3306)/notes_of_ashen"},
		Auth: AuthConf{
			AccessSecret:  "a-long-enough-secret-value",
			AccessExpire:  1,
			RefreshExpire: 1,
		},
		Redis: RedisConf{Addr: "redis:6379"},
	}

	tests := []struct {
		name   string
		mutate func(*Config)
	}{
		{name: "zero access ttl", mutate: func(c *Config) { c.Auth.AccessExpire = 0 }},
		{name: "negative access ttl", mutate: func(c *Config) { c.Auth.AccessExpire = -1 }},
		{name: "zero refresh ttl", mutate: func(c *Config) { c.Auth.RefreshExpire = 0 }},
		{name: "negative refresh ttl", mutate: func(c *Config) { c.Auth.RefreshExpire = -1 }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conf := base
			tt.mutate(&conf)
			if err := conf.ValidateConfig(); err == nil {
				t.Fatal("ValidateConfig() error = nil, want error")
			}
		})
	}
}

func TestValidateOptionalDependencyProfiles(t *testing.T) {
	tests := []struct {
		name      string
		profiles  string
		conf      Config
		wantError string
	}{
		{
			name:      "search enabled without profile",
			profiles:  "",
			conf:      Config{Search: SearchConf{Enabled: true, MeilisearchHost: "http://meilisearch:7700", MeilisearchAPIKey: "strong-meili-key"}},
			wantError: "APP_SEARCH_ENABLED",
		},
		{
			name:     "search enabled with profile",
			profiles: "search",
			conf: Config{Search: SearchConf{
				Enabled: true, MeilisearchHost: "http://meilisearch:7700", MeilisearchAPIKey: "strong-meili-key", MeilisearchIndex: "articles",
			}},
		},
		{
			name:     "rabbitmq enabled with matching credentials",
			profiles: "messaging",
			conf: Config{RabbitMQ: RabbitMQConf{
				Enabled: true, User: "notes_user", Password: "strong-rabbit-password", URL: "amqp://notes_user:strong-rabbit-password@rabbitmq:5672/",
			}},
		},
		{
			name:      "rabbitmq credentials drift",
			profiles:  "messaging",
			conf:      Config{RabbitMQ: RabbitMQConf{Enabled: true, User: "notes_user", Password: "strong-rabbit-password", URL: "amqp://notes_user:another-password@rabbitmq:5672/"}},
			wantError: "credentials must match",
		},
		{
			name:      "disabled search with profile",
			profiles:  "search",
			conf:      Config{},
			wantError: "APP_SEARCH_ENABLED",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("COMPOSE_PROFILES", tt.profiles)
			err := tt.conf.ValidateOptionalDependencyProfiles()
			if tt.wantError == "" {
				if err != nil {
					t.Fatalf("ValidateOptionalDependencyProfiles() error = %v, want nil", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantError) {
				t.Fatalf("ValidateOptionalDependencyProfiles() error = %v, want substring %q", err, tt.wantError)
			}
		})
	}
}
