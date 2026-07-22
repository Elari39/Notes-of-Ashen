-- 为核心关系建立数据库级完整性策略。
--
-- 可安全丢弃的关联孤儿先清理；文章/分类/标签/媒体的必需创建者关系不静默删除，
-- 若仍存在非法数据，后续 ALTER TABLE 会失败并阻止本次迁移，要求运维先修复数据。

DELETE article_tags
FROM article_tags
LEFT JOIN articles ON articles.id = article_tags.article_id
LEFT JOIN tags ON tags.id = article_tags.tag_id
WHERE articles.id IS NULL OR tags.id IS NULL;

DELETE project_tags
FROM project_tags
LEFT JOIN projects ON projects.id = project_tags.project_id
LEFT JOIN tags ON tags.id = project_tags.tag_id
WHERE projects.id IS NULL OR tags.id IS NULL;

DELETE article_likes
FROM article_likes
LEFT JOIN articles ON articles.id = article_likes.article_id
WHERE articles.id IS NULL;

DELETE article_versions
FROM article_versions
LEFT JOIN articles ON articles.id = article_versions.article_id
WHERE articles.id IS NULL;

DELETE refresh_tokens
FROM refresh_tokens
LEFT JOIN users ON users.id = refresh_tokens.user_id
WHERE users.id IS NULL;

UPDATE operation_logs
LEFT JOIN users ON users.id = operation_logs.user_id
SET operation_logs.user_id = NULL
WHERE operation_logs.user_id IS NOT NULL AND users.id IS NULL;

UPDATE articles
LEFT JOIN categories ON categories.id = articles.category_id
SET articles.category_id = NULL
WHERE articles.category_id IS NOT NULL AND categories.id IS NULL;

UPDATE article_versions
LEFT JOIN categories ON categories.id = article_versions.category_id
SET article_versions.category_id = NULL
WHERE article_versions.category_id IS NOT NULL AND categories.id IS NULL;

-- 必需关系不自动删数据。用受约束的临时表在所有 DDL 前统一阻断非法状态，
-- 避免某个核心表存在孤儿时只完成前半段外键、导致重试无法恢复。
CREATE TEMPORARY TABLE noa_relationship_integrity_preflight (
    violation_count BIGINT NOT NULL,
    CHECK (violation_count = 0)
) ENGINE = MEMORY;

INSERT INTO noa_relationship_integrity_preflight (violation_count)
SELECT
    (SELECT COUNT(*) FROM categories c LEFT JOIN users u ON u.id = c.created_by WHERE u.id IS NULL)
    + (SELECT COUNT(*) FROM tags t LEFT JOIN users u ON u.id = t.created_by WHERE u.id IS NULL)
    + (SELECT COUNT(*) FROM articles a LEFT JOIN users u ON u.id = a.author_id WHERE u.id IS NULL)
    + (SELECT COUNT(*) FROM article_versions v LEFT JOIN users u ON u.id = v.changed_by WHERE u.id IS NULL)
    + (SELECT COUNT(*) FROM article_versions v LEFT JOIN users u ON u.id = v.author_id WHERE u.id IS NULL)
    + (SELECT COUNT(*) FROM media_assets m LEFT JOIN users u ON u.id = m.created_by WHERE u.id IS NULL);

DROP TEMPORARY TABLE noa_relationship_integrity_preflight;

ALTER TABLE categories
  ADD CONSTRAINT fk_categories_created_by
    FOREIGN KEY (created_by) REFERENCES users (id)
    ON DELETE RESTRICT ON UPDATE CASCADE;

ALTER TABLE tags
  ADD CONSTRAINT fk_tags_created_by
    FOREIGN KEY (created_by) REFERENCES users (id)
    ON DELETE RESTRICT ON UPDATE CASCADE;

ALTER TABLE articles
  ADD CONSTRAINT fk_articles_author
    FOREIGN KEY (author_id) REFERENCES users (id)
    ON DELETE RESTRICT ON UPDATE CASCADE,
  ADD CONSTRAINT fk_articles_category
    FOREIGN KEY (category_id) REFERENCES categories (id)
    ON DELETE SET NULL ON UPDATE CASCADE;

ALTER TABLE article_tags
  ADD CONSTRAINT fk_article_tags_article
    FOREIGN KEY (article_id) REFERENCES articles (id)
    ON DELETE CASCADE ON UPDATE CASCADE,
  ADD CONSTRAINT fk_article_tags_tag
    FOREIGN KEY (tag_id) REFERENCES tags (id)
    ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE refresh_tokens
  ADD CONSTRAINT fk_refresh_tokens_user
    FOREIGN KEY (user_id) REFERENCES users (id)
    ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE operation_logs
  ADD CONSTRAINT fk_operation_logs_user
    FOREIGN KEY (user_id) REFERENCES users (id)
    ON DELETE SET NULL ON UPDATE CASCADE;

ALTER TABLE article_versions
  ADD CONSTRAINT fk_article_versions_article
    FOREIGN KEY (article_id) REFERENCES articles (id)
    ON DELETE CASCADE ON UPDATE CASCADE,
  ADD CONSTRAINT fk_article_versions_changed_by
    FOREIGN KEY (changed_by) REFERENCES users (id)
    ON DELETE RESTRICT ON UPDATE CASCADE,
  ADD CONSTRAINT fk_article_versions_author
    FOREIGN KEY (author_id) REFERENCES users (id)
    ON DELETE RESTRICT ON UPDATE CASCADE,
  ADD CONSTRAINT fk_article_versions_category
    FOREIGN KEY (category_id) REFERENCES categories (id)
    ON DELETE SET NULL ON UPDATE CASCADE;

ALTER TABLE project_tags
  ADD CONSTRAINT fk_project_tags_project
    FOREIGN KEY (project_id) REFERENCES projects (id)
    ON DELETE CASCADE ON UPDATE CASCADE,
  ADD CONSTRAINT fk_project_tags_tag
    FOREIGN KEY (tag_id) REFERENCES tags (id)
    ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE article_likes
  ADD CONSTRAINT fk_article_likes_article
    FOREIGN KEY (article_id) REFERENCES articles (id)
    ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE media_assets
  ADD CONSTRAINT fk_media_assets_creator
    FOREIGN KEY (created_by) REFERENCES users (id)
    ON DELETE RESTRICT ON UPDATE CASCADE;
