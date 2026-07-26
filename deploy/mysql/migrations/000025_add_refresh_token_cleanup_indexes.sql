ALTER TABLE refresh_tokens
    ADD INDEX idx_refresh_tokens_expires_at (expires_at),
    ADD INDEX idx_refresh_tokens_revoked_at (revoked_at);
