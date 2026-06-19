package config

import "testing"

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
	t.Setenv("APP_EMAIL_ENABLED", "true")
	t.Setenv("APP_EMAIL_SMTP_HOST", "smtp.qq.com")
	t.Setenv("APP_EMAIL_SMTP_PORT", "465")
	t.Setenv("APP_EMAIL_SMTP_USERNAME", "user@qq.com")
	t.Setenv("APP_EMAIL_SMTP_PASSWORD", "mail-auth-code")
	t.Setenv("APP_EMAIL_FROM", "user@qq.com")
	t.Setenv("APP_EMAIL_FROM_NAME", "Notes of Ashen")
	t.Setenv("APP_AI_ENABLED", "true")
	t.Setenv("APP_AI_BASE_URL", "https://api.example.com/v1")
	t.Setenv("APP_AI_API_KEY", "ai-key")
	t.Setenv("APP_AI_MODEL", "chat-model")
	t.Setenv("APP_AI_KEY_ENCRYPTION_SECRET", "ai-encryption-secret")
	t.Setenv("APP_AI_TIMEOUT_SECONDS", "45")
	t.Setenv("APP_AI_FIRST_BYTE_TIMEOUT_SECONDS", "60")
	t.Setenv("APP_AI_STREAM_TIMEOUT_SECONDS", "300")
	t.Setenv("APP_AI_NON_STREAM_TIMEOUT_SECONDS", "600")
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
	if !c.Email.Enabled {
		t.Fatal("Email.Enabled = false, want true")
	}
	if c.Email.SMTPHost != "smtp.qq.com" || c.Email.SMTPPort != 465 {
		t.Fatalf("Email SMTP = %s:%d, want smtp.qq.com:465", c.Email.SMTPHost, c.Email.SMTPPort)
	}
	if c.Email.SMTPUsername != "user@qq.com" || c.Email.SMTPPassword != "mail-auth-code" {
		t.Fatalf("Email credentials were not overridden")
	}
	if !c.AI.Enabled || c.AI.BaseURL != "https://api.example.com/v1" || c.AI.APIKey != "ai-key" || c.AI.Model != "chat-model" || c.AI.KeyEncryptionSecret != "ai-encryption-secret" || c.AI.TimeoutSeconds != 45 || c.AI.FirstByteTimeoutSeconds != 60 || c.AI.StreamTimeoutSeconds != 300 || c.AI.NonStreamTimeoutSeconds != 600 {
		t.Fatalf("AI config was not overridden: %#v", c.AI)
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
			conf: Config{Auth: AuthConf{AccessSecret: "a-long-enough-secret-value"}},
		},
		{
			name: "missing access secret",
			conf: Config{Database: DatabaseConf{DataSource: "user:pass@tcp(mysql:3306)/notes_of_ashen"}},
		},
		{
			name: "default access secret",
			conf: Config{Database: DatabaseConf{DataSource: "user:pass@tcp(mysql:3306)/notes_of_ashen"}, Auth: AuthConf{AccessSecret: "please-change-this-secret-in-production"}},
		},
		{
			name: "short access secret",
			conf: Config{Database: DatabaseConf{DataSource: "user:pass@tcp(mysql:3306)/notes_of_ashen"}, Auth: AuthConf{AccessSecret: "short"}},
		},
		{
			name: "enabled search with placeholder api key",
			conf: Config{Database: DatabaseConf{DataSource: "user:pass@tcp(mysql:3306)/notes_of_ashen"}, Auth: AuthConf{AccessSecret: "a-long-enough-secret-value"}, Search: SearchConf{Enabled: true, MeilisearchAPIKey: "notes_of_ashen_meili_master_key"}},
		},
		{
			name: "enabled rabbitmq with placeholder password",
			conf: Config{Database: DatabaseConf{DataSource: "user:pass@tcp(mysql:3306)/notes_of_ashen"}, Auth: AuthConf{AccessSecret: "a-long-enough-secret-value"}, RabbitMQ: RabbitMQConf{Enabled: true, URL: "amqp://user:replace-with-password@rabbitmq:5672/"}},
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
		Auth:     AuthConf{AccessSecret: "a-long-enough-secret-value"},
		Redis:    RedisConf{Addr: "redis:6379", Password: "strong-redis-password"},
		Search:   SearchConf{Enabled: false},
		RabbitMQ: RabbitMQConf{Enabled: true, URL: "amqp://rabbit_user:strong-rabbitmq-password@rabbitmq:5672/"},
	}

	if err := c.ValidateConfig(); err != nil {
		t.Fatalf("ValidateConfig() error = %v, want nil", err)
	}
}
