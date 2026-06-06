package model

import (
	"context"
	"database/sql"
	"strings"
	"time"
)

type Article struct {
	ID          uint64
	AuthorID    uint64
	CategoryID  uint64
	Title       string
	Slug        string
	Summary     string
	Content     string
	CoverURL    string
	Status      string
	ViewCount   uint64
	PublishedAt *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type ArticleCreate struct {
	AuthorID   uint64
	CategoryID uint64
	Title      string
	Slug       string
	Summary    string
	Content    string
	CoverURL   string
	Status     string
	TagIDs     []uint64
}

type ArticleUpdate struct {
	CategoryID uint64
	Title      string
	Slug       string
	Summary    string
	Content    string
	CoverURL   string
	Status     string
	TagIDs     []uint64
}

type ArticleFilter struct {
	UserID uint64
	Role   string
	Status string
	Page   int
	Size   int
}

func (s *Store) CreateArticle(ctx context.Context, in ArticleCreate) (uint64, error) {
	var id uint64
	err := WithTx(ctx, s.db, func(tx *sql.Tx) error {
		var publishedAt interface{}
		if in.Status == "published" {
			publishedAt = time.Now()
		}
		res, err := tx.ExecContext(ctx, `
INSERT INTO articles (author_id, category_id, title, slug, summary, content, cover_url, status, published_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			in.AuthorID, nullableUint64(in.CategoryID), in.Title, in.Slug, in.Summary, in.Content, in.CoverURL, in.Status, publishedAt)
		if err != nil {
			return err
		}
		insertID, err := res.LastInsertId()
		if err != nil {
			return err
		}
		id = uint64(insertID)
		return replaceArticleTags(ctx, tx, id, in.TagIDs)
	})
	return id, err
}

func (s *Store) UpdateArticle(ctx context.Context, id uint64, in ArticleUpdate) error {
	return WithTx(ctx, s.db, func(tx *sql.Tx) error {
		var currentStatus string
		var currentPublished sql.NullTime
		if err := tx.QueryRowContext(ctx, "SELECT status, published_at FROM articles WHERE id = ?", id).Scan(&currentStatus, &currentPublished); err != nil {
			return scanErr(err)
		}

		var publishedAt interface{} = nil
		if currentPublished.Valid {
			publishedAt = currentPublished.Time
		}
		if currentStatus != "published" && in.Status == "published" {
			publishedAt = time.Now()
		}
		if in.Status != "published" {
			publishedAt = nil
		}

		res, err := tx.ExecContext(ctx, `
UPDATE articles
SET category_id = ?, title = ?, slug = ?, summary = ?, content = ?, cover_url = ?, status = ?, published_at = ?
WHERE id = ?`,
			nullableUint64(in.CategoryID), in.Title, in.Slug, in.Summary, in.Content, in.CoverURL, in.Status, nullableTimePointer(publishedAt), id)
		if err != nil {
			return err
		}
		if err := requireAffected(res); err != nil {
			return err
		}
		return replaceArticleTags(ctx, tx, id, in.TagIDs)
	})
}

func (s *Store) DeleteArticle(ctx context.Context, id uint64) error {
	res, err := s.db.ExecContext(ctx, "DELETE FROM articles WHERE id = ?", id)
	if err != nil {
		return err
	}
	return requireAffected(res)
}

func (s *Store) UpdateArticleStatus(ctx context.Context, id uint64, status string) error {
	var publishedAt interface{}
	if status == "published" {
		publishedAt = time.Now()
	}
	res, err := s.db.ExecContext(ctx, `
UPDATE articles
SET status = ?, published_at = CASE WHEN ? = 'published' AND published_at IS NULL THEN ? WHEN ? <> 'published' THEN NULL ELSE published_at END
WHERE id = ?`, status, status, publishedAt, status, id)
	if err != nil {
		return err
	}
	return requireAffected(res)
}

func (s *Store) FindArticle(ctx context.Context, id uint64) (*Article, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT id, author_id, category_id, title, slug, summary, content, cover_url, status, view_count, published_at, created_at, updated_at
FROM articles WHERE id = ?`, id)
	return scanArticle(row)
}

func (s *Store) IncreaseArticleView(ctx context.Context, id uint64) error {
	_, err := s.db.ExecContext(ctx, "UPDATE articles SET view_count = view_count + 1 WHERE id = ?", id)
	return err
}

func (s *Store) ListArticles(ctx context.Context, filter ArticleFilter) ([]Article, int64, error) {
	where, args := articleWhere(filter)
	countSQL := "SELECT COUNT(*) FROM articles " + where
	var total int64
	if err := s.db.QueryRowContext(ctx, countSQL, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	offset := (filter.Page - 1) * filter.Size
	args = append(args, filter.Size, offset)
	rows, err := s.db.QueryContext(ctx, `
SELECT id, author_id, category_id, title, slug, summary, content, cover_url, status, view_count, published_at, created_at, updated_at
FROM articles `+where+` ORDER BY created_at DESC, id DESC LIMIT ? OFFSET ?`, args...)
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
		var item Tag
		if err := rows.Scan(&item.ID, &item.Name, &item.Slug, &item.Description, &item.CreatedBy, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) EnsureTagsExist(ctx context.Context, tagIDs []uint64) error {
	if len(tagIDs) == 0 {
		return nil
	}
	placeholders := strings.TrimRight(strings.Repeat("?,", len(tagIDs)), ",")
	args := make([]interface{}, 0, len(tagIDs))
	for _, id := range tagIDs {
		args = append(args, id)
	}
	var count int
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM tags WHERE id IN ("+placeholders+")", args...).Scan(&count); err != nil {
		return err
	}
	if count != len(uniqueUint64(tagIDs)) {
		return ErrNotFound
	}
	return nil
}

func articleWhere(filter ArticleFilter) (string, []interface{}) {
	clauses := make([]string, 0)
	args := make([]interface{}, 0)
	if filter.Role == "admin" {
		if filter.Status != "" {
			clauses = append(clauses, "status = ?")
			args = append(args, filter.Status)
		}
	} else if filter.UserID > 0 {
		if filter.Status != "" {
			clauses = append(clauses, "author_id = ? AND status = ?")
			args = append(args, filter.UserID, filter.Status)
		} else {
			clauses = append(clauses, "(status = 'published' OR author_id = ?)")
			args = append(args, filter.UserID)
		}
	} else {
		clauses = append(clauses, "status = 'published'")
	}
	if len(clauses) == 0 {
		return "", args
	}
	return "WHERE " + strings.Join(clauses, " AND "), args
}

func replaceArticleTags(ctx context.Context, tx *sql.Tx, articleID uint64, tagIDs []uint64) error {
	if _, err := tx.ExecContext(ctx, "DELETE FROM article_tags WHERE article_id = ?", articleID); err != nil {
		return err
	}
	for _, tagID := range uniqueUint64(tagIDs) {
		if _, err := tx.ExecContext(ctx, "INSERT INTO article_tags (article_id, tag_id) VALUES (?, ?)", articleID, tagID); err != nil {
			return err
		}
	}
	return nil
}

func scanArticle(row *sql.Row) (*Article, error) {
	var item Article
	var categoryID sql.NullInt64
	var publishedAt sql.NullTime
	err := row.Scan(&item.ID, &item.AuthorID, &categoryID, &item.Title, &item.Slug, &item.Summary, &item.Content, &item.CoverURL, &item.Status, &item.ViewCount, &publishedAt, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return nil, scanErr(err)
	}
	item.CategoryID = uint64FromNull(categoryID)
	item.PublishedAt = timeFromNull(publishedAt)
	return &item, nil
}

func scanArticleRows(rows *sql.Rows) (*Article, error) {
	var item Article
	var categoryID sql.NullInt64
	var publishedAt sql.NullTime
	err := rows.Scan(&item.ID, &item.AuthorID, &categoryID, &item.Title, &item.Slug, &item.Summary, &item.Content, &item.CoverURL, &item.Status, &item.ViewCount, &publishedAt, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return nil, err
	}
	item.CategoryID = uint64FromNull(categoryID)
	item.PublishedAt = timeFromNull(publishedAt)
	return &item, nil
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

func nullableTimePointer(value interface{}) interface{} {
	return value
}
