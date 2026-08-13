package model

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	RAGEnabledKey             = "rag_enabled"
	RAGChatBaseURLKey         = "rag_chat_base_url"
	RAGEmbeddingBaseURLKey    = "rag_embedding_base_url"
	RAGRerankURLKey           = "rag_rerank_url"
	RAGAPIKeyCipherKey        = "rag_api_key_cipher"
	RAGChatModelKey           = "rag_chat_model"
	RAGEmbeddingModelKey      = "rag_embedding_model"
	RAGEmbeddingDimensionsKey = "rag_embedding_dimensions"
	RAGRerankModelKey         = "rag_rerank_model"
	RAGHistoryRetentionKey    = "rag_history_retention_days"

	RAGSyncOperationUpsert = "upsert"
	RAGSyncOperationDelete = "delete"

	RAGIndexStatusNeedsRebuild = "needs_rebuild"
	RAGIndexStatusRebuilding   = "rebuilding"
	RAGIndexStatusReady        = "ready"
	RAGIndexStatusError        = "error"
)

var ErrRAGIndexEpochExhausted = errors.New("rag index epoch is exhausted")

type RAGSettings struct {
	Enabled              bool
	ChatBaseURL          string
	EmbeddingBaseURL     string
	RerankURL            string
	APIKeyCipher         string
	ChatModel            string
	EmbeddingModel       string
	EmbeddingDimensions  int
	RerankModel          string
	HistoryRetentionDays int
}

type RAGSyncJob struct {
	ID             uint64
	ArticleID      uint64
	Operation      string
	ArticleVersion uint64
	RunAfter       time.Time
	LeaseToken     string
	LeaseExpiresAt *time.Time
	Attempts       uint
	LastError      string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type RAGIndexState struct {
	Status                string
	Epoch                 uint64
	EmbeddingFingerprint  string
	LastError             string
	IndexedArticleCount   uint64
	IndexedChunkCount     uint64
	StartedAt             *time.Time
	CompletedAt           *time.Time
	RebuildLeaseToken     string
	RebuildLeaseExpiresAt *time.Time
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

type RAGChatSession struct {
	ID          string
	UserID      uint64
	Title       string
	SourceEpoch uint64
	ExpiresAt   *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type RAGChatMessage struct {
	ID        uint64
	SessionID string
	Role      string
	Content   string
	Sources   json.RawMessage
	HiddenAt  *time.Time
	CreatedAt time.Time
}

func (s *Store) RAGSettings(ctx context.Context) (*RAGSettings, error) {
	keys := []string{RAGEnabledKey, RAGChatBaseURLKey, RAGEmbeddingBaseURLKey, RAGRerankURLKey, RAGAPIKeyCipherKey, RAGChatModelKey, RAGEmbeddingModelKey, RAGEmbeddingDimensionsKey, RAGRerankModelKey, RAGHistoryRetentionKey}
	values, err := s.GetSettingsBatch(ctx, keys)
	if err != nil {
		return nil, err
	}
	get := func(key, fallback string) string {
		if value, ok := values[key]; ok {
			return value
		}
		return fallback
	}
	getBool := func(key string, fallback bool) bool {
		value := strings.ToLower(strings.TrimSpace(get(key, "")))
		if value == "" {
			return fallback
		}
		return value == "true" || value == "1"
	}
	getInt := func(key string, fallback int) int {
		var value int
		if _, err := fmt.Sscanf(strings.TrimSpace(get(key, "")), "%d", &value); err != nil || value < 0 {
			return fallback
		}
		return value
	}
	return &RAGSettings{
		Enabled: getBool(RAGEnabledKey, false), ChatBaseURL: get(RAGChatBaseURLKey, ""), EmbeddingBaseURL: get(RAGEmbeddingBaseURLKey, ""), RerankURL: get(RAGRerankURLKey, ""), APIKeyCipher: get(RAGAPIKeyCipherKey, ""), ChatModel: get(RAGChatModelKey, ""), EmbeddingModel: get(RAGEmbeddingModelKey, ""), EmbeddingDimensions: getInt(RAGEmbeddingDimensionsKey, 1024), RerankModel: get(RAGRerankModelKey, ""), HistoryRetentionDays: getInt(RAGHistoryRetentionKey, 90),
	}, nil
}

func (s *Store) UpdateRAGSettings(ctx context.Context, settings RAGSettings) error {
	_, err := s.db.ExecContext(ctx, `
INSERT INTO site_settings (setting_key, setting_value) VALUES
(?, ?), (?, ?), (?, ?), (?, ?), (?, ?), (?, ?), (?, ?), (?, ?), (?, ?), (?, ?)
ON DUPLICATE KEY UPDATE setting_value = VALUES(setting_value)`,
		RAGEnabledKey, boolSettingValue(settings.Enabled),
		RAGChatBaseURLKey, settings.ChatBaseURL,
		RAGEmbeddingBaseURLKey, settings.EmbeddingBaseURL,
		RAGRerankURLKey, settings.RerankURL,
		RAGAPIKeyCipherKey, settings.APIKeyCipher,
		RAGChatModelKey, settings.ChatModel,
		RAGEmbeddingModelKey, settings.EmbeddingModel,
		RAGEmbeddingDimensionsKey, fmt.Sprintf("%d", settings.EmbeddingDimensions),
		RAGRerankModelKey, settings.RerankModel,
		RAGHistoryRetentionKey, fmt.Sprintf("%d", settings.HistoryRetentionDays),
	)
	return err
}

func (s *Store) EnqueueRAGSyncJob(ctx context.Context, articleID uint64, operation string, runAfter time.Time) error {
	return WithTx(ctx, s.db, func(tx *sql.Tx) error { return enqueueRAGSyncJobTx(ctx, tx, articleID, operation, runAfter) })
}

// enqueueRAGSyncJobTx 与文章写入共享调用，保证文章提交成功就一定有一条最新 outbox。
func enqueueRAGSyncJobTx(ctx context.Context, tx *sql.Tx, articleID uint64, operation string, runAfter time.Time) error {
	if operation != RAGSyncOperationUpsert && operation != RAGSyncOperationDelete {
		return fmt.Errorf("invalid rag sync operation")
	}
	if runAfter.IsZero() {
		runAfter = time.Now().UTC()
	}
	_, err := tx.ExecContext(ctx, `
INSERT INTO rag_sync_jobs (article_id, operation, article_version, run_after, lease_token, lease_expires_at, attempts, last_error)
VALUES (?, ?, 1, ?, NULL, NULL, 0, NULL)
ON DUPLICATE KEY UPDATE operation = VALUES(operation), article_version = article_version + 1, run_after = VALUES(run_after), lease_token = NULL, lease_expires_at = NULL, last_error = NULL`, articleID, operation, runAfter)
	return err
}

func (s *Store) ClaimRAGSyncJobs(ctx context.Context, limit int, leaseToken string, leaseDuration time.Duration) ([]RAGSyncJob, error) {
	if limit < 1 {
		return []RAGSyncJob{}, nil
	}
	if limit > 100 {
		limit = 100
	}
	if strings.TrimSpace(leaseToken) == "" || leaseDuration <= 0 {
		return nil, fmt.Errorf("rag lease is invalid")
	}
	jobs := make([]RAGSyncJob, 0, limit)
	err := WithTx(ctx, s.db, func(tx *sql.Tx) error {
		rows, err := tx.QueryContext(ctx, `SELECT id, article_id, operation, article_version, run_after, lease_token, lease_expires_at, attempts, last_error, created_at, updated_at
FROM rag_sync_jobs WHERE run_after <= NOW(6) AND (lease_expires_at IS NULL OR lease_expires_at <= NOW(6)) ORDER BY run_after, id LIMIT ? FOR UPDATE SKIP LOCKED`, limit)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			job, err := scanRAGSyncJob(rows)
			if err != nil {
				return err
			}
			jobs = append(jobs, *job)
		}
		if err := rows.Err(); err != nil {
			return err
		}
		if len(jobs) == 0 {
			return nil
		}
		expires := time.Now().UTC().Add(leaseDuration)
		for index := range jobs {
			if _, err := tx.ExecContext(ctx, "UPDATE rag_sync_jobs SET lease_token = ?, lease_expires_at = ?, attempts = attempts + 1 WHERE id = ?", leaseToken, expires, jobs[index].ID); err != nil {
				return err
			}
			jobs[index].LeaseToken, jobs[index].LeaseExpiresAt, jobs[index].Attempts = leaseToken, &expires, jobs[index].Attempts+1
		}
		return nil
	})
	return jobs, err
}

func (s *Store) CompleteRAGSyncJob(ctx context.Context, id uint64, leaseToken string) error {
	result, err := s.db.ExecContext(ctx, "DELETE FROM rag_sync_jobs WHERE id = ? AND lease_token = ?", id, leaseToken)
	if err != nil {
		return err
	}
	return requireAffected(result)
}

func (s *Store) FailRAGSyncJob(ctx context.Context, id uint64, leaseToken string, runAfter time.Time, lastError string) error {
	if runAfter.IsZero() {
		runAfter = time.Now().UTC().Add(time.Minute)
	}
	if len(lastError) > 1024 {
		lastError = lastError[:1024]
	}
	result, err := s.db.ExecContext(ctx, "UPDATE rag_sync_jobs SET run_after = ?, lease_token = NULL, lease_expires_at = NULL, last_error = ? WHERE id = ? AND lease_token = ?", runAfter, lastError, id, leaseToken)
	if err != nil {
		return err
	}
	return requireAffected(result)
}

func (s *Store) CountRAGSyncJobs(ctx context.Context) (uint64, error) {
	var count uint64
	err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM rag_sync_jobs").Scan(&count)
	return count, err
}

func (s *Store) RAGIndexState(ctx context.Context) (*RAGIndexState, error) {
	row := s.db.QueryRowContext(ctx, `SELECT status, epoch, embedding_fingerprint, last_error, indexed_article_count, indexed_chunk_count, started_at, completed_at, rebuild_lease_token, rebuild_lease_expires_at, created_at, updated_at FROM rag_index_state WHERE id = 1`)
	return scanRAGIndexState(row)
}

// IsRAGIndexStateCurrent 在向量库写入前后确认本实例仍持有同一索引世代。
// 此检查不能把远程 Qdrant 调用与 MySQL 锁合并为一个事务，但配合 payload 的
// epoch 过滤，可以让跨实例重建期间的旧任务只留下可识别、可清理的旧世代数据。
// rebuilding 状态还必须匹配未过期的 leaseToken，防止旧持有者在租约移交后继续
// 向 Qdrant 写入或错误完成新一轮状态。
func (s *Store) IsRAGIndexStateCurrent(ctx context.Context, status string, epoch uint64, embeddingFingerprint, leaseToken string) (bool, error) {
	var exists bool
	err := s.db.QueryRowContext(ctx, `SELECT EXISTS(
SELECT 1 FROM rag_index_state
WHERE id = 1 AND status = ? AND epoch = ? AND embedding_fingerprint = ?
  AND (status <> ? OR (rebuild_lease_token = ? AND rebuild_lease_expires_at > NOW(6)))
)`, status, epoch, strings.TrimSpace(embeddingFingerprint), RAGIndexStatusRebuilding, strings.TrimSpace(leaseToken)).Scan(&exists)
	return exists, err
}

func (s *Store) UpsertRAGIndexState(ctx context.Context, state RAGIndexState) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO rag_index_state (id, status, epoch, embedding_fingerprint, last_error, indexed_article_count, indexed_chunk_count, started_at, completed_at, rebuild_lease_token, rebuild_lease_expires_at)
VALUES (1, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE status = VALUES(status), epoch = VALUES(epoch), embedding_fingerprint = VALUES(embedding_fingerprint), last_error = VALUES(last_error), indexed_article_count = VALUES(indexed_article_count), indexed_chunk_count = VALUES(indexed_chunk_count), started_at = VALUES(started_at), completed_at = VALUES(completed_at), rebuild_lease_token = VALUES(rebuild_lease_token), rebuild_lease_expires_at = VALUES(rebuild_lease_expires_at)`, state.Status, state.Epoch, state.EmbeddingFingerprint, nullableString(state.LastError), state.IndexedArticleCount, state.IndexedChunkCount, nullableTime(state.StartedAt), nullableTime(state.CompletedAt), nullableString(state.RebuildLeaseToken), nullableTime(state.RebuildLeaseExpiresAt))
	return err
}

// BeginRAGIndexRebuild 通过 rag_index_state 的单行事务锁跨实例领取一次
// collection 重建。有效租约存在时不会抢占；租约到期后以新的 epoch 和 token
// 接管，旧持有者的任何延迟向量写入都不会命中新的检索 epoch。
func (s *Store) BeginRAGIndexRebuild(ctx context.Context, embeddingFingerprint, leaseToken string, leaseDuration time.Duration) (*RAGIndexState, bool, error) {
	embeddingFingerprint = strings.TrimSpace(embeddingFingerprint)
	if embeddingFingerprint == "" {
		return nil, false, fmt.Errorf("rag embedding fingerprint is required")
	}
	leaseToken = strings.TrimSpace(leaseToken)
	if leaseToken == "" || leaseDuration <= 0 {
		return nil, false, fmt.Errorf("rag rebuild lease is invalid")
	}
	var claimed *RAGIndexState
	err := WithTx(ctx, s.db, func(tx *sql.Tx) error {
		// 兼容尚未写入初始行的旧安装；正常迁移会预先创建该行。
		if _, err := tx.ExecContext(ctx, "INSERT IGNORE INTO rag_index_state (id, status) VALUES (?, ?)", 1, RAGIndexStatusNeedsRebuild); err != nil {
			return err
		}
		var current RAGIndexState
		var currentLeaseToken sql.NullString
		var currentLeaseExpiresAt sql.NullTime
		if err := tx.QueryRowContext(ctx, "SELECT status, epoch, embedding_fingerprint, rebuild_lease_token, rebuild_lease_expires_at FROM rag_index_state WHERE id = 1 FOR UPDATE").Scan(&current.Status, &current.Epoch, &current.EmbeddingFingerprint, &currentLeaseToken, &currentLeaseExpiresAt); err != nil {
			return scanErr(err)
		}
		if current.EmbeddingFingerprint != "" && current.EmbeddingFingerprint != embeddingFingerprint {
			return nil
		}
		if current.Epoch == ^uint64(0) {
			return ErrRAGIndexEpochExhausted
		}
		nextEpoch := current.Epoch + 1
		now := time.Now().UTC()
		leaseExpiresAt := now.Add(leaseDuration)
		result, err := tx.ExecContext(ctx, `
UPDATE rag_index_state
SET status = ?, epoch = ?, embedding_fingerprint = ?, last_error = NULL,
    indexed_article_count = 0, indexed_chunk_count = 0, started_at = ?, completed_at = NULL,
    rebuild_lease_token = ?, rebuild_lease_expires_at = ?
WHERE id = 1 AND (status <> ? OR rebuild_lease_expires_at IS NULL OR rebuild_lease_expires_at <= NOW(6))`, RAGIndexStatusRebuilding, nextEpoch, embeddingFingerprint, now, leaseToken, leaseExpiresAt, RAGIndexStatusRebuilding)
		if err != nil {
			return err
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if affected == 0 {
			return nil
		}
		claimed = &RAGIndexState{Status: RAGIndexStatusRebuilding, Epoch: nextEpoch, EmbeddingFingerprint: embeddingFingerprint, StartedAt: &now, RebuildLeaseToken: leaseToken, RebuildLeaseExpiresAt: &leaseExpiresAt}
		return nil
	})
	return claimed, claimed != nil, err
}

// MarkRAGIndexNeedsRebuild 记录最新 embedding 目标。正在运行的实例保留
// rebuilding 状态，但其完成时会因 fingerprint 条件不匹配而原子地降为
// needs_rebuild，避免旧配置错误标记为 ready。
func (s *Store) MarkRAGIndexNeedsRebuild(ctx context.Context, embeddingFingerprint string) error {
	embeddingFingerprint = strings.TrimSpace(embeddingFingerprint)
	if embeddingFingerprint == "" {
		return fmt.Errorf("rag embedding fingerprint is required")
	}
	_, err := s.db.ExecContext(ctx, `
INSERT INTO rag_index_state (id, status, epoch, embedding_fingerprint, last_error, indexed_article_count, indexed_chunk_count, started_at, completed_at)
VALUES (1, ?, 0, ?, NULL, 0, 0, NULL, NULL)
ON DUPLICATE KEY UPDATE
    embedding_fingerprint = VALUES(embedding_fingerprint),
    last_error = IF(status = ?, last_error, NULL),
    indexed_article_count = IF(status = ?, indexed_article_count, 0),
    indexed_chunk_count = IF(status = ?, indexed_chunk_count, 0),
    started_at = IF(status = ?, started_at, NULL),
    completed_at = IF(status = ?, completed_at, NULL),
	 rebuild_lease_token = IF(status = ?, rebuild_lease_token, NULL),
	 rebuild_lease_expires_at = IF(status = ?, rebuild_lease_expires_at, NULL),
    status = IF(status = ?, status, VALUES(status))`,
		RAGIndexStatusNeedsRebuild, embeddingFingerprint,
		RAGIndexStatusRebuilding, RAGIndexStatusRebuilding, RAGIndexStatusRebuilding,
		RAGIndexStatusRebuilding, RAGIndexStatusRebuilding, RAGIndexStatusRebuilding,
		RAGIndexStatusRebuilding, RAGIndexStatusRebuilding,
	)
	return err
}

// CompleteRAGIndexRebuild 仅允许当前 epoch 且仍对应本次 embedding 的所有者把
// 索引标记为 ready。false 表示已被配置变更或另一轮状态转换抢占，调用方必须
// 保持问答不可用并安排最新配置的后续重建。
func (s *Store) CompleteRAGIndexRebuild(ctx context.Context, epoch uint64, embeddingFingerprint, leaseToken string, articleCount, chunkCount uint64) (bool, error) {
	result, err := s.db.ExecContext(ctx, `
UPDATE rag_index_state
SET status = ?, last_error = NULL, indexed_article_count = ?, indexed_chunk_count = ?, completed_at = NOW(6),
    rebuild_lease_token = NULL, rebuild_lease_expires_at = NULL
WHERE id = 1 AND status = ? AND epoch = ? AND embedding_fingerprint = ?
  AND rebuild_lease_token = ? AND rebuild_lease_expires_at > NOW(6)`,
		RAGIndexStatusReady, articleCount, chunkCount, RAGIndexStatusRebuilding, epoch, strings.TrimSpace(embeddingFingerprint), strings.TrimSpace(leaseToken))
	if err != nil {
		return false, err
	}
	affected, err := result.RowsAffected()
	return affected > 0, err
}

// ReleaseRAGIndexRebuildToPending 将当前 epoch 的 running 状态释放为待重建，
// 但保留被配置更新写入的目标 fingerprint。它只影响仍属于调用方 epoch 的行。
func (s *Store) ReleaseRAGIndexRebuildToPending(ctx context.Context, epoch uint64, leaseToken string) (bool, error) {
	result, err := s.db.ExecContext(ctx, `
UPDATE rag_index_state
SET status = ?, last_error = NULL, indexed_article_count = 0, indexed_chunk_count = 0, started_at = NULL, completed_at = NULL,
    rebuild_lease_token = NULL, rebuild_lease_expires_at = NULL
WHERE id = 1 AND status = ? AND epoch = ?
  AND rebuild_lease_token = ? AND rebuild_lease_expires_at > NOW(6)`, RAGIndexStatusNeedsRebuild, RAGIndexStatusRebuilding, epoch, strings.TrimSpace(leaseToken))
	if err != nil {
		return false, err
	}
	affected, err := result.RowsAffected()
	return affected > 0, err
}

// FailRAGIndexRebuild 只让仍拥有 epoch 和 fingerprint 的实例记录失败。若
// fingerprint 已被配置更新，则返回 false，让调用方转为 pending 而非把新目标
// 误标为旧请求的 error。
func (s *Store) FailRAGIndexRebuild(ctx context.Context, epoch uint64, embeddingFingerprint, leaseToken, lastError string) (bool, error) {
	result, err := s.db.ExecContext(ctx, `
UPDATE rag_index_state
SET status = ?, last_error = ?, indexed_article_count = 0, indexed_chunk_count = 0, completed_at = NULL,
    rebuild_lease_token = NULL, rebuild_lease_expires_at = NULL
WHERE id = 1 AND status = ? AND epoch = ? AND embedding_fingerprint = ?
  AND rebuild_lease_token = ? AND rebuild_lease_expires_at > NOW(6)`,
		RAGIndexStatusError, nullableString(lastError), RAGIndexStatusRebuilding, epoch, strings.TrimSpace(embeddingFingerprint), strings.TrimSpace(leaseToken))
	if err != nil {
		return false, err
	}
	affected, err := result.RowsAffected()
	return affected > 0, err
}

// RenewRAGIndexRebuildLease 定期续租长时间的全量重建。false 表示 lease 已到期
// 或已被新实例接管；调用方必须立即取消上游请求并停止写入。
func (s *Store) RenewRAGIndexRebuildLease(ctx context.Context, epoch uint64, embeddingFingerprint, leaseToken string, leaseDuration time.Duration) (bool, error) {
	if strings.TrimSpace(leaseToken) == "" || leaseDuration <= 0 {
		return false, fmt.Errorf("rag rebuild lease is invalid")
	}
	expiresAt := time.Now().UTC().Add(leaseDuration)
	result, err := s.db.ExecContext(ctx, `
UPDATE rag_index_state SET rebuild_lease_expires_at = ?
WHERE id = 1 AND status = ? AND epoch = ? AND embedding_fingerprint = ?
  AND rebuild_lease_token = ? AND rebuild_lease_expires_at > NOW(6)`,
		expiresAt, RAGIndexStatusRebuilding, epoch, strings.TrimSpace(embeddingFingerprint), strings.TrimSpace(leaseToken))
	if err != nil {
		return false, err
	}
	affected, err := result.RowsAffected()
	return affected > 0, err
}

// UpdateRAGIndexStats 只更新仍处于同一 ready epoch 的统计，防止一条较早领取的
// 同步任务在重建开始后把新状态覆盖为旧统计。无匹配行代表重建已接手，可安全忽略。
func (s *Store) UpdateRAGIndexStats(ctx context.Context, epoch, indexedArticleCount, indexedChunkCount uint64) error {
	_, err := s.db.ExecContext(ctx, `
UPDATE rag_index_state
SET indexed_article_count = ?, indexed_chunk_count = ?, last_error = NULL
WHERE id = 1 AND status = ? AND epoch = ?`, indexedArticleCount, indexedChunkCount, RAGIndexStatusReady, epoch)
	return err
}

// ListRAGPublicArticles 返回用于重建和统计的完整公开文章集。通用展示列表有较小
// 的分页上限，不能用于 RAG 全量派生索引。
func (s *Store) ListRAGPublicArticles(ctx context.Context) ([]Article, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT "+articleSelectFields+" FROM articles WHERE status = 'published' AND (scheduled_at IS NULL OR scheduled_at <= NOW()) ORDER BY id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]Article, 0)
	for rows.Next() {
		item, err := scanArticleRows(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	return items, rows.Err()
}

// TouchRAGChatSession 将会话排序时间推进到最新消息写入后，由调用方在两个消息
// 均成功持久化后调用，避免只保存半个 turn 时错误置顶。
func (s *Store) TouchRAGChatSession(ctx context.Context, id string) error {
	result, err := s.db.ExecContext(ctx, "UPDATE rag_chat_sessions SET updated_at = NOW(6) WHERE id = ?", id)
	if err != nil {
		return err
	}
	return requireAffected(result)
}

func (s *Store) CreateRAGChatSession(ctx context.Context, session RAGChatSession) error {
	_, err := s.db.ExecContext(ctx, "INSERT INTO rag_chat_sessions (id, user_id, title, source_epoch, expires_at) VALUES (?, ?, ?, ?, ?)", session.ID, session.UserID, session.Title, session.SourceEpoch, nullableTime(session.ExpiresAt))
	return err
}

func (s *Store) ListRAGChatSessions(ctx context.Context, userID uint64, limit int) ([]RAGChatSession, error) {
	if limit < 1 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}
	rows, err := s.db.QueryContext(ctx, "SELECT id, user_id, title, source_epoch, expires_at, created_at, updated_at FROM rag_chat_sessions WHERE user_id = ? AND (expires_at IS NULL OR expires_at > NOW(6)) ORDER BY updated_at DESC, id DESC LIMIT ?", userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]RAGChatSession, 0)
	for rows.Next() {
		item, err := scanRAGChatSession(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	return items, rows.Err()
}

func (s *Store) FindRAGChatSession(ctx context.Context, id string) (*RAGChatSession, error) {
	return scanRAGChatSession(s.db.QueryRowContext(ctx, "SELECT id, user_id, title, source_epoch, expires_at, created_at, updated_at FROM rag_chat_sessions WHERE id = ?", id))
}

func (s *Store) DeleteRAGChatSession(ctx context.Context, id string, userID uint64) error {
	result, err := s.db.ExecContext(ctx, "DELETE FROM rag_chat_sessions WHERE id = ? AND user_id = ?", id, userID)
	if err != nil {
		return err
	}
	return requireAffected(result)
}

func (s *Store) CreateRAGChatMessage(ctx context.Context, message RAGChatMessage) (uint64, error) {
	result, err := s.db.ExecContext(ctx, "INSERT INTO rag_chat_messages (session_id, role, content, sources, hidden_at) VALUES (?, ?, ?, ?, ?)", message.SessionID, message.Role, message.Content, nullableJSON(message.Sources), nullableTime(message.HiddenAt))
	if err != nil {
		return 0, err
	}
	id, err := result.LastInsertId()
	return uint64(id), err
}

func (s *Store) ListRAGChatMessages(ctx context.Context, sessionID string) ([]RAGChatMessage, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT id, session_id, role, content, sources, hidden_at, created_at FROM rag_chat_messages WHERE session_id = ? ORDER BY id", sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]RAGChatMessage, 0)
	for rows.Next() {
		item, err := scanRAGChatMessage(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	return items, rows.Err()
}

func (s *Store) HideRAGChatMessagesForArticle(ctx context.Context, articleID uint64) error {
	// JSON_CONTAINS 按 JSON 数值比较 articleId。JSON_SEARCH 会将 needle 转为字符串，
	// 无法可靠匹配来源快照中的 JSON number，可能让文章下线后的旧回答继续可见。
	_, err := s.db.ExecContext(ctx, "UPDATE rag_chat_messages SET hidden_at = COALESCE(hidden_at, NOW(6)) WHERE role = 'assistant' AND JSON_CONTAINS(sources, JSON_OBJECT('articleId', CAST(? AS UNSIGNED)))", articleID)
	return err
}

func (s *Store) CleanupExpiredRAGChatSessions(ctx context.Context, cutoff time.Time) (int64, error) {
	result, err := s.db.ExecContext(ctx, "DELETE FROM rag_chat_sessions WHERE expires_at IS NOT NULL AND expires_at <= ?", cutoff)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func scanRAGSyncJob(row rowScanner) (*RAGSyncJob, error) {
	var item RAGSyncJob
	var leaseToken sql.NullString
	var leaseExpires sql.NullTime
	var lastError sql.NullString
	err := row.Scan(&item.ID, &item.ArticleID, &item.Operation, &item.ArticleVersion, &item.RunAfter, &leaseToken, &leaseExpires, &item.Attempts, &lastError, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return nil, scanErr(err)
	}
	item.LeaseToken = stringFromNull(leaseToken)
	item.LeaseExpiresAt = timeFromNull(leaseExpires)
	item.LastError = stringFromNull(lastError)
	return &item, nil
}

func scanRAGIndexState(row rowScanner) (*RAGIndexState, error) {
	var item RAGIndexState
	var lastError, rebuildLeaseToken sql.NullString
	var startedAt, completedAt, rebuildLeaseExpiresAt sql.NullTime
	err := row.Scan(&item.Status, &item.Epoch, &item.EmbeddingFingerprint, &lastError, &item.IndexedArticleCount, &item.IndexedChunkCount, &startedAt, &completedAt, &rebuildLeaseToken, &rebuildLeaseExpiresAt, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return nil, scanErr(err)
	}
	item.LastError = stringFromNull(lastError)
	item.StartedAt = timeFromNull(startedAt)
	item.CompletedAt = timeFromNull(completedAt)
	item.RebuildLeaseToken = stringFromNull(rebuildLeaseToken)
	item.RebuildLeaseExpiresAt = timeFromNull(rebuildLeaseExpiresAt)
	return &item, nil
}

func scanRAGChatSession(row rowScanner) (*RAGChatSession, error) {
	var item RAGChatSession
	var userID sql.NullInt64
	var expiresAt sql.NullTime
	err := row.Scan(&item.ID, &userID, &item.Title, &item.SourceEpoch, &expiresAt, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return nil, scanErr(err)
	}
	item.UserID = uint64FromNull(userID)
	item.ExpiresAt = timeFromNull(expiresAt)
	return &item, nil
}

func scanRAGChatMessage(row rowScanner) (*RAGChatMessage, error) {
	var item RAGChatMessage
	var sources []byte
	var hiddenAt sql.NullTime
	err := row.Scan(&item.ID, &item.SessionID, &item.Role, &item.Content, &sources, &hiddenAt, &item.CreatedAt)
	if err != nil {
		return nil, scanErr(err)
	}
	item.Sources = append(item.Sources[:0], sources...)
	item.HiddenAt = timeFromNull(hiddenAt)
	return &item, nil
}

func nullableString(value string) interface{} {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}
func nullableJSON(value json.RawMessage) interface{} {
	if len(value) == 0 || string(value) == "null" {
		return nil
	}
	return []byte(value)
}
