package auth

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"notes-of-ashen/internal/authutil"
	apperrors "notes-of-ashen/internal/errors"
	"notes-of-ashen/internal/svc"
	"notes-of-ashen/model"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/redis/go-redis/v9"
)

var recordingDriverSequence atomic.Uint64

type recordingExecDriver struct {
	mu      sync.Mutex
	queries []string
}

func (d *recordingExecDriver) Open(string) (driver.Conn, error) {
	return &recordingExecConn{driver: d}, nil
}

func (d *recordingExecDriver) recordedQueries() []string {
	d.mu.Lock()
	defer d.mu.Unlock()
	return append([]string(nil), d.queries...)
}

type recordingExecConn struct {
	driver *recordingExecDriver
}

func (c *recordingExecConn) Prepare(string) (driver.Stmt, error) {
	return nil, driver.ErrSkip
}

func (c *recordingExecConn) Close() error { return nil }

func (c *recordingExecConn) Begin() (driver.Tx, error) { return nil, driver.ErrSkip }

func (c *recordingExecConn) ExecContext(_ context.Context, query string, _ []driver.NamedValue) (driver.Result, error) {
	c.driver.mu.Lock()
	c.driver.queries = append(c.driver.queries, query)
	c.driver.mu.Unlock()
	return driver.RowsAffected(1), nil
}

func TestIssueTokensSucceedsWhenRefreshTokenCacheWriteFails(t *testing.T) {
	recordingDriver := &recordingExecDriver{}
	driverName := fmt.Sprintf("auth-recording-%d", recordingDriverSequence.Add(1))
	sql.Register(driverName, recordingDriver)
	db, err := sql.Open(driverName, "")
	if err != nil {
		t.Fatalf("open recording database: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	redisErr := errors.New("redis unavailable")
	redisClient := redis.NewClient(&redis.Options{
		Addr:       "redis.invalid:6379",
		MaxRetries: 0,
		Dialer: func(context.Context, string, string) (net.Conn, error) {
			return nil, redisErr
		},
	})
	t.Cleanup(func() { _ = redisClient.Close() })

	svcCtx := &svc.ServiceContext{
		Store:  model.NewStore(db),
		Redis:  redisClient,
		Tokens: authutil.NewManager("test-secret", 60, 3600),
	}

	pair, err := issueTokens(context.Background(), svcCtx, 42, "user")
	queries := recordingDriver.recordedQueries()
	for _, query := range queries {
		if strings.Contains(query, "UPDATE refresh_tokens SET revoked_at") {
			t.Fatalf("database refresh token was revoked after redis cache failure: %s", query)
		}
	}
	if err != nil {
		t.Fatalf("issueTokens should fall back to db when redis write fails: %v", err)
	}
	if pair == nil || pair.AccessToken == "" || pair.RefreshToken == "" {
		t.Fatalf("issueTokens returned invalid token pair: %#v", pair)
	}
	if len(queries) != 1 || !strings.Contains(queries[0], "INSERT INTO refresh_tokens") {
		t.Fatalf("database queries = %#v, want only refresh token INSERT", queries)
	}
}

func TestIssueTokensFailsWhenRefreshTokenDatabaseWriteFails(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	databaseErr := errors.New("database unavailable")
	mock.ExpectExec("INSERT INTO refresh_tokens").
		WithArgs(uint64(42), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnError(databaseErr)

	svcCtx := &svc.ServiceContext{
		Store:  model.NewStore(db),
		Tokens: authutil.NewManager("test-secret", 60, 3600),
	}

	pair, err := issueTokens(context.Background(), svcCtx, 42, "user")
	if pair != nil {
		t.Fatalf("issueTokens pair = %#v, want nil", pair)
	}
	if !errors.Is(err, databaseErr) {
		t.Fatalf("issueTokens error = %v, want %v", err, databaseErr)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("database expectations: %v", err)
	}
}

func TestShouldSendResetPasswordCodeHidesAccountState(t *testing.T) {
	systemErr := errors.New("database unavailable")
	tests := []struct {
		name     string
		user     *model.User
		err      error
		wantSend bool
		wantErr  bool
	}{
		{name: "missing account", err: model.ErrNotFound},
		{name: "disabled account", user: &model.User{Status: "disabled"}},
		{name: "active account", user: &model.User{Status: "active"}, wantSend: true},
		{name: "system error", err: systemErr, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			send, err := shouldSendResetPasswordCode(tt.user, tt.err)
			if send != tt.wantSend {
				t.Fatalf("send = %v, want %v", send, tt.wantSend)
			}
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateLogoutRefreshTokenRejectsOtherUserToken(t *testing.T) {
	token := &model.RefreshToken{
		UserID:    100,
		ExpiresAt: time.Now().Add(time.Hour),
	}

	err := validateLogoutRefreshToken(token, 200, time.Now())
	if err == nil {
		t.Fatal("validateLogoutRefreshToken should reject another user's refresh token")
	}
	codeErr, ok := err.(*apperrors.CodeError)
	if !ok || codeErr.Code != 40100 {
		t.Fatalf("error = %#v, want unauthorized CodeError", err)
	}
}

func TestValidateLogoutRefreshTokenAllowsOwnedActiveToken(t *testing.T) {
	now := time.Now()
	token := &model.RefreshToken{
		UserID:    100,
		ExpiresAt: now.Add(time.Hour),
	}

	if err := validateLogoutRefreshToken(token, 100, now); err != nil {
		t.Fatalf("validateLogoutRefreshToken returned error: %v", err)
	}
}

// TestValidateLogoutRefreshTokenIsIdempotentForExpiredOrRevokedToken 验证过期/已撤销 token
// 登出幂等成功（P2-4）：归属正确时不再返回 401，避免前端在 Cookie 过期后登出反复重试。
func TestValidateLogoutRefreshTokenIsIdempotentForExpiredOrRevokedToken(t *testing.T) {
	now := time.Now()
	revokedAt := now.Add(-time.Minute)
	tests := []struct {
		name  string
		token *model.RefreshToken
	}{
		{
			name: "expired",
			token: &model.RefreshToken{
				UserID:    100,
				ExpiresAt: now.Add(-time.Minute),
			},
		},
		{
			name: "revoked",
			token: &model.RefreshToken{
				UserID:    100,
				ExpiresAt: now.Add(time.Hour),
				RevokedAt: &revokedAt,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := validateLogoutRefreshToken(tt.token, 100, now); err != nil {
				t.Fatalf("validateLogoutRefreshToken should be idempotent, got error: %v", err)
			}
		})
	}
}
