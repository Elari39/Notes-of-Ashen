-- DashScope text-embedding-v4 的合法输出维度为
-- 64/128/256/512/768/1024/1536/2048/3072，4096 不在其中；000026 写入的
-- 默认值 4096 会导致按默认配置首次开启 RAG 时 embedding 请求被上游 400 拒绝、
-- 索引重建必然失败。本迁移把仍为 4096 的默认值修正为兼容所有 text-embedding
-- 系列的 1024；管理员已保存的其他合法自定义值不受影响。

INSERT INTO site_settings (setting_key, setting_value)
VALUES ('rag_embedding_dimensions', '1024')
ON DUPLICATE KEY UPDATE setting_value = CASE WHEN setting_value IN ('4096') THEN '1024' ELSE setting_value END;
