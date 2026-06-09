USE notes_of_ashen;

SET @schema_name := DATABASE();

SET @column_exists := (
  SELECT COUNT(*)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = @schema_name
    AND TABLE_NAME = 'articles'
    AND COLUMN_NAME = 'like_count'
);
SET @ddl := IF(
  @column_exists = 0,
  'ALTER TABLE articles ADD COLUMN like_count INT DEFAULT 0 AFTER view_count',
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
    AND COLUMN_NAME = 'like_count'
);
SET @ddl := IF(
  @column_exists = 0,
  'ALTER TABLE article_versions ADD COLUMN like_count INT DEFAULT 0 AFTER view_count',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

CREATE TABLE IF NOT EXISTS resume_experiences (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    role VARCHAR(120) NOT NULL,
    organization VARCHAR(120) NOT NULL,
    location VARCHAR(120) DEFAULT '',
    start_date VARCHAR(32) DEFAULT '',
    end_date VARCHAR(32) DEFAULT '',
    description TEXT,
    highlights JSON,
    display_order INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_resume_experiences_order (display_order, id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS resume_educations (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    school VARCHAR(120) NOT NULL,
    degree VARCHAR(120) DEFAULT '',
    major VARCHAR(120) DEFAULT '',
    location VARCHAR(120) DEFAULT '',
    start_date VARCHAR(32) DEFAULT '',
    end_date VARCHAR(32) DEFAULT '',
    description TEXT,
    highlights JSON,
    display_order INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_resume_educations_order (display_order, id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS resume_skills (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    category VARCHAR(80) NOT NULL,
    name VARCHAR(80) NOT NULL,
    level INT NOT NULL DEFAULT 0,
    description VARCHAR(255) DEFAULT '',
    display_order INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_resume_skills_order (display_order, id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS projects (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    title VARCHAR(120) NOT NULL,
    summary VARCHAR(500) DEFAULT '',
    role VARCHAR(80) DEFAULT '',
    period VARCHAR(80) DEFAULT '',
    cover_url VARCHAR(255) DEFAULT '',
    demo_url VARCHAR(255) DEFAULT '',
    repo_url VARCHAR(255) DEFAULT '',
    content_markdown MEDIUMTEXT,
    featured TINYINT(1) NOT NULL DEFAULT 0,
    display_order INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_projects_order (featured, display_order, id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS project_tags (
    project_id BIGINT UNSIGNED NOT NULL,
    tag_id BIGINT UNSIGNED NOT NULL,
    PRIMARY KEY (project_id, tag_id),
    INDEX idx_project_tags_tag (tag_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS article_likes (
    article_id BIGINT UNSIGNED NOT NULL,
    visitor_hash CHAR(64) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (article_id, visitor_hash)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS traffic_geo_stats (
    stat_date DATE NOT NULL,
    country_code VARCHAR(32) NOT NULL,
    country_name VARCHAR(128) NOT NULL,
    region_name VARCHAR(128) NOT NULL,
    city_name VARCHAR(128) NOT NULL,
    pv BIGINT NOT NULL DEFAULT 0,
    uv BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (stat_date, country_code, region_name, city_name),
    INDEX idx_traffic_geo_stats_date (stat_date),
    INDEX idx_traffic_geo_stats_location (country_code, region_name, city_name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS traffic_geo_daily_visitors (
    stat_date DATE NOT NULL,
    visitor_hash CHAR(64) NOT NULL,
    country_code VARCHAR(32) NOT NULL,
    region_name VARCHAR(128) NOT NULL,
    city_name VARCHAR(128) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (stat_date, visitor_hash, country_code, region_name, city_name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
