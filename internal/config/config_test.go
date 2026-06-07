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
}

func TestApplyEnvRejectsInvalidInteger(t *testing.T) {
	c := Config{}
	t.Setenv("APP_PORT", "not-a-port")

	if err := c.ApplyEnv(); err == nil {
		t.Fatal("ApplyEnv() error = nil, want error")
	}
}
