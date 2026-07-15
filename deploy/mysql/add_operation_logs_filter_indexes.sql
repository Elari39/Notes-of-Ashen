USE notes_of_ashen;

SET @schema_name := DATABASE();

-- 支撑管理员操作日志按事件/IP 精确筛选并继续按时间范围收敛结果。
SET @event_created_exists := (
  SELECT COUNT(*)
  FROM information_schema.STATISTICS
  WHERE TABLE_SCHEMA = @schema_name
    AND TABLE_NAME = 'operation_logs'
    AND INDEX_NAME = 'idx_operation_logs_event_created'
);
SET @event_created_ddl := IF(
  @event_created_exists = 0,
  'ALTER TABLE operation_logs ADD INDEX idx_operation_logs_event_created (event_type, created_at)',
  'SELECT 1'
);
PREPARE stmt FROM @event_created_ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @ip_created_exists := (
  SELECT COUNT(*)
  FROM information_schema.STATISTICS
  WHERE TABLE_SCHEMA = @schema_name
    AND TABLE_NAME = 'operation_logs'
    AND INDEX_NAME = 'idx_operation_logs_ip_created'
);
SET @ip_created_ddl := IF(
  @ip_created_exists = 0,
  'ALTER TABLE operation_logs ADD INDEX idx_operation_logs_ip_created (ip, created_at)',
  'SELECT 1'
);
PREPARE stmt FROM @ip_created_ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;
