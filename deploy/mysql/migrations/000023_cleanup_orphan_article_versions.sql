
-- 清理文章已删除后遗留的历史版本记录；可重复执行。
DELETE article_versions
FROM article_versions
LEFT JOIN articles ON articles.id = article_versions.article_id
WHERE articles.id IS NULL;
