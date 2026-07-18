package authutil

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestParseAccessTokenAcceptsHS256(t *testing.T) {
	manager := NewManager("test-secret", 3600, 7200)
	token, err := manager.CreateAccessToken(42, RoleAdmin, 3)
	if err != nil {
		t.Fatalf("CreateAccessToken() error = %v", err)
	}

	claims, err := manager.ParseAccessToken(token)
	if err != nil {
		t.Fatalf("ParseAccessToken() error = %v", err)
	}
	if claims.UserID != 42 || claims.Role != RoleAdmin || claims.TokenVersion != 3 {
		t.Fatalf("claims = %#v", claims)
	}
}

func TestParseAccessTokenRejectsDisallowedSigningMethods(t *testing.T) {
	manager := NewManager("test-secret", 3600, 7200)
	claims := Claims{
		UserID:       42,
		Role:         RoleAdmin,
		TokenVersion: 3,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			Subject:   "access",
		},
	}
	tests := []struct {
		name   string
		method jwt.SigningMethod
		key    interface{}
	}{
		{name: "HS384", method: jwt.SigningMethodHS384, key: []byte("test-secret")},
		{name: "none", method: jwt.SigningMethodNone, key: jwt.UnsafeAllowNoneSignatureType},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := jwt.NewWithClaims(tt.method, claims).SignedString(tt.key)
			if err != nil {
				t.Fatalf("SignedString() error = %v", err)
			}
			if _, err := manager.ParseAccessToken(token); err == nil {
				t.Fatal("ParseAccessToken() accepted a disallowed signing method")
			}
		})
	}
}
