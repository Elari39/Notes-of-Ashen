
SET @schema_name := DATABASE();

SET @fulltext_exists := (
  SELECT COUNT(*)
  FROM (
    SELECT INDEX_NAME
    FROM information_schema.STATISTICS
    WHERE TABLE_SCHEMA = @schema_name
      AND TABLE_NAME = 'articles'
      AND INDEX_TYPE = 'FULLTEXT'
    GROUP BY INDEX_NAME
    HAVING COUNT(*) = 2
      AND SUM(CASE WHEN SEQ_IN_INDEX = 1 AND COLUMN_NAME = 'title' THEN 1 ELSE 0 END) = 1
      AND SUM(CASE WHEN SEQ_IN_INDEX = 2 AND COLUMN_NAME = 'content' THEN 1 ELSE 0 END) = 1
  ) AS article_fulltext_indexes
);

SET @ddl := IF(
  @fulltext_exists = 0,
  'ALTER TABLE articles ADD FULLTEXT KEY ft_articles_title_content (title, content)',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;
