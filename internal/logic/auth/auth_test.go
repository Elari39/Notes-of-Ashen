package auth

import (
	"testing"
	"time"

	apperrors "notes-of-ashen/internal/errors"
	"notes-of-ashen/model"
)

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

func TestValidateLogoutRefreshTokenRejectsExpiredOrRevokedToken(t *testing.T) {
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
			err := validateLogoutRefreshToken(tt.token, 100, now)
			if err == nil {
				t.Fatal("validateLogoutRefreshToken should reject token")
			}
			codeErr, ok := err.(*apperrors.CodeError)
			if !ok || codeErr.Code != 40100 {
				t.Fatalf("error = %#v, want unauthorized CodeError", err)
			}
		})
	}
}
