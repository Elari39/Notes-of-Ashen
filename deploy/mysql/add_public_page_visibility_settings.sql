USE notes_of_ashen;

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
