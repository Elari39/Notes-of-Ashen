-- RAG 的向量索引属于可再生派生数据；MySQL 仅保存可靠同步任务、索引状态与登录用户的私有会话。
-- rag_sync_jobs 不对 article_id 建立外键，使文章删除后仍可保留删除向量的任务。

CREATE TABLE IF NOT EXISTS rag_sync_jobs (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    article_id BIGINT UNSIGNED NOT NULL,
    operation VARCHAR(16) NOT NULL,
    article_version BIGINT UNSIGNED NOT NULL DEFAULT 0,
    run_after DATETIME(6) NOT NULL,
    lease_token CHAR(36) NULL,
    lease_expires_at DATETIME(6) NULL,
    attempts INT UNSIGNED NOT NULL DEFAULT 0,
    last_error TEXT NULL,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    PRIMARY KEY (id),
    UNIQUE KEY uq_rag_sync_jobs_article_id (article_id),
    KEY idx_rag_sync_jobs_run_after (run_after, id),
    KEY idx_rag_sync_jobs_lease_expires_at (lease_expires_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS rag_index_state (
    id TINYINT UNSIGNED NOT NULL,
    status VARCHAR(16) NOT NULL DEFAULT 'needs_rebuild',
    epoch BIGINT UNSIGNED NOT NULL DEFAULT 0,
    embedding_fingerprint CHAR(64) NOT NULL DEFAULT '',
    last_error TEXT NULL,
    indexed_article_count BIGINT UNSIGNED NOT NULL DEFAULT 0,
    indexed_chunk_count BIGINT UNSIGNED NOT NULL DEFAULT 0,
    started_at DATETIME(6) NULL,
    completed_at DATETIME(6) NULL,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    PRIMARY KEY (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

INSERT INTO rag_index_state (id, status)
VALUES (1, 'needs_rebuild')
ON DUPLICATE KEY UPDATE id = id;

-- 游客会话只保存在浏览器内存，落库的会话始终归属于登录用户。
CREATE TABLE IF NOT EXISTS rag_chat_sessions (
    id CHAR(36) NOT NULL,
    user_id BIGINT UNSIGNED NULL,
    title VARCHAR(160) NOT NULL DEFAULT '',
    source_epoch BIGINT UNSIGNED NOT NULL DEFAULT 0,
    expires_at DATETIME(6) NULL,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    PRIMARY KEY (id),
    KEY idx_rag_chat_sessions_user_updated (user_id, updated_at, id),
    KEY idx_rag_chat_sessions_expires_at (expires_at),
    CONSTRAINT fk_rag_chat_sessions_user
        FOREIGN KEY (user_id) REFERENCES users (id)
        ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS rag_chat_messages (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    session_id CHAR(36) NOT NULL,
    role VARCHAR(16) NOT NULL,
    content MEDIUMTEXT NOT NULL,
    sources JSON NULL,
    hidden_at DATETIME(6) NULL,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    PRIMARY KEY (id),
    KEY idx_rag_chat_messages_session_id (session_id, id),
    KEY idx_rag_chat_messages_hidden_at (hidden_at),
    CONSTRAINT fk_rag_chat_messages_session
        FOREIGN KEY (session_id) REFERENCES rag_chat_sessions (id)
        ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

INSERT INTO site_settings (setting_key, setting_value)
VALUES
    ('rag_enabled', 'false'),
    ('rag_chat_page_enabled', 'false'),
    ('rag_chat_nav_hidden', 'true'),
    ('rag_chat_access_level', 'guest'),
    ('rag_chat_base_url', 'https://dashscope.aliyuncs.com/compatible-mode/v1'),
    ('rag_embedding_base_url', 'https://dashscope.aliyuncs.com/compatible-mode/v1'),
    ('rag_rerank_url', 'https://dashscope.aliyuncs.com/api/v1/services/rerank/text-rerank/text-rerank'),
    ('rag_chat_model', 'qwen-plus'),
    ('rag_embedding_model', 'text-embedding-v4'),
    ('rag_embedding_dimensions', '4096'),
    ('rag_rerank_model', 'qwen3-vl-rerank'),
    ('rag_api_key_cipher', ''),
    ('rag_history_retention_days', '90')
ON DUPLICATE KEY UPDATE setting_value = setting_value;
