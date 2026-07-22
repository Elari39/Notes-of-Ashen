INSERT INTO site_settings (setting_key, setting_value)
VALUES ('ai_api_format', 'openai')
ON DUPLICATE KEY UPDATE setting_value = setting_value;
