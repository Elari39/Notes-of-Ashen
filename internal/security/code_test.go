package security

import (
	"strings"
	"testing"
)

func TestSecurityKeys(t *testing.T) {
	if got := EmailCodeKey("register", " User@QQ.COM "); got != "verify_code:register:user@qq.com" {
		t.Fatalf("EmailCodeKey() = %q", got)
	}
	if got := CaptchaKey("login", "abc"); got != "captcha:login:abc" {
		t.Fatalf("CaptchaKey() = %q", got)
	}
	if got := RateLimitKey("auth_login", "127.0.0.1"); got != "rate_limit:auth_login:127.0.0.1" {
		t.Fatalf("RateLimitKey() = %q", got)
	}
}

func TestRandomDigits(t *testing.T) {
	code, err := RandomDigits(6)
	if err != nil {
		t.Fatalf("RandomDigits() error = %v", err)
	}
	if len(code) != 6 {
		t.Fatalf("code length = %d, want 6", len(code))
	}
	if strings.Trim(code, "0123456789") != "" {
		t.Fatalf("code = %q, want only digits", code)
	}
}

func TestNormalizePurposeRejectsInvalidPurpose(t *testing.T) {
	if _, err := NormalizePurpose("comment"); err == nil {
		t.Fatal("NormalizePurpose() error = nil, want error")
	}
	if _, err := NormalizeEmailPurpose("login"); err == nil {
		t.Fatal("NormalizeEmailPurpose() error = nil, want error")
	}
}
