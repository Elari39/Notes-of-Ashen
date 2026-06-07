package model

import (
	"context"
	"database/sql"
	"time"
)

type Category struct {
	ID          uint64
	Name        string
	Slug        string
	Description string
	CreatedBy   uint64
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type Tag struct {
	ID          uint64
	Name        string
	Slug        string
	Description string
	CreatedBy   uint64
	CreatedAt   time.Time
	UpdatedAt   time.Time
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
	return requireAffected(res)
}

func (s *Store) DeleteCategory(ctx context.Context, id uint64) error {
	res, err := s.db.ExecContext(ctx, "DELETE FROM categories WHERE id = ?", id)
	if err != nil {
		return err
	}
	return requireAffected(res)
}

func (s *Store) FindCategory(ctx context.Context, id uint64) (*Category, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT id, name, slug, description, created_by, created_at, updated_at
FROM categories WHERE id = ?`, id)
	return scanCategory(row)
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

func (s *Store) ListCategories(ctx context.Context, page, size int) ([]Category, int64, error) {
	offset := (page - 1) * size
	var total int64
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM categories").Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT id, name, slug, description, created_by, created_at, updated_at
FROM categories ORDER BY id DESC LIMIT ? OFFSET ?`, size, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items := make([]Category, 0)
	for rows.Next() {
		var item Category
		if err := rows.Scan(&item.ID, &item.Name, &item.Slug, &item.Description, &item.CreatedBy, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, 0, err
		}
		items = append(items, item)
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
	return requireAffected(res)
}

func (s *Store) DeleteTag(ctx context.Context, id uint64) error {
	res, err := s.db.ExecContext(ctx, "DELETE FROM tags WHERE id = ?", id)
	if err != nil {
		return err
	}
	return requireAffected(res)
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

func (s *Store) ListTags(ctx context.Context, page, size int) ([]Tag, int64, error) {
	offset := (page - 1) * size
	var total int64
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM tags").Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT id, name, slug, description, created_by, created_at, updated_at
FROM tags ORDER BY id DESC LIMIT ? OFFSET ?`, size, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items := make([]Tag, 0)
	for rows.Next() {
		var item Tag
		if err := rows.Scan(&item.ID, &item.Name, &item.Slug, &item.Description, &item.CreatedBy, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	return items, total, rows.Err()
}

func scanCategory(row *sql.Row) (*Category, error) {
	var item Category
	err := row.Scan(&item.ID, &item.Name, &item.Slug, &item.Description, &item.CreatedBy, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return nil, scanErr(err)
	}
	return &item, nil
}

func scanTag(row *sql.Row) (*Tag, error) {
	var item Tag
	err := row.Scan(&item.ID, &item.Name, &item.Slug, &item.Description, &item.CreatedBy, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return nil, scanErr(err)
	}
	return &item, nil
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
