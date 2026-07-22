
CREATE TABLE IF NOT EXISTS traffic_daily_stats (
    stat_date DATE NOT NULL PRIMARY KEY,
    pv BIGINT NOT NULL DEFAULT 0,
    uv BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS traffic_daily_visitors (
    stat_date DATE NOT NULL,
    visitor_hash CHAR(64) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (stat_date, visitor_hash)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

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
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
