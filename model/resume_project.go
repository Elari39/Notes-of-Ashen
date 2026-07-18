package model

import (
	"context"
	"database/sql"
	"strconv"
	"strings"
)

func (s *Store) ListProjectItems(ctx context.Context) ([]ProjectItem, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT id, title, summary, role, period, cover_url, demo_url, repo_url, COALESCE(content_markdown, ''), featured, display_order
FROM projects
ORDER BY featured DESC, display_order, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]ProjectItem, 0)
	projectIDs := make([]uint64, 0)
	for rows.Next() {
		item, numericID, err := scanProjectItem(rows)
		if err != nil {
			return nil, err
		}
		projectIDs = append(projectIDs, numericID)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	tagMap, tagIDMap, err := s.projectTags(ctx, projectIDs)
	if err != nil {
		return nil, err
	}
	for index, item := range items {
		numericID, _ := strconv.ParseUint(item.ID, 10, 64)
		item.Tags = nonNilStrings(tagMap[numericID])
		item.TagIDs = nonNilUint64s(tagIDMap[numericID])
		items[index] = item
	}
	return items, nil
}

func scanProjectItem(row rowScanner) (ProjectItem, uint64, error) {
	var numericID uint64
	var featured int
	var displayOrder int
	var item ProjectItem
	var summary sql.NullString
	var role sql.NullString
	var period sql.NullString
	var coverURL sql.NullString
	var demoURL sql.NullString
	var repoURL sql.NullString
	var contentMarkdown sql.NullString
	if err := row.Scan(&numericID, &item.Title, &summary, &role, &period, &coverURL, &demoURL, &repoURL, &contentMarkdown, &featured, &displayOrder); err != nil {
		return ProjectItem{}, 0, err
	}
	item.ID = strconv.FormatUint(numericID, 10)
	item.Summary = stringFromNull(summary)
	item.Role = stringFromNull(role)
	item.Period = stringFromNull(period)
	item.CoverURL = stringFromNull(coverURL)
	item.DemoURL = stringFromNull(demoURL)
	item.RepoURL = stringFromNull(repoURL)
	item.ContentMarkdown = stringFromNull(contentMarkdown)
	item.Featured = featured != 0
	return item, numericID, nil
}

func replaceProjectItemsTx(ctx context.Context, tx *sql.Tx, items []ProjectItem) error {
	if _, err := tx.ExecContext(ctx, "DELETE FROM project_tags"); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, "DELETE FROM projects"); err != nil {
		return err
	}
	if len(items) == 0 {
		return nil
	}
	projectIDs := make([]uint64, 0, len(items))
	for index, item := range items {
		res, err := tx.ExecContext(ctx, `
INSERT INTO projects (title, summary, role, period, cover_url, demo_url, repo_url, content_markdown, featured, display_order)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, item.Title, item.Summary, item.Role, item.Period,
			item.CoverURL, item.DemoURL, item.RepoURL, item.ContentMarkdown, item.Featured, index+1)
		if err != nil {
			return err
		}
		projectID, err := res.LastInsertId()
		if err != nil {
			return err
		}
		projectIDs = append(projectIDs, uint64(projectID))
	}
	// 批量插入 project_tags。
	var tagB strings.Builder
	tagB.WriteString("INSERT INTO project_tags (project_id, tag_id) VALUES ")
	tagArgs := make([]any, 0, len(items)*2)
	tagCount := 0
	for index, item := range items {
		projectID := projectIDs[index]
		for _, tagID := range uniqueUint64(item.TagIDs) {
			if tagCount > 0 {
				tagB.WriteByte(',')
			}
			tagB.WriteString("(?, ?)")
			tagArgs = append(tagArgs, projectID, tagID)
			tagCount++
		}
	}
	if tagCount == 0 {
		return nil
	}
	_, err := tx.ExecContext(ctx, tagB.String(), tagArgs...)
	return err
}

func hydrateProjectTagsTx(ctx context.Context, tx *sql.Tx, items []ProjectItem) ([]ProjectItem, error) {
	allIDs := make([]uint64, 0)
	for _, item := range items {
		allIDs = append(allIDs, item.TagIDs...)
	}
	allIDs = uniqueUint64(allIDs)
	if len(allIDs) == 0 {
		for index := range items {
			items[index].Tags = []string{}
			items[index].TagIDs = nonNilUint64s(items[index].TagIDs)
		}
		return items, nil
	}
	placeholders := strings.TrimRight(strings.Repeat("?,", len(allIDs)), ",")
	args := make([]any, 0, len(allIDs))
	for _, id := range allIDs {
		args = append(args, id)
	}
	rows, err := tx.QueryContext(ctx, "SELECT id, name FROM tags WHERE id IN ("+placeholders+") ORDER BY name", args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	names := make(map[uint64]string, len(allIDs))
	for rows.Next() {
		var id uint64
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			return nil, err
		}
		names[id] = name
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(names) != len(allIDs) {
		return nil, ErrNotFound
	}
	for index, item := range items {
		item.TagIDs = uniqueUint64(item.TagIDs)
		item.Tags = make([]string, 0, len(item.TagIDs))
		for _, id := range item.TagIDs {
			item.Tags = append(item.Tags, names[id])
		}
		items[index] = item
	}
	return items, nil
}

func (s *Store) projectTags(ctx context.Context, projectIDs []uint64) (map[uint64][]string, map[uint64][]uint64, error) {
	names := make(map[uint64][]string, len(projectIDs))
	ids := make(map[uint64][]uint64, len(projectIDs))
	if len(projectIDs) == 0 {
		return names, ids, nil
	}
	placeholders := strings.TrimRight(strings.Repeat("?,", len(projectIDs)), ",")
	args := make([]interface{}, 0, len(projectIDs))
	for _, id := range projectIDs {
		args = append(args, id)
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT pt.project_id, t.id, t.name
FROM project_tags pt
JOIN tags t ON t.id = pt.tag_id
WHERE pt.project_id IN (`+placeholders+`)
ORDER BY pt.project_id, t.name`, args...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var projectID uint64
		var tagID uint64
		var name string
		if err := rows.Scan(&projectID, &tagID, &name); err != nil {
			return nil, nil, err
		}
		names[projectID] = append(names[projectID], name)
		ids[projectID] = append(ids[projectID], tagID)
	}
	return names, ids, rows.Err()
}

func nonNilStrings(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}

func nonNilUint64s(values []uint64) []uint64 {
	if values == nil {
		return []uint64{}
	}
	return values
}
