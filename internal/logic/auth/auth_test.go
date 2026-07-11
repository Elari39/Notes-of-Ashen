package auth

import (
	"errors"
	"testing"
	"time"

	apperrors "notes-of-ashen/internal/errors"
	"notes-of-ashen/model"
)

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
