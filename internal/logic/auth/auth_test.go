package auth

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"notes-of-ashen/internal/authutil"
	apperrors "notes-of-ashen/internal/errors"
	"notes-of-ashen/internal/svc"
	"notes-of-ashen/internal/types"
	"notes-of-ashen/model"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/redis/go-redis/v9"
)

type evalResultHook struct {
	result int64
}

func (h evalResultHook) DialHook(next redis.DialHook) redis.DialHook {
	return next
}

func (h evalResultHook) ProcessHook(redis.ProcessHook) redis.ProcessHook {
	return func(_ context.Context, cmd redis.Cmder) error {
		result, ok := cmd.(*redis.Cmd)
		if !ok {
			return fmt.Errorf("unexpected redis command %T", cmd)
		}
		result.SetVal(h.result)
		return nil
	}
}

func (h evalResultHook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return next
}

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

func (c *recordingExecConn) QueryContext(_ context.Context, query string, _ []driver.NamedValue) (driver.Rows, error) {
	c.driver.mu.Lock()
	c.driver.queries = append(c.driver.queries, query)
	c.driver.mu.Unlock()
	return &singleTokenVersionRow{}, nil
}

type singleTokenVersionRow struct{ read bool }

func (*singleTokenVersionRow) Columns() []string { return []string{"token_version"} }
func (*singleTokenVersionRow) Close() error      { return nil }
func (r *singleTokenVersionRow) Next(dest []driver.Value) error {
	if r.read {
		return io.EOF
	}
	r.read = true
	dest[0] = int64(0)
	return nil
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
	if len(queries) != 2 || !strings.Contains(queries[0], "SELECT token_version") || !strings.Contains(queries[1], "INSERT INTO refresh_tokens") {
		t.Fatalf("database queries = %#v, want token version SELECT and refresh token INSERT", queries)
	}
}

func TestIssueTokensFailsWhenRefreshTokenDatabaseWriteFails(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	databaseErr := errors.New("database unavailable")
	mock.ExpectQuery("SELECT token_version FROM users").
		WithArgs(uint64(42)).
		WillReturnRows(sqlmock.NewRows([]string{"token_version"}).AddRow(uint64(0)))
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

func TestResetPasswordRejectsInvalidCodeBeforeLookingUpAccount(t *testing.T) {
	for name, result := range map[string]int64{"expired": 0, "incorrect": -1} {
		t.Run(name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()

			redisClient := redis.NewClient(&redis.Options{Addr: "unused:6379"})
			redisClient.AddHook(evalResultHook{result: result})
			defer redisClient.Close()

			err = ResetPassword(context.Background(), &svc.ServiceContext{
				Store: model.NewStore(db),
				Redis: redisClient,
			}, types.ResetPasswordReq{
				Email:       "missing@example.com",
				EmailCode:   "123456",
				NewPassword: "new-password",
			}, types.RequestMeta{})
			var codeErr *apperrors.CodeError
			if !errors.As(err, &codeErr) || codeErr.StatusCode != 400 || codeErr.Message != "email code is invalid or expired" {
				t.Fatalf("ResetPassword() error = %#v", err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatalf("password reset queried account before code validation: %v", err)
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

func TestLogoutRevokesRefreshTokenWithoutAccessTokenContext(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock: %v", err)
	}
	defer db.Close()

	refreshToken := "refresh-token"
	hash := authutil.HashRefreshToken(refreshToken)
	now := time.Now()
	mock.ExpectQuery("SELECT id, user_id, token_hash, expires_at, revoked_at, created_at").
		WithArgs(hash).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "token_hash", "expires_at", "revoked_at", "created_at"}).
			AddRow(uint64(1), uint64(42), hash, now.Add(time.Hour), nil, now))
	mock.ExpectExec("UPDATE refresh_tokens SET revoked_at = NOW").
		WithArgs(hash, uint64(42)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	redisErr := errors.New("redis unavailable")
	redisClient := redis.NewClient(&redis.Options{
		Addr:       "redis.invalid:6379",
		MaxRetries: 0,
		Dialer: func(context.Context, string, string) (net.Conn, error) {
			return nil, redisErr
		},
	})
	defer redisClient.Close()

	err = Logout(context.Background(), &svc.ServiceContext{
		Store: model.NewStore(db),
		Redis: redisClient,
	}, types.RefreshReq{RefreshToken: refreshToken}, types.RequestMeta{})
	if err != nil {
		t.Fatalf("Logout() error = %v, want nil", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("database expectations: %v", err)
	}
}
