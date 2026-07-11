USE notes_of_ashen;

-- 管理员状态更新会按 role、status 筛选并按 id 加锁。
-- 通过 information_schema 检查使脚本可重复执行，兼容已经升级过的部署。
SET @index_exists = (
    SELECT COUNT(*)
    FROM information_schema.statistics
    WHERE table_schema = DATABASE()
      AND table_name = 'users'
      AND index_name = 'idx_users_role_status_id'
);

SET @sql = IF(
    @index_exists = 0,
    'ALTER TABLE users ADD INDEX idx_users_role_status_id (role, status, id)',
    'SELECT 1'
);

PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;
