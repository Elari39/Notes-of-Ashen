USE notes_of_ashen;

-- Site settings for system-level switches such as account registration.
CREATE TABLE IF NOT EXISTS site_settings (
    setting_key VARCHAR(64) NOT NULL PRIMARY KEY,
    setting_value VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

INSERT INTO site_settings (setting_key, setting_value)
VALUES ('registration_enabled', 'true')
ON DUPLICATE KEY UPDATE setting_value = setting_value;

INSERT INTO site_settings (setting_key, setting_value)
VALUES ('home_article_layout', 'standard')
ON DUPLICATE KEY UPDATE setting_value = setting_value;
