
-- 前置依赖：add_site_settings.sql（建表）。
-- 注意：site_settings.setting_value 的 MEDIUMTEXT 变更由 alter_site_settings_value_text.sql 负责，
-- 本脚本仅插入公开页面内容相关默认设置，避免与该脚本重复 ALTER。

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
