package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeEnvFile(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write .env: %v", err)
	}
	return path
}

func TestLoadDotEnv_QuotedAndHashInValue(t *testing.T) {
	path := writeEnvFile(t, `# comment line
APP_DISPLAY_NAME="Notes of Ashen"
APP_DATABASE_DSN=user:pass@tcp(host:3306)/db?charset=utf8mb4
APP_REDIS_PASSWORD=redis_r7PfKf
APP_RABBITMQ_URL=amqp://user:Elaina10#0d1017@mq@host:50212/
APP_WITH_INLINE_COMMENT=value # trailing note
APP_EMPTY=
`)

	if err := LoadDotEnv(path); err != nil {
		t.Fatalf("LoadDotEnv: %v", err)
	}

	cases := map[string]string{
		"APP_DISPLAY_NAME":   "Notes of Ashen",
		"APP_DATABASE_DSN":   "user:pass@tcp(host:3306)/db?charset=utf8mb4",
		"APP_REDIS_PASSWORD": "redis_r7PfKf",
		// '#' inside an unquoted value without preceding whitespace must be preserved.
		"APP_RABBITMQ_URL": "amqp://user:Elaina10#0d1017@mq@host:50212/",
		// Inline " #" comment stripped for unquoted values.
		"APP_WITH_INLINE_COMMENT": "value",
		"APP_EMPTY":               "",
	}
	for key, want := range cases {
		if got := os.Getenv(key); got != want {
			t.Errorf("env %s = %q, want %q", key, got, want)
		}
	}
}

func TestLoadDotEnv_DoesNotClobberExisting(t *testing.T) {
	t.Setenv("APP_KEEP_REAL", "from-shell")
	path := writeEnvFile(t, "APP_KEEP_REAL=from-file\nAPP_ONLY_FILE=yes\n")

	if err := LoadDotEnv(path); err != nil {
		t.Fatalf("LoadDotEnv: %v", err)
	}

	if got := os.Getenv("APP_KEEP_REAL"); got != "from-shell" {
		t.Errorf("existing env overwritten: got %q, want from-shell", got)
	}
	if got := os.Getenv("APP_ONLY_FILE"); got != "yes" {
		t.Errorf("file-only env not loaded: got %q, want yes", got)
	}
}

func TestLoadDotEnv_MissingFileIsNoOp(t *testing.T) {
	if err := LoadDotEnv("definitely-not-exists.env"); err != nil {
		t.Errorf("missing file should be no-op, got %v", err)
	}
}

func TestLoadDotEnv_ExportPrefixAndSingleQuote(t *testing.T) {
	path := writeEnvFile(t, "export APP_SINGLE='it''s ok'\nexport APP_NUM=42\n")
	if err := LoadDotEnv(path); err != nil {
		t.Fatalf("LoadDotEnv: %v", err)
	}
	// Single-quoted value keeps inner content verbatim (no escape processing).
	if got := os.Getenv("APP_SINGLE"); got != "it''s ok" {
		t.Errorf("APP_SINGLE = %q, want it''s ok", got)
	}
	if got := os.Getenv("APP_NUM"); got != "42" {
		t.Errorf("APP_NUM = %q, want 42", got)
	}
}

func TestLoadDotEnv_LongValue(t *testing.T) {
	longValue := strings.Repeat("x", 128*1024)
	path := writeEnvFile(t, "APP_LONG_VALUE="+longValue+"\n")

	if err := LoadDotEnv(path); err != nil {
		t.Fatalf("LoadDotEnv: %v", err)
	}
	if got := os.Getenv("APP_LONG_VALUE"); got != longValue {
		t.Fatalf("APP_LONG_VALUE length = %d, want %d", len(got), len(longValue))
	}
}
