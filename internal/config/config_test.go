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
}

func TestApplyEnvRejectsInvalidInteger(t *testing.T) {
	c := Config{}
	t.Setenv("APP_PORT", "not-a-port")

	if err := c.ApplyEnv(); err == nil {
		t.Fatal("ApplyEnv() error = nil, want error")
	}
}
