-- ============================================================
-- 脚本：add_article_category_author_index.sql
-- 作用：补 articles 表的 category_id、author_id 索引
-- 背景：articleWhere 按 category_id / author_id 过滤但无对应索引，
--       数据量增长后按分类筛选或后台按作者筛选会退化为全表扫描。
-- 前置依赖：schema.sql（articles 表必须已存在）
-- ============================================================
USE notes_of_ashen;

SET @schema_name := DATABASE();

-- idx_articles_category (category_id, status, scheduled_at)
SET @index_exists := (
  SELECT COUNT(*)
  FROM information_schema.STATISTICS
  WHERE TABLE_SCHEMA = @schema_name
    AND TABLE_NAME = 'articles'
    AND INDEX_NAME = 'idx_articles_category'
);
SET @ddl := IF(
  @index_exists = 0,
  'ALTER TABLE articles ADD INDEX idx_articles_category (category_id, status, scheduled_at)',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- idx_articles_author (author_id, status)
SET @index_exists := (
  SELECT COUNT(*)
  FROM information_schema.STATISTICS
  WHERE TABLE_SCHEMA = @schema_name
    AND TABLE_NAME = 'articles'
    AND INDEX_NAME = 'idx_articles_author'
);
SET @ddl := IF(
  @index_exists = 0,
  'ALTER TABLE articles ADD INDEX idx_articles_author (author_id, status)',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;
