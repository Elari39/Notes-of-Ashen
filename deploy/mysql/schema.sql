CREATE DATABASE IF NOT EXISTS notes_of_ashen CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci;

USE notes_of_ashen;

-- Users
CREATE TABLE IF NOT EXISTS users (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    account VARCHAR(64) NOT NULL UNIQUE,
    password_hash VARCHAR(128) NOT NULL,
    email VARCHAR(128) NOT NULL UNIQUE,
    avatar_url VARCHAR(255) DEFAULT '',
    nickname VARCHAR(64) DEFAULT '',
    role VARCHAR(20) DEFAULT 'user',
    status VARCHAR(20) DEFAULT 'active',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_users_role_status_id (role, status, id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE utf8mb4_0900_ai_ci;

-- 站点设置
CREATE TABLE IF NOT EXISTS site_settings (
    setting_key VARCHAR(64) NOT NULL PRIMARY KEY,
    setting_value MEDIUMTEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE utf8mb4_0900_ai_ci;

INSERT INTO site_settings (setting_key, setting_value)
VALUES ('registration_enabled', 'true')
ON DUPLICATE KEY UPDATE setting_value = setting_value;

INSERT INTO site_settings (setting_key, setting_value)
VALUES ('home_article_layout', 'standard')
ON DUPLICATE KEY UPDATE setting_value = setting_value;

INSERT INTO site_settings (setting_key, setting_value)
VALUES ('site_title', 'Notes of Ashen')
ON DUPLICATE KEY UPDATE setting_value = setting_value;

INSERT INTO site_settings (setting_key, setting_value)
VALUES ('site_description', 'A personal blog written slowly by the lamp of ink.')
ON DUPLICATE KEY UPDATE setting_value = setting_value;

INSERT INTO site_settings (setting_key, setting_value)
VALUES ('site_keywords', 'blog,notes,writing')
ON DUPLICATE KEY UPDATE setting_value = setting_value;

INSERT INTO site_settings (setting_key, setting_value)
VALUES ('site_base_url', '')
ON DUPLICATE KEY UPDATE setting_value = setting_value;

INSERT INTO site_settings (setting_key, setting_value)
VALUES ('resume_page_enabled', 'false')
ON DUPLICATE KEY UPDATE setting_value = setting_value;

INSERT INTO site_settings (setting_key, setting_value)
VALUES ('resume_nav_hidden', 'true')
ON DUPLICATE KEY UPDATE setting_value = setting_value;

INSERT INTO site_settings (setting_key, setting_value)
VALUES ('projects_page_enabled', 'false')
ON DUPLICATE KEY UPDATE setting_value = setting_value;

INSERT INTO site_settings (setting_key, setting_value)
VALUES ('projects_nav_hidden', 'true')
ON DUPLICATE KEY UPDATE setting_value = setting_value;

INSERT INTO site_settings (setting_key, setting_value)
VALUES ('resume_title', '简介')
ON DUPLICATE KEY UPDATE setting_value = setting_value;

INSERT INTO site_settings (setting_key, setting_value)
VALUES ('resume_subtitle', '')
ON DUPLICATE KEY UPDATE setting_value = setting_value;

INSERT INTO site_settings (setting_key, setting_value)
VALUES ('resume_content_markdown', '')
ON DUPLICATE KEY UPDATE setting_value = setting_value;

INSERT INTO site_settings (setting_key, setting_value)
VALUES ('projects_title', '项目')
ON DUPLICATE KEY UPDATE setting_value = setting_value;

INSERT INTO site_settings (setting_key, setting_value)
VALUES ('projects_subtitle', '')
ON DUPLICATE KEY UPDATE setting_value = setting_value;

INSERT INTO site_settings (setting_key, setting_value)
VALUES ('projects_items_json', '[]')
ON DUPLICATE KEY UPDATE setting_value = setting_value;

INSERT INTO site_settings (setting_key, setting_value)
VALUES ('ai_enabled', 'false')
ON DUPLICATE KEY UPDATE setting_value = setting_value;

INSERT INTO site_settings (setting_key, setting_value)
VALUES ('ai_api_format', 'openai')
ON DUPLICATE KEY UPDATE setting_value = setting_value;

INSERT INTO site_settings (setting_key, setting_value)
VALUES ('ai_base_url', '')
ON DUPLICATE KEY UPDATE setting_value = setting_value;

INSERT INTO site_settings (setting_key, setting_value)
VALUES ('ai_api_key_cipher', '')
ON DUPLICATE KEY UPDATE setting_value = setting_value;

INSERT INTO site_settings (setting_key, setting_value)
VALUES ('ai_model', '')
ON DUPLICATE KEY UPDATE setting_value = setting_value;

INSERT INTO site_settings (setting_key, setting_value)
VALUES ('ai_first_byte_timeout_seconds', '60')
ON DUPLICATE KEY UPDATE setting_value = setting_value;

INSERT INTO site_settings (setting_key, setting_value)
VALUES ('ai_non_stream_timeout_seconds', '600')
ON DUPLICATE KEY UPDATE setting_value = setting_value;

-- Structured resume
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
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE utf8mb4_0900_ai_ci;

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
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE utf8mb4_0900_ai_ci;

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
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE utf8mb4_0900_ai_ci;

-- Categories
CREATE TABLE IF NOT EXISTS categories (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(64) NOT NULL UNIQUE,
    slug VARCHAR(96) NOT NULL UNIQUE,
    description TEXT,
    created_by BIGINT UNSIGNED NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE utf8mb4_0900_ai_ci;

-- Tags
CREATE TABLE IF NOT EXISTS tags (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(64) NOT NULL UNIQUE,
    slug VARCHAR(96) NOT NULL UNIQUE,
    description TEXT,
    created_by BIGINT UNSIGNED NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE utf8mb4_0900_ai_ci;

-- Portfolio projects
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
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS project_tags (
    project_id BIGINT UNSIGNED NOT NULL,
    tag_id BIGINT UNSIGNED NOT NULL,
    PRIMARY KEY (project_id, tag_id),
    INDEX idx_project_tags_tag (tag_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE utf8mb4_0900_ai_ci;

-- Articles
CREATE TABLE IF NOT EXISTS articles (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    author_id BIGINT UNSIGNED NOT NULL,
    category_id BIGINT UNSIGNED,
    title VARCHAR(160) NOT NULL,
    slug VARCHAR(180) NOT NULL UNIQUE,
    summary TEXT,
    content MEDIUMTEXT NOT NULL,
    cover_url VARCHAR(255) DEFAULT '',
    status VARCHAR(20) DEFAULT 'draft',
    view_count INT DEFAULT 0,
    like_count INT DEFAULT 0,
    scheduled_at TIMESTAMP NULL,
    published_at TIMESTAMP NULL,
    is_pinned TINYINT(1) NOT NULL DEFAULT 0,
    display_priority INT NOT NULL DEFAULT 0,
    seo_title VARCHAR(160) DEFAULT '',
    seo_description VARCHAR(255) DEFAULT '',
    seo_keywords VARCHAR(255) DEFAULT '',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_articles_status_schedule (status, scheduled_at),
    INDEX idx_articles_display_order (status, scheduled_at, is_pinned, display_priority, published_at, id),
    INDEX idx_articles_category (category_id, status, scheduled_at),
    INDEX idx_articles_author (author_id, status),
    FULLTEXT KEY ft_articles_title_content (title, content)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE utf8mb4_0900_ai_ci;

-- Article versions
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
    like_count INT DEFAULT 0,
    scheduled_at TIMESTAMP NULL,
    published_at TIMESTAMP NULL,
    is_pinned TINYINT(1) NOT NULL DEFAULT 0,
    display_priority INT NOT NULL DEFAULT 0,
    seo_title VARCHAR(160) DEFAULT '',
    seo_description VARCHAR(255) DEFAULT '',
    seo_keywords VARCHAR(255) DEFAULT '',
    tag_ids JSON,
    original_created_at TIMESTAMP NULL,
    original_updated_at TIMESTAMP NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY uniq_article_version (article_id, version_no),
    INDEX idx_article_versions_article (article_id, version_no)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE utf8mb4_0900_ai_ci;

-- Article tags
CREATE TABLE IF NOT EXISTS article_tags (
    article_id BIGINT UNSIGNED NOT NULL,
    tag_id BIGINT UNSIGNED NOT NULL,
    PRIMARY KEY (article_id, tag_id),
    INDEX idx_article_tags_tag (tag_id, article_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE utf8mb4_0900_ai_ci;

-- Article likes
CREATE TABLE IF NOT EXISTS article_likes (
    article_id BIGINT UNSIGNED NOT NULL,
    visitor_hash CHAR(64) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (article_id, visitor_hash)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE utf8mb4_0900_ai_ci;

-- Refresh tokens
CREATE TABLE IF NOT EXISTS refresh_tokens (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    user_id BIGINT UNSIGNED NOT NULL,
    token_hash VARCHAR(128) NOT NULL UNIQUE,
    expires_at TIMESTAMP NOT NULL,
    revoked_at TIMESTAMP NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX (user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE utf8mb4_0900_ai_ci;

-- Operation logs
CREATE TABLE IF NOT EXISTS operation_logs (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    user_id BIGINT UNSIGNED,
    event_type VARCHAR(64) NOT NULL,
    resource_type VARCHAR(64) NOT NULL,
    resource_id BIGINT UNSIGNED,
    metadata JSON,
    ip VARCHAR(45),
    user_agent TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_operation_logs_created (created_at),
    INDEX idx_operation_logs_user (user_id),
    INDEX idx_operation_logs_event_created (event_type, created_at),
    INDEX idx_operation_logs_ip_created (ip, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE utf8mb4_0900_ai_ci;

-- Traffic daily stats
CREATE TABLE IF NOT EXISTS traffic_daily_stats (
    stat_date DATE NOT NULL PRIMARY KEY,
    pv BIGINT NOT NULL DEFAULT 0,
    uv BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE utf8mb4_0900_ai_ci;

-- Traffic daily visitors
CREATE TABLE IF NOT EXISTS traffic_daily_visitors (
    stat_date DATE NOT NULL,
    visitor_hash CHAR(64) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (stat_date, visitor_hash)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE utf8mb4_0900_ai_ci;

-- Traffic referer stats
CREATE TABLE IF NOT EXISTS traffic_referer_stats (
    stat_date DATE NOT NULL,
    article_id BIGINT UNSIGNED NOT NULL DEFAULT 0,
    source_type VARCHAR(32) NOT NULL,
    source_name VARCHAR(128) NOT NULL,
    pv BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (stat_date, article_id, source_type, source_name),
    INDEX idx_traffic_referer_stats_date (stat_date),
    INDEX idx_traffic_referer_stats_article (article_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE utf8mb4_0900_ai_ci;
