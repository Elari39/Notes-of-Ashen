USE notes_of_ashen;

INSERT INTO site_settings (setting_key, setting_value)
VALUES ('home_cta_hidden', 'false')
ON DUPLICATE KEY UPDATE setting_value = setting_value;
