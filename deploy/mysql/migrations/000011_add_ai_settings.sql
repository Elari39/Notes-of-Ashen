
INSERT INTO site_settings (setting_key, setting_value)
VALUES ('ai_enabled', 'false')
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
VALUES ('ai_stream_timeout_seconds', '300')
ON DUPLICATE KEY UPDATE setting_value = setting_value;

INSERT INTO site_settings (setting_key, setting_value)
VALUES ('ai_non_stream_timeout_seconds', '600')
ON DUPLICATE KEY UPDATE setting_value = setting_value;

INSERT INTO site_settings (setting_key, setting_value)
VALUES ('ai_settings_configured', 'false')
ON DUPLICATE KEY UPDATE setting_value = setting_value;
