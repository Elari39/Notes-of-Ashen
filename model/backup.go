package model

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"
)

// BackupRestoreMarkerKey is written in the same transaction as a restored
// snapshot. It lets startup complete the corresponding media directory switch
// after a process interruption without exposing the internal marker in exports.
const BackupRestoreMarkerKey = "__backup_restore_marker"

type BackupSetting struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type BackupArticle struct {
	Article Article  `json:"article"`
	TagIDs  []uint64 `json:"tagIds"`
}

type BackupSnapshot struct {
	Users       []User           `json:"users"`
	Settings    []BackupSetting  `json:"settings"`
	Categories  []Category       `json:"categories"`
	Tags        []Tag            `json:"tags"`
	Projects    []ProjectItem    `json:"projects"`
	Articles    []BackupArticle  `json:"articles"`
	Versions    []ArticleVersion `json:"versions"`
	MediaAssets []MediaAsset     `json:"mediaAssets"`
}

func (s *Store) ExportBackup(ctx context.Context) (*BackupSnapshot, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true, Isolation: sql.LevelRepeatableRead})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	snapshot := &BackupSnapshot{}
	if snapshot.Users, err = backupUsers(ctx, tx); err != nil {
		return nil, err
	}
	if snapshot.Settings, err = backupSettings(ctx, tx); err != nil {
		return nil, err
	}
	if snapshot.Categories, err = backupCategories(ctx, tx); err != nil {
		return nil, err
	}
	if snapshot.Tags, err = backupTags(ctx, tx); err != nil {
		return nil, err
	}
	if snapshot.Projects, err = backupProjects(ctx, tx); err != nil {
		return nil, err
	}
	if snapshot.Articles, err = backupArticles(ctx, tx); err != nil {
		return nil, err
	}
	if snapshot.Versions, err = backupVersions(ctx, tx); err != nil {
		return nil, err
	}
	if snapshot.MediaAssets, err = backupMedia(ctx, tx); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return snapshot, nil
}

func backupUsers(ctx context.Context, tx *sql.Tx) ([]User, error) {
	rows, err := tx.QueryContext(ctx, `SELECT id, account, password_hash, email, avatar_url, nickname, role, status, created_at, updated_at FROM users ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]User, 0)
	for rows.Next() {
		var item User
		if err := rows.Scan(&item.ID, &item.Account, &item.PasswordHash, &item.Email, &item.AvatarURL, &item.Nickname, &item.Role, &item.Status, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func backupSettings(ctx context.Context, tx *sql.Tx) ([]BackupSetting, error) {
	rows, err := tx.QueryContext(ctx, `SELECT setting_key, setting_value FROM site_settings WHERE setting_key NOT IN (?, ?) ORDER BY setting_key`, "ai_api_key_cipher", BackupRestoreMarkerKey)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]BackupSetting, 0)
	for rows.Next() {
		var item BackupSetting
		if err := rows.Scan(&item.Key, &item.Value); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func backupCategories(ctx context.Context, tx *sql.Tx) ([]Category, error) {
	rows, err := tx.QueryContext(ctx, `SELECT id,name,slug,description,created_by,created_at,updated_at FROM categories ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]Category, 0)
	for rows.Next() {
		item, err := scanCategory(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	return items, rows.Err()
}
func backupTags(ctx context.Context, tx *sql.Tx) ([]Tag, error) {
	rows, err := tx.QueryContext(ctx, `SELECT id,name,slug,description,created_by,created_at,updated_at FROM tags ORDER BY id`)
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

func backupProjects(ctx context.Context, tx *sql.Tx) ([]ProjectItem, error) {
	rows, err := tx.QueryContext(ctx, `SELECT id,title,summary,role,period,cover_url,demo_url,repo_url,COALESCE(content_markdown,''),featured,display_order FROM projects ORDER BY display_order,id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]ProjectItem, 0)
	ids := make([]uint64, 0)
	for rows.Next() {
		item, id, err := scanProjectItem(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	tagIDs := map[uint64][]uint64{}
	tagNames := map[uint64][]string{}
	tagRows, err := tx.QueryContext(ctx, `SELECT pt.project_id,t.id,t.name FROM project_tags pt JOIN tags t ON t.id=pt.tag_id ORDER BY pt.project_id,t.name`)
	if err != nil {
		return nil, err
	}
	defer tagRows.Close()
	for tagRows.Next() {
		var pid, tid uint64
		var name string
		if err := tagRows.Scan(&pid, &tid, &name); err != nil {
			return nil, err
		}
		tagIDs[pid] = append(tagIDs[pid], tid)
		tagNames[pid] = append(tagNames[pid], name)
	}
	for i, id := range ids {
		items[i].TagIDs = nonNilUint64s(tagIDs[id])
		items[i].Tags = nonNilStrings(tagNames[id])
	}
	return items, tagRows.Err()
}

func backupArticles(ctx context.Context, tx *sql.Tx) ([]BackupArticle, error) {
	rows, err := tx.QueryContext(ctx, "SELECT "+articleSelectFields+" FROM articles ORDER BY id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]BackupArticle, 0)
	for rows.Next() {
		item, err := scanArticleRows(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, BackupArticle{Article: *item, TagIDs: []uint64{}})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	byID := map[uint64][]uint64{}
	tagRows, err := tx.QueryContext(ctx, `SELECT article_id,tag_id FROM article_tags ORDER BY article_id,tag_id`)
	if err != nil {
		return nil, err
	}
	defer tagRows.Close()
	for tagRows.Next() {
		var aid, tid uint64
		if err := tagRows.Scan(&aid, &tid); err != nil {
			return nil, err
		}
		byID[aid] = append(byID[aid], tid)
	}
	for i := range items {
		items[i].TagIDs = nonNilUint64s(byID[items[i].Article.ID])
	}
	return items, tagRows.Err()
}

func backupVersions(ctx context.Context, tx *sql.Tx) ([]ArticleVersion, error) {
	rows, err := tx.QueryContext(ctx, "SELECT "+articleVersionSelectFields+" FROM article_versions ORDER BY article_id,version_no")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]ArticleVersion, 0)
	for rows.Next() {
		item, err := scanArticleVersionRows(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	return items, rows.Err()
}
func backupMedia(ctx context.Context, tx *sql.Tx) ([]MediaAsset, error) {
	rows, err := tx.QueryContext(ctx, "SELECT "+mediaSelectFields+" FROM media_assets ORDER BY id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]MediaAsset, 0)
	for rows.Next() {
		item, err := scanMediaAsset(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	return items, rows.Err()
}

func (s *Store) RestoreBackup(ctx context.Context, snapshot BackupSnapshot) error {
	return s.restoreBackup(ctx, snapshot, "")
}

// RestoreBackupWithMarker replaces the snapshot and records the media restore
// transaction ID in the same SQL transaction. Callers must only clear the
// marker after the staged media directory has been published.
func (s *Store) RestoreBackupWithMarker(ctx context.Context, snapshot BackupSnapshot, restoreMarker string) error {
	if restoreMarker == "" {
		return errors.New("backup restore marker is empty")
	}
	return s.restoreBackup(ctx, snapshot, restoreMarker)
}

func (s *Store) restoreBackup(ctx context.Context, snapshot BackupSnapshot, restoreMarker string) error {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return err
	}
	defer tx.Rollback()
	deletes := []string{"DELETE FROM refresh_tokens", "DELETE FROM operation_logs", "DELETE FROM traffic_content_daily_visitors", "DELETE FROM traffic_content_daily_stats", "DELETE FROM traffic_referer_stats", "DELETE FROM traffic_daily_visitors", "DELETE FROM traffic_daily_stats", "DELETE FROM article_likes", "DELETE FROM project_tags", "DELETE FROM article_tags", "DELETE FROM article_versions", "DELETE FROM projects", "DELETE FROM media_assets", "DELETE FROM articles", "DELETE FROM categories", "DELETE FROM tags", "DELETE FROM site_settings", "DELETE FROM users"}
	for _, query := range deletes {
		if _, err := tx.ExecContext(ctx, query); err != nil {
			return err
		}
	}
	for _, u := range snapshot.Users {
		if _, err := tx.ExecContext(ctx, `INSERT INTO users(id,account,password_hash,email,avatar_url,nickname,role,status,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?,?)`, u.ID, u.Account, u.PasswordHash, u.Email, u.AvatarURL, u.Nickname, u.Role, u.Status, u.CreatedAt, u.UpdatedAt); err != nil {
			return err
		}
	}
	for _, c := range snapshot.Categories {
		if _, err := tx.ExecContext(ctx, `INSERT INTO categories(id,name,slug,description,created_by,created_at,updated_at) VALUES(?,?,?,?,?,?,?)`, c.ID, c.Name, c.Slug, c.Description, c.CreatedBy, c.CreatedAt, c.UpdatedAt); err != nil {
			return err
		}
	}
	for _, t := range snapshot.Tags {
		if _, err := tx.ExecContext(ctx, `INSERT INTO tags(id,name,slug,description,created_by,created_at,updated_at) VALUES(?,?,?,?,?,?,?)`, t.ID, t.Name, t.Slug, t.Description, t.CreatedBy, t.CreatedAt, t.UpdatedAt); err != nil {
			return err
		}
	}
	for index, p := range snapshot.Projects {
		id, err := strconv.ParseUint(p.ID, 10, 64)
		if err != nil || id == 0 {
			return ErrNotFound
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO projects(id,title,summary,role,period,cover_url,demo_url,repo_url,content_markdown,featured,display_order) VALUES(?,?,?,?,?,?,?,?,?,?,?)`, id, p.Title, p.Summary, p.Role, p.Period, p.CoverURL, p.DemoURL, p.RepoURL, p.ContentMarkdown, p.Featured, index+1); err != nil {
			return err
		}
		for _, tagID := range uniqueUint64(p.TagIDs) {
			if _, err := tx.ExecContext(ctx, `INSERT INTO project_tags(project_id,tag_id) VALUES(?,?)`, id, tagID); err != nil {
				return err
			}
		}
	}
	for _, entry := range snapshot.Articles {
		a := entry.Article
		if _, err := tx.ExecContext(ctx, `INSERT INTO articles(id,author_id,category_id,title,slug,summary,content,cover_url,status,view_count,like_count,scheduled_at,published_at,is_pinned,display_priority,seo_title,seo_description,seo_keywords,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, a.ID, a.AuthorID, nullableUint64(a.CategoryID), a.Title, a.Slug, a.Summary, a.Content, a.CoverURL, a.Status, a.ViewCount, a.LikeCount, a.ScheduledAt, a.PublishedAt, a.IsPinned, a.DisplayPriority, a.SEOTitle, a.SEODescription, a.SEOKeywords, a.CreatedAt, a.UpdatedAt); err != nil {
			return err
		}
		for _, tagID := range uniqueUint64(entry.TagIDs) {
			if _, err := tx.ExecContext(ctx, `INSERT INTO article_tags(article_id,tag_id) VALUES(?,?)`, a.ID, tagID); err != nil {
				return err
			}
		}
	}
	for _, v := range snapshot.Versions {
		tagJSON, err := json.Marshal(nonNilUint64s(v.TagIDs))
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO article_versions(id,article_id,version_no,changed_by,author_id,category_id,title,slug,summary,content,cover_url,status,view_count,like_count,scheduled_at,published_at,is_pinned,display_priority,seo_title,seo_description,seo_keywords,tag_ids,original_created_at,original_updated_at,created_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, v.ID, v.ArticleID, v.VersionNo, v.ChangedBy, v.AuthorID, nullableUint64(v.CategoryID), v.Title, v.Slug, v.Summary, v.Content, v.CoverURL, v.Status, v.ViewCount, v.LikeCount, v.ScheduledAt, v.PublishedAt, v.IsPinned, v.DisplayPriority, v.SEOTitle, v.SEODescription, v.SEOKeywords, string(tagJSON), v.OriginalCreatedAt, v.OriginalUpdatedAt, v.CreatedAt); err != nil {
			return err
		}
	}
	for _, m := range snapshot.MediaAssets {
		if _, err := tx.ExecContext(ctx, `INSERT INTO media_assets(id,storage_key,original_name,mime_type,size_bytes,width,height,alt_text,sha256,created_by,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?)`, m.ID, m.StorageKey, m.OriginalName, m.MIMEType, m.SizeBytes, m.Width, m.Height, m.AltText, m.SHA256, m.CreatedBy, m.CreatedAt, m.UpdatedAt); err != nil {
			return err
		}
	}
	settings := make(map[string]string, len(snapshot.Settings)+2)
	for _, item := range snapshot.Settings {
		if item.Key != "ai_api_key_cipher" && item.Key != BackupRestoreMarkerKey {
			settings[item.Key] = item.Value
		}
	}
	settings["ai_enabled"] = "false"
	settings["ai_api_key_cipher"] = ""
	now := time.Now()
	for key, value := range settings {
		if _, err := tx.ExecContext(ctx, `INSERT INTO site_settings(setting_key,setting_value,created_at,updated_at) VALUES(?,?,?,?)`, key, value, now, now); err != nil {
			return err
		}
	}
	if restoreMarker != "" {
		if err := insertBackupRestoreMarker(ctx, tx, restoreMarker, now); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func insertBackupRestoreMarker(ctx context.Context, tx *sql.Tx, marker string, now time.Time) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO site_settings(setting_key,setting_value,created_at,updated_at) VALUES(?,?,?,?)`, BackupRestoreMarkerKey, marker, now, now)
	return err
}

// BackupRestoreMarker returns the marker left by a committed restore whose
// media publication or cleanup has not finished yet.
func (s *Store) BackupRestoreMarker(ctx context.Context) (string, error) {
	var marker string
	err := s.db.QueryRowContext(ctx, `SELECT setting_value FROM site_settings WHERE setting_key = ? LIMIT 1`, BackupRestoreMarkerKey).Scan(&marker)
	if errors.Is(scanErr(err), ErrNotFound) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	if marker == "" {
		return "", fmt.Errorf("backup restore marker is empty")
	}
	return marker, nil
}

// ClearBackupRestoreMarker only clears the marker written by the matching
// restore. A mismatch means recovery must stop rather than finalize a different
// restore transaction.
func (s *Store) ClearBackupRestoreMarker(ctx context.Context, marker string) error {
	result, err := s.db.ExecContext(ctx, `DELETE FROM site_settings WHERE setting_key = ? AND setting_value = ?`, BackupRestoreMarkerKey, marker)
	if err != nil {
		return err
	}
	updated, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if updated != 1 {
		return fmt.Errorf("backup restore marker changed")
	}
	return nil
}
