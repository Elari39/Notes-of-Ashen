USE notes_of_ashen;

SET @schema_name := DATABASE();

-- article_tags 反向索引：按 tag_id 查文章时避免全表扫描。
-- 与 project_tags 的 idx_project_tags_tag 保持一致。
SET @index_exists := (
  SELECT COUNT(*)
  FROM information_schema.STATISTICS
  WHERE TABLE_SCHEMA = @schema_name
    AND TABLE_NAME = 'article_tags'
    AND INDEX_NAME = 'idx_article_tags_tag'
);
SET @ddl := IF(
  @index_exists = 0,
  'ALTER TABLE article_tags ADD INDEX idx_article_tags_tag (tag_id, article_id)',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;
