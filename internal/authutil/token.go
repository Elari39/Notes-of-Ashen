package authutil

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Manager struct {
	secret        []byte
	accessExpire  time.Duration
	refreshExpire time.Duration
}

type Claims struct {
	UserID       uint64 `json:"userId"`
	Role         string `json:"role"`
	TokenVersion uint64 `json:"tokenVersion"`
	jwt.RegisteredClaims
}

func NewManager(secret string, accessExpire, refreshExpire int64) *Manager {
	return &Manager{
		secret:        []byte(secret),
		accessExpire:  time.Duration(accessExpire) * time.Second,
		refreshExpire: time.Duration(refreshExpire) * time.Second,
	}
}

func (m *Manager) AccessExpiresIn() int64 {
	return int64(m.accessExpire / time.Second)
}

func (m *Manager) RefreshTTL() time.Duration {
	return m.refreshExpire
}

func (m *Manager) CreateAccessToken(userID uint64, role string, tokenVersion uint64) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID:       userID,
		Role:         role,
		TokenVersion: tokenVersion,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(m.accessExpire)),
			IssuedAt:  jwt.NewNumericDate(now),
			Subject:   "access",
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(m.secret)
}

func (m *Manager) ParseAccessToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(
		tokenString,
		&Claims{},
		func(token *jwt.Token) (interface{}, error) {
			return m.secret, nil
		},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
	)
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, jwt.ErrTokenInvalidClaims
	}
	return claims, nil
}

func (m *Manager) CreateRefreshToken() (string, string, time.Time, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", "", time.Time{}, err
	}
	token := hex.EncodeToString(raw)
	return token, HashRefreshToken(token), time.Now().Add(m.refreshExpire), nil
}

func HashRefreshToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
