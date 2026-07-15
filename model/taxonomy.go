package model

import (
	"context"
	"database/sql"
	"strings"
	"time"
)

type Category struct {
	ID           uint64
	Name         string
	Slug         string
	Description  string
	CreatedBy    uint64
	CreatedAt    time.Time
	UpdatedAt    time.Time
	ArticleCount int64
}

type Tag struct {
	ID           uint64
	Name         string
	Slug         string
	Description  string
	CreatedBy    uint64
	CreatedAt    time.Time
	UpdatedAt    time.Time
	ArticleCount int64
}

type TaxonomyCreate struct {
	Name        string
	Slug        string
	Description string
	CreatedBy   uint64
}

type TaxonomyUpdate struct {
	Name        string
	Slug        string
	Description string
}

func (s *Store) CreateCategory(ctx context.Context, in TaxonomyCreate) (uint64, error) {
	res, err := s.db.ExecContext(ctx, `
INSERT INTO categories (name, slug, description, created_by)
VALUES (?, ?, ?, ?)`, in.Name, in.Slug, in.Description, in.CreatedBy)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	return uint64(id), err
}

func (s *Store) UpdateCategory(ctx context.Context, id uint64, in TaxonomyUpdate) error {
	res, err := s.db.ExecContext(ctx, `
UPDATE categories SET name = ?, slug = ?, description = ? WHERE id = ?`,
		in.Name, in.Slug, in.Description, id)
	if err != nil {
		return err
	}
	return requireUpdateAffected(ctx, res, func(ctx context.Context) error {
		_, err := s.FindCategory(ctx, id)
		return err
	})
}

func (s *Store) DeleteCategory(ctx context.Context, id uint64) error {
	return WithTx(ctx, s.db, func(tx *sql.Tx) error {
		return deleteCategoryTx(ctx, tx, id)
	})
}

func (s *Store) FindCategory(ctx context.Context, id uint64) (*Category, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT id, name, slug, description, created_by, created_at, updated_at
FROM categories WHERE id = ?`, id)
	return scanCategory(row)
}

// FindCategoriesByIDs 一次性加载多个分类，用于列表/搜索/索引重建等批量组装场景，
// 避免逐条 FindCategory 触发的 N+1 查询。空入参返回空 map 且不执行查询。
func (s *Store) FindCategoriesByIDs(ctx context.Context, ids []uint64) (map[uint64]Category, error) {
	ids = uniqueUint64(ids)
	if len(ids) == 0 {
		return map[uint64]Category{}, nil
	}
	placeholders := strings.TrimRight(strings.Repeat("?,", len(ids)), ",")
	args := make([]interface{}, 0, len(ids))
	for _, id := range ids {
		args = append(args, id)
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT id, name, slug, description, created_by, created_at, updated_at
FROM categories WHERE id IN (`+placeholders+`)`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make(map[uint64]Category, len(ids))
	for rows.Next() {
		item, err := scanCategory(rows)
		if err != nil {
			return nil, err
		}
		out[item.ID] = *item
	}
	return out, rows.Err()
}

func (s *Store) FindCategoryByNameOrSlug(ctx context.Context, name, slug string) (*Category, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT id, name, slug, description, created_by, created_at, updated_at
FROM categories
WHERE name = ? OR slug = ?
ORDER BY CASE WHEN slug = ? THEN 0 ELSE 1 END
LIMIT 1`, name, slug, slug)
	return scanCategory(row)
}

func (s *Store) ListCategories(ctx context.Context, page, size int, publicOnly bool) ([]Category, int64, error) {
	offset := (page - 1) * size
	var total int64
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM categories").Scan(&total); err != nil {
		return nil, 0, err
	}
	visibility := ""
	if publicOnly {
		visibility = " AND a.status = 'published' AND (a.scheduled_at IS NULL OR a.scheduled_at <= NOW())"
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT c.id, c.name, c.slug, c.description, c.created_by, c.created_at, c.updated_at, COUNT(a.id)
FROM categories c
LEFT JOIN articles a ON a.category_id = c.id`+visibility+`
GROUP BY c.id, c.name, c.slug, c.description, c.created_by, c.created_at, c.updated_at
ORDER BY c.id DESC LIMIT ? OFFSET ?`, size, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items := make([]Category, 0)
	for rows.Next() {
		item, err := scanCategoryWithCount(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, *item)
	}
	return items, total, rows.Err()
}

func (s *Store) CreateTag(ctx context.Context, in TaxonomyCreate) (uint64, error) {
	res, err := s.db.ExecContext(ctx, `
INSERT INTO tags (name, slug, description, created_by)
VALUES (?, ?, ?, ?)`, in.Name, in.Slug, in.Description, in.CreatedBy)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	return uint64(id), err
}

func (s *Store) UpdateTag(ctx context.Context, id uint64, in TaxonomyUpdate) error {
	res, err := s.db.ExecContext(ctx, `
UPDATE tags SET name = ?, slug = ?, description = ? WHERE id = ?`,
		in.Name, in.Slug, in.Description, id)
	if err != nil {
		return err
	}
	return requireUpdateAffected(ctx, res, func(ctx context.Context) error {
		_, err := s.FindTag(ctx, id)
		return err
	})
}

func (s *Store) DeleteTag(ctx context.Context, id uint64) error {
	return WithTx(ctx, s.db, func(tx *sql.Tx) error {
		return deleteTagTx(ctx, tx, id)
	})
}

func (s *Store) FindTag(ctx context.Context, id uint64) (*Tag, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT id, name, slug, description, created_by, created_at, updated_at
FROM tags WHERE id = ?`, id)
	return scanTag(row)
}

func (s *Store) FindTagByNameOrSlug(ctx context.Context, name, slug string) (*Tag, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT id, name, slug, description, created_by, created_at, updated_at
FROM tags
WHERE name = ? OR slug = ?
ORDER BY CASE WHEN slug = ? THEN 0 ELSE 1 END
LIMIT 1`, name, slug, slug)
	return scanTag(row)
}

func (s *Store) ListTags(ctx context.Context, page, size int, publicOnly bool) ([]Tag, int64, error) {
	offset := (page - 1) * size
	var total int64
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM tags").Scan(&total); err != nil {
		return nil, 0, err
	}
	visibility := ""
	if publicOnly {
		visibility = " AND a.status = 'published' AND (a.scheduled_at IS NULL OR a.scheduled_at <= NOW())"
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT t.id, t.name, t.slug, t.description, t.created_by, t.created_at, t.updated_at, COUNT(a.id)
FROM tags t
LEFT JOIN article_tags at ON at.tag_id = t.id
LEFT JOIN articles a ON a.id = at.article_id`+visibility+`
GROUP BY t.id, t.name, t.slug, t.description, t.created_by, t.created_at, t.updated_at
ORDER BY t.id DESC LIMIT ? OFFSET ?`, size, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items := make([]Tag, 0)
	for rows.Next() {
		item, err := scanTagWithCount(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, *item)
	}
	return items, total, rows.Err()
}

func (s *Store) SuggestCategories(ctx context.Context, query string, limit int) ([]Category, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT c.id, c.name, c.slug, c.description, c.created_by, c.created_at, c.updated_at, COUNT(a.id)
FROM categories c
JOIN articles a ON a.category_id = c.id
  AND a.status = 'published' AND (a.scheduled_at IS NULL OR a.scheduled_at <= NOW())
WHERE c.name LIKE ?
GROUP BY c.id, c.name, c.slug, c.description, c.created_by, c.created_at, c.updated_at
ORDER BY CASE WHEN c.name LIKE ? THEN 0 ELSE 1 END, COUNT(a.id) DESC, c.name
LIMIT ?`, "%"+query+"%", query+"%", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]Category, 0, limit)
	for rows.Next() {
		item, err := scanCategoryWithCount(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	return items, rows.Err()
}

func (s *Store) SuggestTags(ctx context.Context, query string, limit int) ([]Tag, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT t.id, t.name, t.slug, t.description, t.created_by, t.created_at, t.updated_at, COUNT(a.id)
FROM tags t
JOIN article_tags at ON at.tag_id = t.id
JOIN articles a ON a.id = at.article_id
  AND a.status = 'published' AND (a.scheduled_at IS NULL OR a.scheduled_at <= NOW())
WHERE t.name LIKE ?
GROUP BY t.id, t.name, t.slug, t.description, t.created_by, t.created_at, t.updated_at
ORDER BY CASE WHEN t.name LIKE ? THEN 0 ELSE 1 END, COUNT(a.id) DESC, t.name
LIMIT ?`, "%"+query+"%", query+"%", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]Tag, 0, limit)
	for rows.Next() {
		item, err := scanTagWithCount(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	return items, rows.Err()
}

func scanCategory(row rowScanner) (*Category, error) {
	var item Category
	var description sql.NullString
	err := row.Scan(&item.ID, &item.Name, &item.Slug, &description, &item.CreatedBy, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return nil, scanErr(err)
	}
	item.Description = stringFromNull(description)
	return &item, nil
}

func scanTag(row rowScanner) (*Tag, error) {
	var item Tag
	var description sql.NullString
	err := row.Scan(&item.ID, &item.Name, &item.Slug, &description, &item.CreatedBy, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return nil, scanErr(err)
	}
	item.Description = stringFromNull(description)
	return &item, nil
}

func scanCategoryWithCount(row rowScanner) (*Category, error) {
	var item Category
	var description sql.NullString
	err := row.Scan(&item.ID, &item.Name, &item.Slug, &description, &item.CreatedBy, &item.CreatedAt, &item.UpdatedAt, &item.ArticleCount)
	if err != nil {
		return nil, scanErr(err)
	}
	item.Description = stringFromNull(description)
	return &item, nil
}

func scanTagWithCount(row rowScanner) (*Tag, error) {
	var item Tag
	var description sql.NullString
	err := row.Scan(&item.ID, &item.Name, &item.Slug, &description, &item.CreatedBy, &item.CreatedAt, &item.UpdatedAt, &item.ArticleCount)
	if err != nil {
		return nil, scanErr(err)
	}
	item.Description = stringFromNull(description)
	return &item, nil
}

type execContexter interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
}

func deleteCategoryTx(ctx context.Context, execer execContexter, id uint64) error {
	if _, err := execer.ExecContext(ctx, "UPDATE articles SET category_id = NULL WHERE category_id = ?", id); err != nil {
		return err
	}
	res, err := execer.ExecContext(ctx, "DELETE FROM categories WHERE id = ?", id)
	if err != nil {
		return err
	}
	return requireAffected(res)
}

func deleteTagTx(ctx context.Context, execer execContexter, id uint64) error {
	if _, err := execer.ExecContext(ctx, "DELETE FROM article_tags WHERE tag_id = ?", id); err != nil {
		return err
	}
	if _, err := execer.ExecContext(ctx, "DELETE FROM project_tags WHERE tag_id = ?", id); err != nil {
		return err
	}
	res, err := execer.ExecContext(ctx, "DELETE FROM tags WHERE id = ?", id)
	if err != nil {
		return err
	}
	return requireAffected(res)
}

func requireAffected(res sql.Result) error {
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrNotFound
	}
	return nil
}

func requireUpdateAffected(ctx context.Context, res sql.Result, exists func(context.Context) error) error {
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected > 0 {
		return nil
	}
	return exists(ctx)
}
