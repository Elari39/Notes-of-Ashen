USE notes_of_ashen;

SET @schema_name := DATABASE();

SET @column_exists := (
  SELECT COUNT(*)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = @schema_name
    AND TABLE_NAME = 'users'
    AND COLUMN_NAME = 'token_version'
);
SET @ddl := IF(
  @column_exists = 0,
  'ALTER TABLE users ADD COLUMN token_version BIGINT UNSIGNED NOT NULL DEFAULT 0 AFTER status',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;
