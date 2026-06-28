USE notes_of_ashen;

SET @schema_name := DATABASE();

-- operation_logs 是只增的审计表，长期运行后行数增长。
-- created_at 索引支撑按时间范围查询/归档；user_id 索引支撑按用户筛选操作历史。
SET @index_created_exists := (
  SELECT COUNT(*)
  FROM information_schema.STATISTICS
  WHERE TABLE_SCHEMA = @schema_name
    AND TABLE_NAME = 'operation_logs'
    AND INDEX_NAME = 'idx_operation_logs_created'
);
SET @ddl_created := IF(
  @index_created_exists = 0,
  'ALTER TABLE operation_logs ADD INDEX idx_operation_logs_created (created_at)',
  'SELECT 1'
);
PREPARE stmt FROM @ddl_created;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @index_user_exists := (
  SELECT COUNT(*)
  FROM information_schema.STATISTICS
  WHERE TABLE_SCHEMA = @schema_name
    AND TABLE_NAME = 'operation_logs'
    AND INDEX_NAME = 'idx_operation_logs_user'
);
SET @ddl_user := IF(
  @index_user_exists = 0,
  'ALTER TABLE operation_logs ADD INDEX idx_operation_logs_user (user_id)',
  'SELECT 1'
);
PREPARE stmt FROM @ddl_user;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;
