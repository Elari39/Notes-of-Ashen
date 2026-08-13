package model

import (
	"context"
	"database/sql/driver"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

// timeAfterArgument 让依赖当前时间的租约 SQL 仍可被严格断言，而不把测试绑定到
// 某个精确的 wall clock 时间点。
type timeAfterArgument struct {
	notBefore time.Time
}

func (argument timeAfterArgument) Match(value driver.Value) bool {
	timestamp, ok := value.(time.Time)
	return ok && timestamp.After(argument.notBefore)
}

func TestEnqueueRAGSyncJobInitializesAndAdvancesArticleVersion(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	defer db.Close()

	const statement = `INSERT INTO rag_sync_jobs (article_id, operation, article_version, run_after, lease_token, lease_expires_at, attempts, last_error)
VALUES (?, ?, 1, ?, NULL, NULL, 0, NULL)
ON DUPLICATE KEY UPDATE operation = VALUES(operation), article_version = article_version + 1, run_after = VALUES(run_after), lease_token = NULL, lease_expires_at = NULL, last_error = NULL`
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(statement)).
		WithArgs(uint64(7), RAGSyncOperationUpsert, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	if err := NewStore(db).EnqueueRAGSyncJob(context.Background(), 7, RAGSyncOperationUpsert, time.Date(2026, time.August, 13, 0, 0, 0, 0, time.UTC)); err != nil {
		t.Fatalf("EnqueueRAGSyncJob() error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestEnqueueRAGSyncJobRejectsInvalidOperation(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectRollback()
	if err := NewStore(db).EnqueueRAGSyncJob(context.Background(), 7, "replace", time.Time{}); err == nil {
		t.Fatal("EnqueueRAGSyncJob() error = nil, want invalid operation error")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestUpdateRAGIndexStatsGuardsReadyEpoch(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	defer db.Close()

	mock.ExpectExec(regexp.QuoteMeta(`
UPDATE rag_index_state
SET indexed_article_count = ?, indexed_chunk_count = ?, last_error = NULL
WHERE id = 1 AND status = ? AND epoch = ?`)).
		WithArgs(uint64(3), uint64(9), RAGIndexStatusReady, uint64(5)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	if err := NewStore(db).UpdateRAGIndexStats(context.Background(), 5, 3, 9); err != nil {
		t.Fatalf("UpdateRAGIndexStats() error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestBeginRAGIndexRebuildTakesOverExpiredLease(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	defer db.Close()

	const fingerprint = "embedding-fingerprint"
	const previousToken = "previous-owner"
	const nextToken = "next-owner"
	leaseDuration := 30 * time.Second
	expiredAt := time.Now().UTC().Add(-time.Minute)

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("INSERT IGNORE INTO rag_index_state (id, status) VALUES (?, ?)")).
		WithArgs(1, RAGIndexStatusNeedsRebuild).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT status, epoch, embedding_fingerprint, rebuild_lease_token, rebuild_lease_expires_at FROM rag_index_state WHERE id = 1 FOR UPDATE")).
		WillReturnRows(sqlmock.NewRows([]string{"status", "epoch", "embedding_fingerprint", "rebuild_lease_token", "rebuild_lease_expires_at"}).
			AddRow(RAGIndexStatusRebuilding, uint64(7), fingerprint, previousToken, expiredAt))
	mock.ExpectExec(regexp.QuoteMeta(`
UPDATE rag_index_state
SET status = ?, epoch = ?, embedding_fingerprint = ?, last_error = NULL,
    indexed_article_count = 0, indexed_chunk_count = 0, started_at = ?, completed_at = NULL,
    rebuild_lease_token = ?, rebuild_lease_expires_at = ?
WHERE id = 1 AND (status <> ? OR rebuild_lease_expires_at IS NULL OR rebuild_lease_expires_at <= NOW(6))`)).
		WithArgs(RAGIndexStatusRebuilding, uint64(8), fingerprint, sqlmock.AnyArg(), nextToken, sqlmock.AnyArg(), RAGIndexStatusRebuilding).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	state, claimed, err := NewStore(db).BeginRAGIndexRebuild(context.Background(), fingerprint, "  "+nextToken+"  ", leaseDuration)
	if err != nil {
		t.Fatalf("BeginRAGIndexRebuild() error = %v", err)
	}
	if !claimed {
		t.Fatal("BeginRAGIndexRebuild() claimed = false, want true after prior lease expiry")
	}
	if state == nil {
		t.Fatal("BeginRAGIndexRebuild() state = nil, want claimed state")
	}
	if state.Status != RAGIndexStatusRebuilding || state.Epoch != 8 || state.EmbeddingFingerprint != fingerprint || state.RebuildLeaseToken != nextToken {
		t.Fatalf("BeginRAGIndexRebuild() state = %#v, want epoch 8 owned by %q", state, nextToken)
	}
	if state.StartedAt == nil || state.RebuildLeaseExpiresAt == nil || !state.RebuildLeaseExpiresAt.After(*state.StartedAt) {
		t.Fatalf("BeginRAGIndexRebuild() lease window is invalid: startedAt=%v expiresAt=%v", state.StartedAt, state.RebuildLeaseExpiresAt)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestRenewRAGIndexRebuildLeaseExtendsCurrentOwnerLease(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	defer db.Close()

	leaseDuration := 30 * time.Second
	notBefore := time.Now().UTC().Add(leaseDuration - time.Second)
	mock.ExpectExec(regexp.QuoteMeta(`
UPDATE rag_index_state SET rebuild_lease_expires_at = ?
WHERE id = 1 AND status = ? AND epoch = ? AND embedding_fingerprint = ?
  AND rebuild_lease_token = ? AND rebuild_lease_expires_at > NOW(6)`)).
		WithArgs(timeAfterArgument{notBefore: notBefore}, RAGIndexStatusRebuilding, uint64(8), "embedding-fingerprint", "current-owner").
		WillReturnResult(sqlmock.NewResult(0, 1))

	renewed, err := NewStore(db).RenewRAGIndexRebuildLease(context.Background(), 8, " embedding-fingerprint ", " current-owner ", leaseDuration)
	if err != nil {
		t.Fatalf("RenewRAGIndexRebuildLease() error = %v", err)
	}
	if !renewed {
		t.Fatal("RenewRAGIndexRebuildLease() renewed = false, want true for current owner")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestRenewRAGIndexRebuildLeaseRejectsExpiredOrTakenOverOwner(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	defer db.Close()

	mock.ExpectExec(regexp.QuoteMeta(`
UPDATE rag_index_state SET rebuild_lease_expires_at = ?
WHERE id = 1 AND status = ? AND epoch = ? AND embedding_fingerprint = ?
  AND rebuild_lease_token = ? AND rebuild_lease_expires_at > NOW(6)`)).
		WithArgs(sqlmock.AnyArg(), RAGIndexStatusRebuilding, uint64(7), "embedding-fingerprint", "previous-owner").
		WillReturnResult(sqlmock.NewResult(0, 0))

	renewed, err := NewStore(db).RenewRAGIndexRebuildLease(context.Background(), 7, "embedding-fingerprint", "previous-owner", time.Minute)
	if err != nil {
		t.Fatalf("RenewRAGIndexRebuildLease() error = %v", err)
	}
	if renewed {
		t.Fatal("RenewRAGIndexRebuildLease() renewed = true, want false after expiry or takeover")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestCompleteRAGIndexRebuildRejectsStaleLeaseHolder(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	defer db.Close()

	mock.ExpectExec(regexp.QuoteMeta(`
UPDATE rag_index_state
SET status = ?, last_error = NULL, indexed_article_count = ?, indexed_chunk_count = ?, completed_at = NOW(6),
    rebuild_lease_token = NULL, rebuild_lease_expires_at = NULL
WHERE id = 1 AND status = ? AND epoch = ? AND embedding_fingerprint = ?
  AND rebuild_lease_token = ? AND rebuild_lease_expires_at > NOW(6)`)).
		WithArgs(RAGIndexStatusReady, uint64(3), uint64(9), RAGIndexStatusRebuilding, uint64(7), "embedding-fingerprint", "previous-owner").
		WillReturnResult(sqlmock.NewResult(0, 0))

	completed, err := NewStore(db).CompleteRAGIndexRebuild(context.Background(), 7, "embedding-fingerprint", "previous-owner", 3, 9)
	if err != nil {
		t.Fatalf("CompleteRAGIndexRebuild() error = %v", err)
	}
	if completed {
		t.Fatal("CompleteRAGIndexRebuild() completed = true, want false for stale lease holder")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestFailRAGIndexRebuildRejectsStaleLeaseHolder(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	defer db.Close()

	mock.ExpectExec(regexp.QuoteMeta(`
UPDATE rag_index_state
SET status = ?, last_error = ?, indexed_article_count = 0, indexed_chunk_count = 0, completed_at = NULL,
    rebuild_lease_token = NULL, rebuild_lease_expires_at = NULL
WHERE id = 1 AND status = ? AND epoch = ? AND embedding_fingerprint = ?
  AND rebuild_lease_token = ? AND rebuild_lease_expires_at > NOW(6)`)).
		WithArgs(RAGIndexStatusError, "upstream unavailable", RAGIndexStatusRebuilding, uint64(7), "embedding-fingerprint", "previous-owner").
		WillReturnResult(sqlmock.NewResult(0, 0))

	failed, err := NewStore(db).FailRAGIndexRebuild(context.Background(), 7, "embedding-fingerprint", "previous-owner", "upstream unavailable")
	if err != nil {
		t.Fatalf("FailRAGIndexRebuild() error = %v", err)
	}
	if failed {
		t.Fatal("FailRAGIndexRebuild() failed = true, want false for stale lease holder")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestReleaseRAGIndexRebuildToPendingRejectsStaleLeaseHolder(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	defer db.Close()

	mock.ExpectExec(regexp.QuoteMeta(`
UPDATE rag_index_state
SET status = ?, last_error = NULL, indexed_article_count = 0, indexed_chunk_count = 0, started_at = NULL, completed_at = NULL,
    rebuild_lease_token = NULL, rebuild_lease_expires_at = NULL
WHERE id = 1 AND status = ? AND epoch = ?
  AND rebuild_lease_token = ? AND rebuild_lease_expires_at > NOW(6)`)).
		WithArgs(RAGIndexStatusNeedsRebuild, RAGIndexStatusRebuilding, uint64(7), "previous-owner").
		WillReturnResult(sqlmock.NewResult(0, 0))

	released, err := NewStore(db).ReleaseRAGIndexRebuildToPending(context.Background(), 7, "previous-owner")
	if err != nil {
		t.Fatalf("ReleaseRAGIndexRebuildToPending() error = %v", err)
	}
	if released {
		t.Fatal("ReleaseRAGIndexRebuildToPending() released = true, want false for stale lease holder")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestTouchRAGChatSessionUpdatesExistingSession(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	defer db.Close()

	mock.ExpectExec(regexp.QuoteMeta("UPDATE rag_chat_sessions SET updated_at = NOW(6) WHERE id = ?")).
		WithArgs("session-1").
		WillReturnResult(sqlmock.NewResult(0, 1))
	if err := NewStore(db).TouchRAGChatSession(context.Background(), "session-1"); err != nil {
		t.Fatalf("TouchRAGChatSession() error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestHideRAGChatMessagesForArticleMatchesNumericSourceID(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	defer db.Close()

	mock.ExpectExec(regexp.QuoteMeta("UPDATE rag_chat_messages SET hidden_at = COALESCE(hidden_at, NOW(6)) WHERE role = 'assistant' AND JSON_CONTAINS(sources, JSON_OBJECT('articleId', CAST(? AS UNSIGNED)))")).
		WithArgs(uint64(42)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	if err := NewStore(db).HideRAGChatMessagesForArticle(context.Background(), 42); err != nil {
		t.Fatalf("HideRAGChatMessagesForArticle() error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
