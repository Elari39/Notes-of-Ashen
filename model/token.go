package model

import (
	"context"
	"database/sql"
	"time"
)

type RefreshToken struct {
	ID        uint64
	UserID    uint64
	TokenHash string
	ExpiresAt time.Time
	RevokedAt *time.Time
	CreatedAt time.Time
}

func (s *Store) CreateRefreshToken(ctx context.Context, userID uint64, tokenHash string, expiresAt time.Time) error {
	_, err := s.db.ExecContext(ctx, `
INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
VALUES (?, ?, ?)`, userID, tokenHash, expiresAt)
	return err
}

func (s *Store) FindRefreshToken(ctx context.Context, tokenHash string) (*RefreshToken, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT id, user_id, token_hash, expires_at, revoked_at, created_at
FROM refresh_tokens WHERE token_hash = ? LIMIT 1`, tokenHash)
	var token RefreshToken
	var revokedAt sql.NullTime
	if err := row.Scan(&token.ID, &token.UserID, &token.TokenHash, &token.ExpiresAt, &revokedAt, &token.CreatedAt); err != nil {
		return nil, scanErr(err)
	}
	token.RevokedAt = timeFromNull(revokedAt)
	return &token, nil
}

func (s *Store) RevokeRefreshToken(ctx context.Context, tokenHash string) error {
	res, err := s.db.ExecContext(ctx, `
UPDATE refresh_tokens SET revoked_at = NOW()
WHERE token_hash = ? AND revoked_at IS NULL`, tokenHash)
	if err != nil {
		return err
	}
	return requireAffected(res)
}

func (s *Store) RevokeRefreshTokenForUser(ctx context.Context, tokenHash string, userID uint64) error {
	res, err := s.db.ExecContext(ctx, `
UPDATE refresh_tokens SET revoked_at = NOW()
WHERE token_hash = ? AND user_id = ? AND revoked_at IS NULL`, tokenHash, userID)
	if err != nil {
		return err
	}
	return requireAffected(res)
}

func (s *Store) RevokeUserRefreshTokens(ctx context.Context, userID uint64) error {
	_, err := s.db.ExecContext(ctx, `
UPDATE refresh_tokens SET revoked_at = NOW()
WHERE user_id = ? AND revoked_at IS NULL`, userID)
	return err
}
