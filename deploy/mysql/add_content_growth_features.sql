USE notes_of_ashen;

SET @schema_name := DATABASE();

SET @column_exists := (
  SELECT COUNT(*)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = @schema_name
    AND TABLE_NAME = 'articles'
    AND COLUMN_NAME = 'scheduled_at'
);
SET @ddl := IF(
  @column_exists = 0,
  'ALTER TABLE articles ADD COLUMN scheduled_at TIMESTAMP NULL AFTER view_count',
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
    AND COLUMN_NAME = 'seo_title'
);
SET @ddl := IF(
  @column_exists = 0,
  'ALTER TABLE articles ADD COLUMN seo_title VARCHAR(160) DEFAULT '''' AFTER published_at',
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
    AND COLUMN_NAME = 'seo_description'
);
SET @ddl := IF(
  @column_exists = 0,
  'ALTER TABLE articles ADD COLUMN seo_description VARCHAR(255) DEFAULT '''' AFTER seo_title',
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
    AND COLUMN_NAME = 'seo_keywords'
);
SET @ddl := IF(
  @column_exists = 0,
  'ALTER TABLE articles ADD COLUMN seo_keywords VARCHAR(255) DEFAULT '''' AFTER seo_description',
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
    AND INDEX_NAME = 'idx_articles_status_schedule'
);
SET @ddl := IF(
  @index_exists = 0,
  'ALTER TABLE articles ADD INDEX idx_articles_status_schedule (status, scheduled_at)',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

CREATE TABLE IF NOT EXISTS article_versions (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    article_id BIGINT UNSIGNED NOT NULL,
    version_no INT NOT NULL,
    changed_by BIGINT UNSIGNED NOT NULL,
    author_id BIGINT UNSIGNED NOT NULL,
    category_id BIGINT UNSIGNED,
    title VARCHAR(160) NOT NULL,
    slug VARCHAR(180) NOT NULL,
    summary TEXT,
    content MEDIUMTEXT NOT NULL,
    cover_url VARCHAR(255) DEFAULT '',
    status VARCHAR(20) NOT NULL,
    view_count INT DEFAULT 0,
    scheduled_at TIMESTAMP NULL,
    published_at TIMESTAMP NULL,
    seo_title VARCHAR(160) DEFAULT '',
    seo_description VARCHAR(255) DEFAULT '',
    seo_keywords VARCHAR(255) DEFAULT '',
    tag_ids JSON,
    original_created_at TIMESTAMP NULL,
    original_updated_at TIMESTAMP NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY uniq_article_version (article_id, version_no),
    INDEX idx_article_versions_article (article_id, version_no)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS site_settings (
    setting_key VARCHAR(64) NOT NULL PRIMARY KEY,
    setting_value VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

INSERT INTO site_settings (setting_key, setting_value)
VALUES
  ('site_title', 'Notes of Ashen'),
  ('site_description', 'A personal blog written slowly by the lamp of ink.'),
  ('site_keywords', 'blog,notes,writing'),
  ('site_base_url', '')
ON DUPLICATE KEY UPDATE setting_value = setting_value;
