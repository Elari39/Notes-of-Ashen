package security

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/redis/go-redis/v9"
)

type atomicCodeEvaluator struct {
	mu    sync.Mutex
	value string
}

func (e *atomicCodeEvaluator) Eval(_ context.Context, _ string, _ []string, args ...interface{}) *redis.Cmd {
	e.mu.Lock()
	defer e.mu.Unlock()
	expected, _ := args[0].(string)
	if e.value == "" {
		return redis.NewCmdResult(int64(0), nil)
	}
	if e.value != expected {
		return redis.NewCmdResult(int64(-1), nil)
	}
	e.value = ""
	return redis.NewCmdResult(int64(1), nil)
}

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

func TestConsumeCodeIsSingleUseUnderConcurrency(t *testing.T) {
	evaluator := &atomicCodeEvaluator{value: "123456"}
	const workers = 12
	results := make(chan int64, workers)
	errorsCh := make(chan error, workers)
	var group sync.WaitGroup
	for range workers {
		group.Add(1)
		go func() {
			defer group.Done()
			result, err := consumeCode(context.Background(), evaluator, "verify:test", "123456")
			results <- result
			errorsCh <- err
		}()
	}
	group.Wait()
	close(results)
	close(errorsCh)

	successes := 0
	for result := range results {
		if result == 1 {
			successes++
		}
	}
	if successes != 1 {
		t.Fatalf("successful consumes = %d, want 1", successes)
	}
	for err := range errorsCh {
		if err != nil {
			t.Fatalf("consumeCode() error = %v", err)
		}
	}
}

func TestConsumeCodeRejectsUnexpectedRedisResult(t *testing.T) {
	evaluator := redisEvalFunc(func(context.Context, string, []string, ...interface{}) *redis.Cmd {
		return redis.NewCmdResult(int64(2), nil)
	})
	_, err := consumeCode(context.Background(), evaluator, "verify:test", "123456")
	if err == nil || !strings.Contains(err.Error(), "unexpected code consume result") {
		t.Fatalf("consumeCode() error = %v", err)
	}
}

func TestConsumeCodePropagatesRedisError(t *testing.T) {
	want := errors.New("redis unavailable")
	evaluator := redisEvalFunc(func(context.Context, string, []string, ...interface{}) *redis.Cmd {
		return redis.NewCmdResult(nil, want)
	})
	_, err := consumeCode(context.Background(), evaluator, "verify:test", "123456")
	if !errors.Is(err, want) {
		t.Fatalf("consumeCode() error = %v, want %v", err, want)
	}
}

type redisEvalFunc func(context.Context, string, []string, ...interface{}) *redis.Cmd

func (fn redisEvalFunc) Eval(ctx context.Context, script string, keys []string, args ...interface{}) *redis.Cmd {
	return fn(ctx, script, keys, args...)
}
