USE notes_of_ashen;

CREATE TABLE IF NOT EXISTS media_assets (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    storage_key VARCHAR(96) NOT NULL UNIQUE,
    original_name VARCHAR(255) NOT NULL,
    mime_type VARCHAR(64) NOT NULL,
    size_bytes BIGINT UNSIGNED NOT NULL,
    width INT UNSIGNED NOT NULL DEFAULT 0,
    height INT UNSIGNED NOT NULL DEFAULT 0,
    alt_text VARCHAR(255) NOT NULL DEFAULT '',
    sha256 CHAR(64) NOT NULL UNIQUE,
    created_by BIGINT UNSIGNED NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_media_assets_created (created_at, id),
    INDEX idx_media_assets_creator (created_by, id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS traffic_content_daily_stats (
    stat_date DATE NOT NULL,
    route_type VARCHAR(32) NOT NULL,
    article_id BIGINT UNSIGNED NOT NULL DEFAULT 0,
    path VARCHAR(255) NOT NULL,
    pv BIGINT NOT NULL DEFAULT 0,
    uv BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (stat_date, route_type, article_id, path),
    INDEX idx_content_stats_date_article (stat_date, article_id),
    INDEX idx_content_stats_date_pv (stat_date, pv)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS traffic_content_daily_visitors (
    stat_date DATE NOT NULL,
    route_type VARCHAR(32) NOT NULL,
    article_id BIGINT UNSIGNED NOT NULL DEFAULT 0,
    path VARCHAR(255) NOT NULL,
    visitor_hash CHAR(64) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (stat_date, route_type, article_id, path, visitor_hash),
    INDEX idx_content_visitors_created (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE utf8mb4_0900_ai_ci;
