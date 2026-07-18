package model

import (
	"context"
	"database/sql"
	"strings"
	"time"
)

type MediaAsset struct {
	ID           uint64
	StorageKey   string
	OriginalName string
	MIMEType     string
	SizeBytes    uint64
	Width        uint
	Height       uint
	AltText      string
	SHA256       string
	CreatedBy    uint64
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type MediaAssetCreate struct {
	StorageKey   string
	OriginalName string
	MIMEType     string
	SizeBytes    uint64
	Width        uint
	Height       uint
	AltText      string
	SHA256       string
	CreatedBy    uint64
}

const mediaSelectFields = "id, storage_key, original_name, mime_type, size_bytes, width, height, alt_text, sha256, created_by, created_at, updated_at"

func (s *Store) CreateMediaAsset(ctx context.Context, in MediaAssetCreate) (uint64, error) {
	res, err := s.db.ExecContext(ctx, `
INSERT INTO media_assets (storage_key, original_name, mime_type, size_bytes, width, height, alt_text, sha256, created_by)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`, in.StorageKey, in.OriginalName, in.MIMEType, in.SizeBytes, in.Width, in.Height, in.AltText, in.SHA256, in.CreatedBy)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	return uint64(id), err
}

func (s *Store) FindMediaAsset(ctx context.Context, id uint64) (*MediaAsset, error) {
	return scanMediaAsset(s.db.QueryRowContext(ctx, "SELECT "+mediaSelectFields+" FROM media_assets WHERE id = ?", id))
}

func (s *Store) FindMediaAssetBySHA256(ctx context.Context, hash string) (*MediaAsset, error) {
	return scanMediaAsset(s.db.QueryRowContext(ctx, "SELECT "+mediaSelectFields+" FROM media_assets WHERE sha256 = ?", hash))
}

func (s *Store) FindMediaAssetByStorageKey(ctx context.Context, storageKey string) (*MediaAsset, error) {
	return scanMediaAsset(s.db.QueryRowContext(ctx, "SELECT "+mediaSelectFields+" FROM media_assets WHERE storage_key = ?", storageKey))
}

func (s *Store) ListMediaAssets(ctx context.Context, page, size int, query string) ([]MediaAsset, int64, error) {
	offset := (page - 1) * size
	where := ""
	args := make([]interface{}, 0, 4)
	if query = strings.TrimSpace(query); query != "" {
		where = " WHERE original_name LIKE ? OR alt_text LIKE ?"
		like := "%" + query + "%"
		args = append(args, like, like)
	}
	var total int64
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM media_assets"+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	args = append(args, size, offset)
	rows, err := s.db.QueryContext(ctx, "SELECT "+mediaSelectFields+" FROM media_assets"+where+" ORDER BY created_at DESC, id DESC LIMIT ? OFFSET ?", args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := make([]MediaAsset, 0, size)
	for rows.Next() {
		item, err := scanMediaAsset(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, *item)
	}
	return items, total, rows.Err()
}

func (s *Store) UpdateMediaAlt(ctx context.Context, id uint64, altText string) error {
	res, err := s.db.ExecContext(ctx, "UPDATE media_assets SET alt_text = ? WHERE id = ?", altText, id)
	if err != nil {
		return err
	}
	return requireUpdateAffected(ctx, res, func(ctx context.Context) error {
		_, err := s.FindMediaAsset(ctx, id)
		return err
	})
}

func (s *Store) DeleteMediaAsset(ctx context.Context, id uint64) error {
	res, err := s.db.ExecContext(ctx, "DELETE FROM media_assets WHERE id = ?", id)
	if err != nil {
		return err
	}
	return requireAffected(res)
}

func (s *Store) MediaURLReferenced(ctx context.Context, mediaURL string) (bool, error) {
	like := "%" + mediaURL + "%"
	queries := []struct {
		query string
		args  []interface{}
	}{
		{"SELECT EXISTS(SELECT 1 FROM articles WHERE cover_url = ? OR content LIKE ?)", []interface{}{mediaURL, like}},
		{"SELECT EXISTS(SELECT 1 FROM article_versions INNER JOIN articles ON articles.id = article_versions.article_id WHERE article_versions.cover_url = ? OR article_versions.content LIKE ?)", []interface{}{mediaURL, like}},
		{"SELECT EXISTS(SELECT 1 FROM projects WHERE cover_url = ? OR content_markdown LIKE ?)", []interface{}{mediaURL, like}},
		{"SELECT EXISTS(SELECT 1 FROM users WHERE avatar_url = ?)", []interface{}{mediaURL}},
	}
	for _, item := range queries {
		var found int
		if err := s.db.QueryRowContext(ctx, item.query, item.args...).Scan(&found); err != nil {
			return false, err
		}
		if found > 0 {
			return true, nil
		}
	}
	return false, nil
}

func scanMediaAsset(row rowScanner) (*MediaAsset, error) {
	var item MediaAsset
	if err := row.Scan(&item.ID, &item.StorageKey, &item.OriginalName, &item.MIMEType, &item.SizeBytes, &item.Width, &item.Height, &item.AltText, &item.SHA256, &item.CreatedBy, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return nil, scanErr(err)
	}
	return &item, nil
}

var _ = sql.ErrNoRows
