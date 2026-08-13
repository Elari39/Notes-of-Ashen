-- 重建 collection 是跨实例的排他操作。租约使持有实例异常退出后，其他实例可以
-- 在租约到期后接管；epoch 与 token 同时作为 fencing 条件，旧实例不能再完成或
-- 写入新一轮状态。

ALTER TABLE rag_index_state
    ADD COLUMN rebuild_lease_token CHAR(36) NULL AFTER completed_at,
    ADD COLUMN rebuild_lease_expires_at DATETIME(6) NULL AFTER rebuild_lease_token;
