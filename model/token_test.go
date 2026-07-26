package model

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestCleanupRefreshTokensUsesExpiryAndRevocationCutoffs(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock: %v", err)
	}
	defer db.Close()

	now := time.Date(2026, 7, 26, 12, 0, 0, 0, time.UTC)
	revokedBefore := now.Add(-30 * 24 * time.Hour)
	mock.ExpectExec(regexp.QuoteMeta(`
DELETE FROM refresh_tokens
WHERE expires_at <= ?
   OR (revoked_at IS NOT NULL AND revoked_at <= ?)`)).
		WithArgs(now, revokedBefore).
		WillReturnResult(sqlmock.NewResult(0, 3))

	deleted, err := NewStore(db).CleanupRefreshTokens(context.Background(), now, revokedBefore)
	if err != nil {
		t.Fatalf("CleanupRefreshTokens() error = %v", err)
	}
	if deleted != 3 {
		t.Fatalf("CleanupRefreshTokens() deleted = %d, want 3", deleted)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}
