package model

import (
	"context"
	"database/sql"
	"encoding/json"
	"strconv"
	"strings"
)

type ResumeExperience struct {
	ID           uint64
	Role         string
	Organization string
	Location     string
	StartDate    string
	EndDate      string
	Description  string
	Highlights   []string
	DisplayOrder int
}

type ResumeEducation struct {
	ID           uint64
	School       string
	Degree       string
	Major        string
	Location     string
	StartDate    string
	EndDate      string
	Description  string
	Highlights   []string
	DisplayOrder int
}

type ResumeSkill struct {
	ID           uint64
	Category     string
	Name         string
	Level        int
	Description  string
	DisplayOrder int
}

func (s *Store) ListResumeExperiences(ctx context.Context) ([]ResumeExperience, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT id, role, organization, location, start_date, end_date, description, COALESCE(CAST(highlights AS CHAR), '[]'), display_order
FROM resume_experiences
ORDER BY display_order, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]ResumeExperience, 0)
	for rows.Next() {
		item, err := scanResumeExperience(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) ListResumeEducations(ctx context.Context) ([]ResumeEducation, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT id, school, degree, major, location, start_date, end_date, description, COALESCE(CAST(highlights AS CHAR), '[]'), display_order
FROM resume_educations
ORDER BY display_order, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]ResumeEducation, 0)
	for rows.Next() {
		item, err := scanResumeEducation(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) ListResumeSkills(ctx context.Context) ([]ResumeSkill, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT id, category, name, level, description, display_order
FROM resume_skills
ORDER BY display_order, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]ResumeSkill, 0)
	for rows.Next() {
		item, err := scanResumeSkill(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func scanResumeExperience(row rowScanner) (ResumeExperience, error) {
	var item ResumeExperience
	var description sql.NullString
	var highlightsRaw string
	if err := row.Scan(&item.ID, &item.Role, &item.Organization, &item.Location, &item.StartDate, &item.EndDate, &description, &highlightsRaw, &item.DisplayOrder); err != nil {
		return ResumeExperience{}, err
	}
	item.Description = stringFromNull(description)
	item.Highlights = decodeStringList(highlightsRaw)
	return item, nil
}

func scanResumeEducation(row rowScanner) (ResumeEducation, error) {
	var item ResumeEducation
	var description sql.NullString
	var highlightsRaw string
	if err := row.Scan(&item.ID, &item.School, &item.Degree, &item.Major, &item.Location, &item.StartDate, &item.EndDate, &description, &highlightsRaw, &item.DisplayOrder); err != nil {
		return ResumeEducation{}, err
	}
	item.Description = stringFromNull(description)
	item.Highlights = decodeStringList(highlightsRaw)
	return item, nil
}

func scanResumeSkill(row rowScanner) (ResumeSkill, error) {
	var item ResumeSkill
	var description sql.NullString
	if err := row.Scan(&item.ID, &item.Category, &item.Name, &item.Level, &description, &item.DisplayOrder); err != nil {
		return ResumeSkill{}, err
	}
	item.Description = stringFromNull(description)
	return item, nil
}

func replaceResumeExperiencesTx(ctx context.Context, tx *sql.Tx, items []ResumeExperience) error {
	if _, err := tx.ExecContext(ctx, "DELETE FROM resume_experiences"); err != nil {
		return err
	}
	for index, item := range items {
		highlights, err := encodeStringList(item.Highlights)
		if err != nil {
			return err
		}
		if item.DisplayOrder == 0 {
			item.DisplayOrder = index + 1
		}
		if _, err := tx.ExecContext(ctx, `
INSERT INTO resume_experiences (role, organization, location, start_date, end_date, description, highlights, display_order)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			item.Role, item.Organization, item.Location, item.StartDate, item.EndDate, item.Description, highlights, item.DisplayOrder); err != nil {
			return err
		}
	}
	return nil
}

func replaceResumeEducationsTx(ctx context.Context, tx *sql.Tx, items []ResumeEducation) error {
	if _, err := tx.ExecContext(ctx, "DELETE FROM resume_educations"); err != nil {
		return err
	}
	for index, item := range items {
		highlights, err := encodeStringList(item.Highlights)
		if err != nil {
			return err
		}
		if item.DisplayOrder == 0 {
			item.DisplayOrder = index + 1
		}
		if _, err := tx.ExecContext(ctx, `
INSERT INTO resume_educations (school, degree, major, location, start_date, end_date, description, highlights, display_order)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			item.School, item.Degree, item.Major, item.Location, item.StartDate, item.EndDate, item.Description, highlights, item.DisplayOrder); err != nil {
			return err
		}
	}
	return nil
}

func replaceResumeSkillsTx(ctx context.Context, tx *sql.Tx, items []ResumeSkill) error {
	if _, err := tx.ExecContext(ctx, "DELETE FROM resume_skills"); err != nil {
		return err
	}
	for index, item := range items {
		if item.DisplayOrder == 0 {
			item.DisplayOrder = index + 1
		}
		if _, err := tx.ExecContext(ctx, `
INSERT INTO resume_skills (category, name, level, description, display_order)
VALUES (?, ?, ?, ?, ?)`,
			item.Category, item.Name, item.Level, item.Description, item.DisplayOrder); err != nil {
			return err
		}
	}
	return nil
}

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
	for index, item := range items {
		res, err := tx.ExecContext(ctx, `
INSERT INTO projects (title, summary, role, period, cover_url, demo_url, repo_url, content_markdown, featured, display_order)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			item.Title, item.Summary, item.Role, item.Period, item.CoverURL, item.DemoURL, item.RepoURL, item.ContentMarkdown, item.Featured, index+1)
		if err != nil {
			return err
		}
		insertID, err := res.LastInsertId()
		if err != nil {
			return err
		}
		for _, tagID := range uniqueUint64(item.TagIDs) {
			if _, err := tx.ExecContext(ctx, "INSERT INTO project_tags (project_id, tag_id) VALUES (?, ?)", uint64(insertID), tagID); err != nil {
				return err
			}
		}
	}
	return nil
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

func encodeStringList(values []string) (string, error) {
	normalized := normalizeStringList(values)
	raw, err := json.Marshal(normalized)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func decodeStringList(raw string) []string {
	var values []string
	if err := json.Unmarshal([]byte(raw), &values); err != nil {
		return []string{}
	}
	return normalizeStringList(values)
}

func normalizeStringList(values []string) []string {
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			normalized = append(normalized, value)
		}
	}
	return normalized
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
