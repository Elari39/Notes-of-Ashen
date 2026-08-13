package model

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
)

const (
	ArticleStatusDraft     = "draft"
	ArticleStatusPublished = "published"
	ArticleStatusArchived  = "archived"
	ArticleStatusScheduled = "scheduled"

	articleSelectFields        = "id, author_id, category_id, title, slug, summary, content, cover_url, status, view_count, like_count, scheduled_at, published_at, is_pinned, display_priority, seo_title, seo_description, seo_keywords, created_at, updated_at"
	articleDisplayOrder        = "is_pinned DESC, display_priority DESC, COALESCE(published_at, created_at) DESC, id DESC"
	articleDisplayOrderAsc     = "is_pinned ASC, display_priority ASC, COALESCE(published_at, created_at) ASC, id ASC"
	articleTimeOrder           = "COALESCE(published_at, created_at) DESC, id DESC"
	articleVersionSelectFields = "id, article_id, version_no, changed_by, author_id, category_id, title, slug, summary, content, cover_url, status, view_count, like_count, scheduled_at, published_at, is_pinned, display_priority, seo_title, seo_description, seo_keywords, COALESCE(CAST(tag_ids AS CHAR), '[]'), original_created_at, original_updated_at, created_at"
)

type Article struct {
	ID              uint64
	AuthorID        uint64
	CategoryID      uint64
	Title           string
	Slug            string
	Summary         string
	Content         string
	CoverURL        string
	Status          string
	ViewCount       uint64
	LikeCount       uint64
	ScheduledAt     *time.Time
	PublishedAt     *time.Time
	IsPinned        bool
	DisplayPriority int
	SEOTitle        string
	SEODescription  string
	SEOKeywords     string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// PublicArticleEntry 是 RSS 与 Sitemap 使用的轻量公开文章条目，避免读取正文等大字段。
type PublicArticleEntry struct {
	ID          uint64
	Title       string
	Summary     string
	PublishedAt *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type ArticleCreate struct {
	AuthorID        uint64
	CategoryID      uint64
	Title           string
	Slug            string
	Summary         string
	Content         string
	CoverURL        string
	Status          string
	ScheduledAt     *time.Time
	IsPinned        bool
	DisplayPriority int
	SEOTitle        string
	SEODescription  string
	SEOKeywords     string
	TagIDs          []uint64
}

type MarkdownArticleImport struct {
	Article  ArticleCreate
	Category *TaxonomyCreate
	Tags     []TaxonomyCreate
}

type ArticleUpdate struct {
	CategoryID      uint64
	Title           string
	Slug            string
	Summary         string
	Content         string
	CoverURL        string
	Status          string
	ScheduledAt     *time.Time
	IsPinned        bool
	DisplayPriority int
	SEOTitle        string
	SEODescription  string
	SEOKeywords     string
	TagIDs          []uint64
}

type ArticleFilter struct {
	UserID     uint64
	Role       string
	Status     string
	Query      string
	CategoryID uint64
	TagID      uint64
	Page       int
	Size       int
}

type ArticleVersion struct {
	ID                uint64
	ArticleID         uint64
	VersionNo         int
	ChangedBy         uint64
	AuthorID          uint64
	CategoryID        uint64
	Title             string
	Slug              string
	Summary           string
	Content           string
	CoverURL          string
	Status            string
	ViewCount         uint64
	LikeCount         uint64
	ScheduledAt       *time.Time
	PublishedAt       *time.Time
	IsPinned          bool
	DisplayPriority   int
	SEOTitle          string
	SEODescription    string
	SEOKeywords       string
	TagIDs            []uint64
	OriginalCreatedAt *time.Time
	OriginalUpdatedAt *time.Time
	CreatedAt         time.Time
}

func (s *Store) CreateArticle(ctx context.Context, in ArticleCreate) (uint64, error) {
	var id uint64
	err := WithTx(ctx, s.db, func(tx *sql.Tx) error {
		var err error
		id, err = createArticleTx(ctx, tx, in)
		if err != nil {
			return err
		}
		return enqueueArticleRAGSyncTx(ctx, tx, id, in.Status, in.ScheduledAt)
	})
	return id, err
}

// CreateMarkdownArticle 将 Markdown 导入所需的 taxonomy ensure、文章和标签关系
// 放入同一事务，后续任一写入失败都会整体回滚。
func (s *Store) CreateMarkdownArticle(ctx context.Context, in MarkdownArticleImport) (uint64, error) {
	var id uint64
	err := WithTx(ctx, s.db, func(tx *sql.Tx) error {
		article := in.Article
		if in.Category != nil {
			categoryID, err := ensureImportCategoryTx(ctx, tx, *in.Category)
			if err != nil {
				return err
			}
			article.CategoryID = categoryID
		}
		article.TagIDs = make([]uint64, 0, len(in.Tags))
		for _, tag := range in.Tags {
			tagID, err := ensureImportTagTx(ctx, tx, tag)
			if err != nil {
				return err
			}
			article.TagIDs = append(article.TagIDs, tagID)
		}
		var err error
		id, err = createArticleTx(ctx, tx, article)
		if err != nil {
			return err
		}
		return enqueueArticleRAGSyncTx(ctx, tx, id, article.Status, article.ScheduledAt)
	})
	return id, err
}

func createArticleTx(ctx context.Context, tx *sql.Tx, in ArticleCreate) (uint64, error) {
	publishedAt := publishedAtForCreate(in.Status, in.ScheduledAt)
	res, err := tx.ExecContext(ctx, `
INSERT INTO articles (author_id, category_id, title, slug, summary, content, cover_url, status, scheduled_at, published_at, is_pinned, display_priority, seo_title, seo_description, seo_keywords)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		in.AuthorID, nullableUint64(in.CategoryID), in.Title, in.Slug, in.Summary, in.Content, in.CoverURL, in.Status,
		nullableTime(in.ScheduledAt), nullableTime(publishedAt), in.IsPinned, in.DisplayPriority, in.SEOTitle, in.SEODescription, in.SEOKeywords)
	if err != nil {
		return 0, err
	}
	insertID, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	id := uint64(insertID)
	if err := replaceArticleTags(ctx, tx, id, in.TagIDs); err != nil {
		return 0, err
	}
	return id, nil
}

func ensureImportCategoryTx(ctx context.Context, tx *sql.Tx, in TaxonomyCreate) (uint64, error) {
	item, err := scanCategory(tx.QueryRowContext(ctx, `
SELECT id, name, slug, description, created_by, created_at, updated_at
FROM categories WHERE name = ? OR slug = ? ORDER BY CASE WHEN slug = ? THEN 0 ELSE 1 END LIMIT 1`, in.Name, in.Slug, in.Slug))
	if err == nil {
		return item.ID, nil
	}
	if err != ErrNotFound {
		return 0, err
	}
	res, err := tx.ExecContext(ctx, `INSERT INTO categories (name, slug, description, created_by) VALUES (?, ?, ?, ?)`, in.Name, in.Slug, in.Description, in.CreatedBy)
	if err != nil {
		if existing, findErr := scanCategory(tx.QueryRowContext(ctx, `
SELECT id, name, slug, description, created_by, created_at, updated_at
FROM categories WHERE name = ? OR slug = ? ORDER BY CASE WHEN slug = ? THEN 0 ELSE 1 END LIMIT 1`, in.Name, in.Slug, in.Slug)); findErr == nil {
			return existing.ID, nil
		}
		return 0, err
	}
	id, err := res.LastInsertId()
	return uint64(id), err
}

func ensureImportTagTx(ctx context.Context, tx *sql.Tx, in TaxonomyCreate) (uint64, error) {
	item, err := scanTag(tx.QueryRowContext(ctx, `
SELECT id, name, slug, description, created_by, created_at, updated_at
FROM tags WHERE name = ? OR slug = ? ORDER BY CASE WHEN slug = ? THEN 0 ELSE 1 END LIMIT 1`, in.Name, in.Slug, in.Slug))
	if err == nil {
		return item.ID, nil
	}
	if err != ErrNotFound {
		return 0, err
	}
	res, err := tx.ExecContext(ctx, `INSERT INTO tags (name, slug, description, created_by) VALUES (?, ?, ?, ?)`, in.Name, in.Slug, in.Description, in.CreatedBy)
	if err != nil {
		if existing, findErr := scanTag(tx.QueryRowContext(ctx, `
SELECT id, name, slug, description, created_by, created_at, updated_at
FROM tags WHERE name = ? OR slug = ? ORDER BY CASE WHEN slug = ? THEN 0 ELSE 1 END LIMIT 1`, in.Name, in.Slug, in.Slug)); findErr == nil {
			return existing.ID, nil
		}
		return 0, err
	}
	id, err := res.LastInsertId()
	return uint64(id), err
}

func (s *Store) UpdateArticle(ctx context.Context, id uint64, in ArticleUpdate, changedBy uint64) error {
	return WithTx(ctx, s.db, func(tx *sql.Tx) error {
		if err := createArticleVersion(ctx, tx, id, changedBy); err != nil {
			return err
		}

		var currentStatus string
		var currentPublished sql.NullTime
		if err := tx.QueryRowContext(ctx, "SELECT status, published_at FROM articles WHERE id = ?", id).Scan(&currentStatus, &currentPublished); err != nil {
			return scanErr(err)
		}
		publishedAt := publishedAtForUpdate(currentStatus, timeFromNull(currentPublished), in.Status, in.ScheduledAt)

		res, err := tx.ExecContext(ctx, `
UPDATE articles
SET category_id = ?, title = ?, slug = ?, summary = ?, content = ?, cover_url = ?, status = ?, scheduled_at = ?, published_at = ?, is_pinned = ?, display_priority = ?, seo_title = ?, seo_description = ?, seo_keywords = ?
WHERE id = ?`,
			nullableUint64(in.CategoryID), in.Title, in.Slug, in.Summary, in.Content, in.CoverURL, in.Status,
			nullableTime(in.ScheduledAt), nullableTime(publishedAt), in.IsPinned, in.DisplayPriority, in.SEOTitle, in.SEODescription, in.SEOKeywords, id)
		if err != nil {
			return err
		}
		if err := requireUpdateAffected(ctx, res, func(ctx context.Context) error {
			return articleExistsTx(ctx, tx, id)
		}); err != nil {
			return err
		}
		if err := replaceArticleTags(ctx, tx, id, in.TagIDs); err != nil {
			return err
		}
		return enqueueArticleRAGSyncTx(ctx, tx, id, in.Status, in.ScheduledAt)
	})
}

func (s *Store) DeleteArticle(ctx context.Context, id uint64) error {
	return WithTx(ctx, s.db, func(tx *sql.Tx) error {
		// 删除任务先写入同一事务；rag_sync_jobs 不存在文章外键，因此提交后仍可
		// 清理已被删除文章的向量和历史回答。
		if err := enqueueRAGSyncJobTx(ctx, tx, id, RAGSyncOperationDelete, time.Now().UTC()); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, "DELETE FROM article_tags WHERE article_id = ?", id); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, "DELETE FROM article_likes WHERE article_id = ?", id); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, "DELETE FROM article_versions WHERE article_id = ?", id); err != nil {
			return err
		}
		res, err := tx.ExecContext(ctx, "DELETE FROM articles WHERE id = ?", id)
		if err != nil {
			return err
		}
		return requireAffected(res)
	})
}

func (s *Store) UpdateArticleStatus(ctx context.Context, id uint64, status string, changedBy uint64) error {
	return WithTx(ctx, s.db, func(tx *sql.Tx) error {
		if err := createArticleVersion(ctx, tx, id, changedBy); err != nil {
			return err
		}
		var currentStatus string
		var currentPublished sql.NullTime
		var scheduledAt sql.NullTime
		if err := tx.QueryRowContext(ctx, "SELECT status, published_at, scheduled_at FROM articles WHERE id = ?", id).Scan(&currentStatus, &currentPublished, &scheduledAt); err != nil {
			return scanErr(err)
		}
		// 状态接口语义为「立即切换状态」，不沿用 scheduled_at：转 published 表示立即发布，
		// published_at 取原首发时间（若有）否则 now；清空 scheduled_at 避免未来时间导致
		// IsArticlePubliclyVisible 判定不可见。需要定时发布应走 UpdateArticle（PUT）。
		publishedAt := publishedAtForUpdate(currentStatus, timeFromNull(currentPublished), status, nil)
		res, err := tx.ExecContext(ctx, `
UPDATE articles
SET status = ?, published_at = ?, scheduled_at = NULL
WHERE id = ?`, status, nullableTime(publishedAt), id)
		if err != nil {
			return err
		}
		if err := requireUpdateAffected(ctx, res, func(ctx context.Context) error {
			return articleExistsTx(ctx, tx, id)
		}); err != nil {
			return err
		}
		return enqueueArticleRAGSyncTx(ctx, tx, id, status, nil)
	})
}

func (s *Store) RestoreArticleVersion(ctx context.Context, articleID uint64, versionNo int, changedBy uint64) error {
	return WithTx(ctx, s.db, func(tx *sql.Tx) error {
		if err := createArticleVersion(ctx, tx, articleID, changedBy); err != nil {
			return err
		}
		version, err := findArticleVersion(ctx, tx, articleID, versionNo)
		if err != nil {
			return err
		}
		res, err := tx.ExecContext(ctx, `
UPDATE articles
SET category_id = ?, title = ?, slug = ?, summary = ?, content = ?, cover_url = ?, status = ?, scheduled_at = ?, published_at = ?, is_pinned = ?, display_priority = ?, seo_title = ?, seo_description = ?, seo_keywords = ?
WHERE id = ?`,
			nullableUint64(version.CategoryID), version.Title, version.Slug, version.Summary, version.Content, version.CoverURL, version.Status,
			nullableTime(version.ScheduledAt), nullableTime(version.PublishedAt), version.IsPinned, version.DisplayPriority, version.SEOTitle, version.SEODescription, version.SEOKeywords, articleID)
		if err != nil {
			return err
		}
		if err := requireUpdateAffected(ctx, res, func(ctx context.Context) error {
			return articleExistsTx(ctx, tx, articleID)
		}); err != nil {
			return err
		}
		if err := replaceArticleTags(ctx, tx, articleID, version.TagIDs); err != nil {
			return err
		}
		return enqueueArticleRAGSyncTx(ctx, tx, articleID, version.Status, version.ScheduledAt)
	})
}

func enqueueArticleRAGSyncTx(ctx context.Context, tx *sql.Tx, articleID uint64, status string, scheduledAt *time.Time) error {
	operation := RAGSyncOperationUpsert
	runAfter := time.Now().UTC()
	if status != ArticleStatusPublished {
		operation = RAGSyncOperationDelete
	} else if scheduledAt != nil && scheduledAt.After(runAfter) {
		// 定时文章在到点前绝不发送正文给 embedding 上游。
		runAfter = *scheduledAt
	}
	return enqueueRAGSyncJobTx(ctx, tx, articleID, operation, runAfter)
}

func (s *Store) FindArticle(ctx context.Context, id uint64) (*Article, error) {
	row := s.db.QueryRowContext(ctx, "SELECT "+articleSelectFields+" FROM articles WHERE id = ?", id)
	return scanArticle(row)
}

func (s *Store) ArticleSlugExists(ctx context.Context, slug string) (bool, error) {
	var exists int
	if err := s.db.QueryRowContext(ctx, "SELECT 1 FROM articles WHERE slug = ? LIMIT 1", slug).Scan(&exists); err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// ArticleSlugsTakenByPrefix 一次性返回与 base 同名或 base-{n} 形式冲突的已占用 slug 集合，
// 供 uniqueArticleSlug 在内存中找出首个可用候选，避免逐次查询。
// 同时匹配 base 本身（slug = ?）与 base- 前缀族（slug LIKE base-% ESCAPE '!'），
// base 中的 LIKE 通配符经 escapeLike 转义防止语义偏移。
func (s *Store) ArticleSlugsTakenByPrefix(ctx context.Context, base string) (map[string]struct{}, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT slug FROM articles WHERE slug = ? OR slug LIKE ? ESCAPE '!'",
		base, escapeLike(base)+"-%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	taken := make(map[string]struct{})
	for rows.Next() {
		var slug string
		if err := rows.Scan(&slug); err != nil {
			return nil, err
		}
		taken[slug] = struct{}{}
	}
	return taken, rows.Err()
}

func (s *Store) IncreaseArticleView(ctx context.Context, id uint64) error {
	_, err := s.db.ExecContext(ctx, "UPDATE articles SET view_count = view_count + 1 WHERE id = ?", id)
	return err
}

func (s *Store) LikeArticle(ctx context.Context, id uint64, visitorHash string) (uint64, bool, error) {
	var likeCount uint64
	liked := false
	err := WithTx(ctx, s.db, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx, `
INSERT IGNORE INTO article_likes (article_id, visitor_hash)
VALUES (?, ?)`, id, visitorHash)
		if err != nil {
			return err
		}
		affected, err := res.RowsAffected()
		if err != nil {
			return err
		}
		if affected > 0 {
			liked = true
			if _, err := tx.ExecContext(ctx, "UPDATE articles SET like_count = like_count + 1 WHERE id = ?", id); err != nil {
				return err
			}
		}
		return tx.QueryRowContext(ctx, "SELECT like_count FROM articles WHERE id = ?", id).Scan(&likeCount)
	})
	return likeCount, liked, err
}

// queryMode 控制 filter.Query 的拼接方式：queryFulltext 仅走全文索引（命中时使用），
// queryLike 仅走 title LIKE 兜底（fulltext 无命中时使用），queryNone 不拼接 query 条件。
type queryMode int

const (
	queryNone queryMode = iota
	queryFulltext
	queryLike
)

func (s *Store) ListArticles(ctx context.Context, filter ArticleFilter) ([]Article, int64, error) {
	mode := queryNone
	if filter.Query != "" {
		// 先以基础过滤（不含 query）判定 fulltext 是否命中，命中则走全文索引，
		// 否则退化为单字段 title LIKE，避免 MATCH 与 LIKE 用 OR 并列导致全表扫描。
		mode = queryFulltext
		baseWhere, baseArgs := articleWhere(filter, queryNone)
		var checkSQL string
		if baseWhere == "" {
			checkSQL = "SELECT COUNT(*) FROM articles WHERE MATCH(title, content) AGAINST(? IN NATURAL LANGUAGE MODE)"
		} else {
			checkSQL = "SELECT COUNT(*) FROM articles " + baseWhere + " AND MATCH(title, content) AGAINST(? IN NATURAL LANGUAGE MODE)"
		}
		var hits int64
		if err := s.db.QueryRowContext(ctx, checkSQL, append(baseArgs, filter.Query)...).Scan(&hits); err != nil {
			return nil, 0, err
		}
		if hits == 0 {
			mode = queryLike
		}
	}

	where, args := articleWhere(filter, mode)
	countSQL := "SELECT COUNT(*) FROM articles " + where
	var total int64
	if err := s.db.QueryRowContext(ctx, countSQL, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	offset := (filter.Page - 1) * filter.Size
	args = append(args, filter.Size, offset)
	rows, err := s.db.QueryContext(ctx, "SELECT "+articleSelectFields+" FROM articles "+where+" ORDER BY "+articleDisplayOrder+" LIMIT ? OFFSET ?", args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items := make([]Article, 0)
	for rows.Next() {
		item, err := scanArticleRows(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, *item)
	}
	return items, total, rows.Err()
}

func (s *Store) ListPublicArticles(ctx context.Context, limit int) ([]Article, error) {
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	rows, err := s.db.QueryContext(ctx, "SELECT "+articleSelectFields+" FROM articles WHERE status = 'published' AND (scheduled_at IS NULL OR scheduled_at <= NOW()) ORDER BY "+articleDisplayOrder+" LIMIT ?", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]Article, 0)
	for rows.Next() {
		item, err := scanArticleRows(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	return items, rows.Err()
}

func (s *Store) ListPublicArticleEntries(ctx context.Context, limit int) ([]PublicArticleEntry, error) {
	if limit < 1 {
		return []PublicArticleEntry{}, nil
	}
	if limit > 50000 {
		limit = 50000
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT id, title, summary, published_at, created_at, updated_at
FROM articles
WHERE status = 'published' AND (scheduled_at IS NULL OR scheduled_at <= NOW())
ORDER BY `+articleTimeOrder+`
LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]PublicArticleEntry, 0)
	for rows.Next() {
		var item PublicArticleEntry
		var publishedAt sql.NullTime
		if err := rows.Scan(&item.ID, &item.Title, &item.Summary, &publishedAt, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		item.PublishedAt = timeFromNull(publishedAt)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) PublicArticleContext(ctx context.Context, current Article, limit int) (*Article, *Article, []Article, error) {
	if limit < 1 {
		limit = 3
	}
	refTime := current.CreatedAt
	if current.PublishedAt != nil {
		refTime = *current.PublishedAt
	}

	currentPinned := boolOrderValue(current.IsPinned)
	prev, err := findOneArticle(ctx, s.db.QueryRowContext(ctx, "SELECT "+articleSelectFields+` FROM articles
WHERE status = 'published' AND (scheduled_at IS NULL OR scheduled_at <= NOW())
  AND (
    is_pinned < ?
    OR (is_pinned = ? AND display_priority < ?)
    OR (is_pinned = ? AND display_priority = ? AND COALESCE(published_at, created_at) < ?)
    OR (is_pinned = ? AND display_priority = ? AND COALESCE(published_at, created_at) = ? AND id < ?)
  )
ORDER BY `+articleDisplayOrder+`
LIMIT 1`, currentPinned, currentPinned, current.DisplayPriority, currentPinned, current.DisplayPriority, refTime, currentPinned, current.DisplayPriority, refTime, current.ID))
	if err != nil && err != ErrNotFound {
		return nil, nil, nil, err
	}

	next, err := findOneArticle(ctx, s.db.QueryRowContext(ctx, "SELECT "+articleSelectFields+` FROM articles
WHERE status = 'published' AND (scheduled_at IS NULL OR scheduled_at <= NOW())
  AND (
    is_pinned > ?
    OR (is_pinned = ? AND display_priority > ?)
    OR (is_pinned = ? AND display_priority = ? AND COALESCE(published_at, created_at) > ?)
    OR (is_pinned = ? AND display_priority = ? AND COALESCE(published_at, created_at) = ? AND id > ?)
  )
ORDER BY `+articleDisplayOrderAsc+`
LIMIT 1`, currentPinned, currentPinned, current.DisplayPriority, currentPinned, current.DisplayPriority, refTime, currentPinned, current.DisplayPriority, refTime, current.ID))
	if err != nil && err != ErrNotFound {
		return nil, nil, nil, err
	}

	tags, err := s.ArticleTags(ctx, current.ID)
	if err != nil {
		return nil, nil, nil, err
	}
	tagIDs := make([]uint64, 0, len(tags))
	for _, tag := range tags {
		tagIDs = append(tagIDs, tag.ID)
	}

	related, err := s.relatedPublicArticles(ctx, current.ID, current.CategoryID, tagIDs, limit)
	if err != nil {
		return nil, nil, nil, err
	}
	return prev, next, related, nil
}

func (s *Store) ArticleTags(ctx context.Context, articleID uint64) ([]Tag, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT t.id, t.name, t.slug, t.description, t.created_by, t.created_at, t.updated_at
FROM tags t
JOIN article_tags at ON at.tag_id = t.id
WHERE at.article_id = ?
ORDER BY t.id`, articleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]Tag, 0)
	for rows.Next() {
		item, err := scanTag(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	return items, rows.Err()
}

// ArticleTagsBatch 一次性按 article ID 加载多篇文章的标签，用于列表/搜索/索引重建等
// 批量组装场景，避免逐条 ArticleTags 触发的 N+1 查询。
// 返回值按 article_id 分组，组内顺序与单条 ArticleTags 一致（ORDER BY t.id）。
// 空入参返回空 map 且不执行查询。
func (s *Store) ArticleTagsBatch(ctx context.Context, articleIDs []uint64) (map[uint64][]Tag, error) {
	articleIDs = uniqueUint64(articleIDs)
	if len(articleIDs) == 0 {
		return map[uint64][]Tag{}, nil
	}
	placeholders := strings.TrimRight(strings.Repeat("?,", len(articleIDs)), ",")
	args := make([]interface{}, 0, len(articleIDs))
	for _, id := range articleIDs {
		args = append(args, id)
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT at.article_id, t.id, t.name, t.slug, t.description, t.created_by, t.created_at, t.updated_at
FROM article_tags at
JOIN tags t ON at.tag_id = t.id
WHERE at.article_id IN (`+placeholders+`)
ORDER BY at.article_id, t.id`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make(map[uint64][]Tag, len(articleIDs))
	for rows.Next() {
		var articleID uint64
		var item Tag
		var description sql.NullString
		if err := rows.Scan(&articleID, &item.ID, &item.Name, &item.Slug, &description, &item.CreatedBy, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, scanErr(err)
		}
		item.Description = stringFromNull(description)
		out[articleID] = append(out[articleID], item)
	}
	return out, rows.Err()
}

func (s *Store) EnsureTagsExist(ctx context.Context, tagIDs []uint64) error {
	if len(tagIDs) == 0 {
		return nil
	}
	ids := uniqueUint64(tagIDs)
	placeholders := strings.TrimRight(strings.Repeat("?,", len(ids)), ",")
	args := make([]interface{}, 0, len(ids))
	for _, id := range ids {
		args = append(args, id)
	}
	var count int
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM tags WHERE id IN ("+placeholders+")", args...).Scan(&count); err != nil {
		return err
	}
	if count != len(ids) {
		return ErrNotFound
	}
	return nil
}

func (s *Store) ListArticleVersions(ctx context.Context, articleID uint64, page, size int) ([]ArticleVersion, int64, error) {
	offset := (page - 1) * size
	var total int64
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM article_versions WHERE article_id = ?", articleID).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := s.db.QueryContext(ctx, "SELECT "+articleVersionSelectFields+" FROM article_versions WHERE article_id = ? ORDER BY version_no DESC LIMIT ? OFFSET ?", articleID, size, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items := make([]ArticleVersion, 0)
	for rows.Next() {
		item, err := scanArticleVersionRows(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, *item)
	}
	return items, total, rows.Err()
}

func (s *Store) FindArticleVersion(ctx context.Context, articleID uint64, versionNo int) (*ArticleVersion, error) {
	row := s.db.QueryRowContext(ctx, "SELECT "+articleVersionSelectFields+" FROM article_versions WHERE article_id = ? AND version_no = ?", articleID, versionNo)
	return scanArticleVersion(row)
}

func articleWhere(filter ArticleFilter, mode queryMode) (string, []interface{}) {
	clauses := make([]string, 0)
	args := make([]interface{}, 0)
	if isContentRole(filter.Role) {
		switch filter.Status {
		case ArticleStatusScheduled:
			clauses = append(clauses, "status = 'published' AND scheduled_at > NOW()")
		case "":
		default:
			// admin 筛选 published 时与 AdminStats.PublishedTotal 对齐：仅“已发布且当前可见”，
			// 不含未到点的排程文章（排程中应通过 status=scheduled 筛选），避免仪表盘与列表数字对不上。
			if filter.Status == ArticleStatusPublished {
				clauses = append(clauses, "status = 'published' AND (scheduled_at IS NULL OR scheduled_at <= NOW())")
			} else {
				clauses = append(clauses, "status = ?")
				args = append(args, filter.Status)
			}
		}
	} else if filter.UserID > 0 {
		switch filter.Status {
		case ArticleStatusScheduled:
			clauses = append(clauses, "author_id = ? AND status = 'published' AND scheduled_at > NOW()")
			args = append(args, filter.UserID)
		case "":
			clauses = append(clauses, "(status = 'published' AND (scheduled_at IS NULL OR scheduled_at <= NOW()) OR author_id = ?)")
			args = append(args, filter.UserID)
		default:
			clauses = append(clauses, "author_id = ? AND status = ?")
			args = append(args, filter.UserID, filter.Status)
		}
	} else {
		clauses = append(clauses, "status = 'published' AND (scheduled_at IS NULL OR scheduled_at <= NOW())")
	}
	if filter.Query != "" && mode != queryNone {
		likeQuery := "%" + escapeLike(filter.Query) + "%"
		switch mode {
		case queryFulltext:
			// fulltext 命中时仅走全文索引，避免 LIKE 拖累索引。
			clauses = append(clauses, "MATCH(title, content) AGAINST(? IN NATURAL LANGUAGE MODE)")
			args = append(args, filter.Query)
		case queryLike:
			// fulltext 无命中时退化为主键字段单 LIKE，去掉 summary/content 冗余 LIKE。
			clauses = append(clauses, "title LIKE ? ESCAPE '!'")
			args = append(args, likeQuery)
		}
	}
	if filter.CategoryID > 0 {
		clauses = append(clauses, "category_id = ?")
		args = append(args, filter.CategoryID)
	}
	if filter.TagID > 0 {
		clauses = append(clauses, "EXISTS (SELECT 1 FROM article_tags at WHERE at.article_id = articles.id AND at.tag_id = ?)")
		args = append(args, filter.TagID)
	}
	if len(clauses) == 0 {
		return "", args
	}
	return "WHERE " + strings.Join(clauses, " AND "), args
}

func escapeLike(value string) string {
	var b strings.Builder
	b.Grow(len(value))
	for _, r := range value {
		switch r {
		case '!', '%', '_':
			b.WriteRune('!')
		}
		b.WriteRune(r)
	}
	return b.String()
}

func replaceArticleTags(ctx context.Context, tx *sql.Tx, articleID uint64, tagIDs []uint64) error {
	if _, err := tx.ExecContext(ctx, "DELETE FROM article_tags WHERE article_id = ?", articleID); err != nil {
		return err
	}
	tagIDs = uniqueUint64(tagIDs)
	if len(tagIDs) == 0 {
		return nil
	}
	var b strings.Builder
	b.WriteString("INSERT INTO article_tags (article_id, tag_id) VALUES ")
	args := make([]any, 0, len(tagIDs)*2)
	for i, tagID := range tagIDs {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString("(?, ?)")
		args = append(args, articleID, tagID)
	}
	_, err := tx.ExecContext(ctx, b.String(), args...)
	return err
}

func scanArticle(row rowScanner) (*Article, error) {
	var item Article
	var categoryID sql.NullInt64
	var summary sql.NullString
	var coverURL sql.NullString
	var seoTitle sql.NullString
	var seoDescription sql.NullString
	var seoKeywords sql.NullString
	var scheduledAt sql.NullTime
	var publishedAt sql.NullTime
	var isPinned int
	err := row.Scan(
		&item.ID, &item.AuthorID, &categoryID, &item.Title, &item.Slug, &summary, &item.Content,
		&coverURL, &item.Status, &item.ViewCount, &item.LikeCount, &scheduledAt, &publishedAt, &isPinned, &item.DisplayPriority, &seoTitle,
		&seoDescription, &seoKeywords, &item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		return nil, scanErr(err)
	}
	item.CategoryID = uint64FromNull(categoryID)
	item.Summary = stringFromNull(summary)
	item.CoverURL = stringFromNull(coverURL)
	item.ScheduledAt = timeFromNull(scheduledAt)
	item.PublishedAt = timeFromNull(publishedAt)
	item.IsPinned = isPinned != 0
	item.SEOTitle = stringFromNull(seoTitle)
	item.SEODescription = stringFromNull(seoDescription)
	item.SEOKeywords = stringFromNull(seoKeywords)
	return &item, nil
}

func scanArticleRows(rows *sql.Rows) (*Article, error) {
	return scanArticle(rows)
}

func scanArticleVersion(row rowScanner) (*ArticleVersion, error) {
	var item ArticleVersion
	var categoryID sql.NullInt64
	var summary sql.NullString
	var coverURL sql.NullString
	var seoTitle sql.NullString
	var seoDescription sql.NullString
	var seoKeywords sql.NullString
	var scheduledAt sql.NullTime
	var publishedAt sql.NullTime
	var originalCreatedAt sql.NullTime
	var originalUpdatedAt sql.NullTime
	var tagIDsRaw string
	var isPinned int
	err := row.Scan(
		&item.ID, &item.ArticleID, &item.VersionNo, &item.ChangedBy, &item.AuthorID, &categoryID,
		&item.Title, &item.Slug, &summary, &item.Content, &coverURL, &item.Status,
		&item.ViewCount, &item.LikeCount, &scheduledAt, &publishedAt, &isPinned, &item.DisplayPriority, &seoTitle, &seoDescription,
		&seoKeywords, &tagIDsRaw, &originalCreatedAt, &originalUpdatedAt, &item.CreatedAt,
	)
	if err != nil {
		return nil, scanErr(err)
	}
	item.CategoryID = uint64FromNull(categoryID)
	item.Summary = stringFromNull(summary)
	item.CoverURL = stringFromNull(coverURL)
	item.ScheduledAt = timeFromNull(scheduledAt)
	item.PublishedAt = timeFromNull(publishedAt)
	item.IsPinned = isPinned != 0
	item.SEOTitle = stringFromNull(seoTitle)
	item.SEODescription = stringFromNull(seoDescription)
	item.SEOKeywords = stringFromNull(seoKeywords)
	item.OriginalCreatedAt = timeFromNull(originalCreatedAt)
	item.OriginalUpdatedAt = timeFromNull(originalUpdatedAt)
	if err := json.Unmarshal([]byte(tagIDsRaw), &item.TagIDs); err != nil {
		logx.Errorf("scanArticleVersion: unmarshal tag_ids failed article_id=%d version_no=%d raw=%q err=%v", item.ArticleID, item.VersionNo, tagIDsRaw, err)
		item.TagIDs = nil
	}
	return &item, nil
}

func scanArticleVersionRows(rows *sql.Rows) (*ArticleVersion, error) {
	return scanArticleVersion(rows)
}

func createArticleVersion(ctx context.Context, tx *sql.Tx, articleID, changedBy uint64) error {
	article, err := scanArticle(tx.QueryRowContext(ctx, "SELECT "+articleSelectFields+" FROM articles WHERE id = ?", articleID))
	if err != nil {
		return err
	}

	tagIDs, err := articleTagIDs(ctx, tx, articleID)
	if err != nil {
		return err
	}
	tagIDsJSON, err := json.Marshal(tagIDs)
	if err != nil {
		return err
	}
	var versionNo int
	// FOR UPDATE 在事务内对 article_id 的版本行加排他锁（空结果集时 RR 隔离级别下加 gap lock），
	// 串行化并发版本号分配，避免两个事务读到相同 MAX(version_no)+1 后依赖唯一键冲突兜底。
	// uniq_article_version (article_id, version_no) 仍作为最终兜底。
	if err := tx.QueryRowContext(ctx, "SELECT COALESCE(MAX(version_no), 0) + 1 FROM article_versions WHERE article_id = ? FOR UPDATE", articleID).Scan(&versionNo); err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `
INSERT INTO article_versions (article_id, version_no, changed_by, author_id, category_id, title, slug, summary, content, cover_url, status, view_count, like_count, scheduled_at, published_at, is_pinned, display_priority, seo_title, seo_description, seo_keywords, tag_ids, original_created_at, original_updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		articleID, versionNo, changedBy, article.AuthorID, nullableUint64(article.CategoryID), article.Title, article.Slug, article.Summary,
		article.Content, article.CoverURL, article.Status, article.ViewCount, article.LikeCount, nullableTime(article.ScheduledAt), nullableTime(article.PublishedAt),
		article.IsPinned, article.DisplayPriority, article.SEOTitle, article.SEODescription, article.SEOKeywords, string(tagIDsJSON), article.CreatedAt, article.UpdatedAt)
	return err
}

func findArticleVersion(ctx context.Context, tx *sql.Tx, articleID uint64, versionNo int) (*ArticleVersion, error) {
	return scanArticleVersion(tx.QueryRowContext(ctx, "SELECT "+articleVersionSelectFields+" FROM article_versions WHERE article_id = ? AND version_no = ?", articleID, versionNo))
}

func articleTagIDs(ctx context.Context, tx *sql.Tx, articleID uint64) ([]uint64, error) {
	rows, err := tx.QueryContext(ctx, "SELECT tag_id FROM article_tags WHERE article_id = ? ORDER BY tag_id", articleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tagIDs := make([]uint64, 0)
	for rows.Next() {
		var tagID uint64
		if err := rows.Scan(&tagID); err != nil {
			return nil, err
		}
		tagIDs = append(tagIDs, tagID)
	}
	return tagIDs, rows.Err()
}

func findOneArticle(ctx context.Context, row *sql.Row) (*Article, error) {
	return scanArticle(row)
}

func articleExistsTx(ctx context.Context, tx *sql.Tx, articleID uint64) error {
	var exists int
	if err := tx.QueryRowContext(ctx, "SELECT 1 FROM articles WHERE id = ? LIMIT 1", articleID).Scan(&exists); err != nil {
		return scanErr(err)
	}
	return nil
}

func (s *Store) relatedPublicArticles(ctx context.Context, articleID, categoryID uint64, tagIDs []uint64, limit int) ([]Article, error) {
	clauses := []string{"status = 'published'", "(scheduled_at IS NULL OR scheduled_at <= NOW())", "id <> ?"}
	args := []interface{}{articleID}
	relatedClauses := make([]string, 0)
	if categoryID > 0 {
		relatedClauses = append(relatedClauses, "category_id = ?")
		args = append(args, categoryID)
	}
	if len(tagIDs) > 0 {
		placeholders := strings.TrimRight(strings.Repeat("?,", len(tagIDs)), ",")
		relatedClauses = append(relatedClauses, "EXISTS (SELECT 1 FROM article_tags at WHERE at.article_id = articles.id AND at.tag_id IN ("+placeholders+"))")
		for _, tagID := range tagIDs {
			args = append(args, tagID)
		}
	}
	if len(relatedClauses) > 0 {
		clauses = append(clauses, "("+strings.Join(relatedClauses, " OR ")+")")
	}
	args = append(args, limit)

	rows, err := s.db.QueryContext(ctx, "SELECT "+articleSelectFields+" FROM articles WHERE "+strings.Join(clauses, " AND ")+" ORDER BY "+articleDisplayOrder+" LIMIT ?", args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]Article, 0)
	for rows.Next() {
		item, err := scanArticleRows(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	return items, rows.Err()
}

func publishedAtForCreate(status string, scheduledAt *time.Time) *time.Time {
	if status != ArticleStatusPublished {
		return nil
	}
	if scheduledAt != nil {
		return scheduledAt
	}
	now := time.Now()
	return &now
}

func publishedAtForUpdate(currentStatus string, currentPublished *time.Time, nextStatus string, scheduledAt *time.Time) *time.Time {
	if nextStatus != ArticleStatusPublished {
		// 非 published（draft/archived/scheduled）保留原 published_at，
		// 避免暂存/取消发布后重新发布丢失原始首发时间（P4-4）。
		return currentPublished
	}
	if scheduledAt != nil {
		return scheduledAt
	}
	if currentStatus == ArticleStatusPublished && currentPublished != nil {
		return currentPublished
	}
	now := time.Now()
	return &now
}

func IsArticlePubliclyVisible(item Article, now time.Time) bool {
	if item.Status != ArticleStatusPublished {
		return false
	}
	return item.ScheduledAt == nil || !item.ScheduledAt.After(now)
}

func boolOrderValue(value bool) int {
	if value {
		return 1
	}
	return 0
}

func isContentRole(role string) bool {
	return role == "admin" || role == "editor"
}

func uniqueUint64(values []uint64) []uint64 {
	seen := make(map[uint64]struct{}, len(values))
	out := make([]uint64, 0, len(values))
	for _, value := range values {
		if value == 0 {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}
