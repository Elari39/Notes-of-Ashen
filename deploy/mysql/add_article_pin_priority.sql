-- ============================================================
-- 脚本：add_article_pin_priority.sql
-- 作用：补 articles / article_versions 的 is_pinned、display_priority 列
-- 前置依赖：add_content_growth_features.sql（article_versions 表必须已存在）
-- ============================================================
USE notes_of_ashen;

SET @schema_name := DATABASE();

SET @column_exists := (
  SELECT COUNT(*)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = @schema_name
    AND TABLE_NAME = 'articles'
    AND COLUMN_NAME = 'is_pinned'
);
SET @ddl := IF(
  @column_exists = 0,
  'ALTER TABLE articles ADD COLUMN is_pinned TINYINT(1) NOT NULL DEFAULT 0 AFTER published_at',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @column_exists := (
  SELECT COUNT(*)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = @schema_name
    AND TABLE_NAME = 'articles'
    AND COLUMN_NAME = 'display_priority'
);
SET @ddl := IF(
  @column_exists = 0,
  'ALTER TABLE articles ADD COLUMN display_priority INT NOT NULL DEFAULT 0 AFTER is_pinned',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @index_exists := (
  SELECT COUNT(*)
  FROM information_schema.STATISTICS
  WHERE TABLE_SCHEMA = @schema_name
    AND TABLE_NAME = 'articles'
    AND INDEX_NAME = 'idx_articles_display_order'
);
SET @ddl := IF(
  @index_exists = 0,
  'ALTER TABLE articles ADD INDEX idx_articles_display_order (status, scheduled_at, is_pinned, display_priority, published_at, id)',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @column_exists := (
  SELECT COUNT(*)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = @schema_name
    AND TABLE_NAME = 'article_versions'
    AND COLUMN_NAME = 'is_pinned'
);
SET @ddl := IF(
  @column_exists = 0,
  'ALTER TABLE article_versions ADD COLUMN is_pinned TINYINT(1) NOT NULL DEFAULT 0 AFTER published_at',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @column_exists := (
  SELECT COUNT(*)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = @schema_name
    AND TABLE_NAME = 'article_versions'
    AND COLUMN_NAME = 'display_priority'
);
SET @ddl := IF(
  @column_exists = 0,
  'ALTER TABLE article_versions ADD COLUMN display_priority INT NOT NULL DEFAULT 0 AFTER is_pinned',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;
