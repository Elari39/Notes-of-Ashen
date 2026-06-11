package model

import (
	"database/sql"
	"testing"
	"time"
)

type scanFunc func(dest ...interface{}) error

func (f scanFunc) Scan(dest ...interface{}) error {
	return f(dest...)
}

func TestScanTaxonomyAllowsNullDescription(t *testing.T) {
	now := time.Date(2026, 6, 11, 10, 0, 0, 0, time.UTC)
	scanner := scanFunc(func(dest ...interface{}) error {
		*dest[0].(*uint64) = 1
		*dest[1].(*string) = "Go"
		*dest[2].(*string) = "go"
		*dest[3].(*sql.NullString) = sql.NullString{}
		*dest[4].(*uint64) = 2
		*dest[5].(*time.Time) = now
		*dest[6].(*time.Time) = now
		return nil
	})

	category, err := scanCategory(scanner)
	if err != nil {
		t.Fatalf("scanCategory() error = %v, want nil", err)
	}
	if category.Description != "" {
		t.Fatalf("category.Description = %q, want empty string", category.Description)
	}

	tag, err := scanTag(scanner)
	if err != nil {
		t.Fatalf("scanTag() error = %v, want nil", err)
	}
	if tag.Description != "" {
		t.Fatalf("tag.Description = %q, want empty string", tag.Description)
	}
}

func TestScanArticleAllowsNullableTextFields(t *testing.T) {
	now := time.Date(2026, 6, 11, 10, 0, 0, 0, time.UTC)
	article, err := scanArticle(scanFunc(func(dest ...interface{}) error {
		*dest[0].(*uint64) = 10
		*dest[1].(*uint64) = 2
		*dest[2].(*sql.NullInt64) = sql.NullInt64{}
		*dest[3].(*string) = "Title"
		*dest[4].(*string) = "title"
		*dest[5].(*sql.NullString) = sql.NullString{}
		*dest[6].(*string) = "content"
		*dest[7].(*sql.NullString) = sql.NullString{}
		*dest[8].(*string) = ArticleStatusPublished
		*dest[9].(*uint64) = 3
		*dest[10].(*uint64) = 4
		*dest[11].(*sql.NullTime) = sql.NullTime{}
		*dest[12].(*sql.NullTime) = sql.NullTime{}
		*dest[13].(*int) = 0
		*dest[14].(*int) = 0
		*dest[15].(*sql.NullString) = sql.NullString{}
		*dest[16].(*sql.NullString) = sql.NullString{}
		*dest[17].(*sql.NullString) = sql.NullString{}
		*dest[18].(*time.Time) = now
		*dest[19].(*time.Time) = now
		return nil
	}))
	if err != nil {
		t.Fatalf("scanArticle() error = %v, want nil", err)
	}
	if article.Summary != "" || article.CoverURL != "" || article.SEOTitle != "" || article.SEODescription != "" || article.SEOKeywords != "" {
		t.Fatalf("nullable article strings = %#v", article)
	}
}

func TestScanArticleVersionAllowsNullableTextFields(t *testing.T) {
	now := time.Date(2026, 6, 11, 10, 0, 0, 0, time.UTC)
	version, err := scanArticleVersion(scanFunc(func(dest ...interface{}) error {
		*dest[0].(*uint64) = 1
		*dest[1].(*uint64) = 10
		*dest[2].(*int) = 2
		*dest[3].(*uint64) = 3
		*dest[4].(*uint64) = 4
		*dest[5].(*sql.NullInt64) = sql.NullInt64{}
		*dest[6].(*string) = "Title"
		*dest[7].(*string) = "title"
		*dest[8].(*sql.NullString) = sql.NullString{}
		*dest[9].(*string) = "content"
		*dest[10].(*sql.NullString) = sql.NullString{}
		*dest[11].(*string) = ArticleStatusDraft
		*dest[12].(*uint64) = 0
		*dest[13].(*uint64) = 0
		*dest[14].(*sql.NullTime) = sql.NullTime{}
		*dest[15].(*sql.NullTime) = sql.NullTime{}
		*dest[16].(*int) = 0
		*dest[17].(*int) = 0
		*dest[18].(*sql.NullString) = sql.NullString{}
		*dest[19].(*sql.NullString) = sql.NullString{}
		*dest[20].(*sql.NullString) = sql.NullString{}
		*dest[21].(*string) = "[]"
		*dest[22].(*sql.NullTime) = sql.NullTime{}
		*dest[23].(*sql.NullTime) = sql.NullTime{}
		*dest[24].(*time.Time) = now
		return nil
	}))
	if err != nil {
		t.Fatalf("scanArticleVersion() error = %v, want nil", err)
	}
	if version.Summary != "" || version.CoverURL != "" || version.SEOTitle != "" || version.SEODescription != "" || version.SEOKeywords != "" {
		t.Fatalf("nullable article version strings = %#v", version)
	}
}

func TestScanResumeAndProjectAllowsNullableTextFields(t *testing.T) {
	experience, err := scanResumeExperience(scanFunc(func(dest ...interface{}) error {
		*dest[0].(*uint64) = 1
		*dest[1].(*string) = "Developer"
		*dest[2].(*string) = "Org"
		*dest[3].(*string) = ""
		*dest[4].(*string) = ""
		*dest[5].(*string) = ""
		*dest[6].(*sql.NullString) = sql.NullString{}
		*dest[7].(*string) = "[]"
		*dest[8].(*int) = 1
		return nil
	}))
	if err != nil {
		t.Fatalf("scanResumeExperience() error = %v, want nil", err)
	}
	if experience.Description != "" {
		t.Fatalf("experience.Description = %q, want empty string", experience.Description)
	}

	education, err := scanResumeEducation(scanFunc(func(dest ...interface{}) error {
		*dest[0].(*uint64) = 1
		*dest[1].(*string) = "School"
		*dest[2].(*string) = ""
		*dest[3].(*string) = ""
		*dest[4].(*string) = ""
		*dest[5].(*string) = ""
		*dest[6].(*string) = ""
		*dest[7].(*sql.NullString) = sql.NullString{}
		*dest[8].(*string) = "[]"
		*dest[9].(*int) = 1
		return nil
	}))
	if err != nil {
		t.Fatalf("scanResumeEducation() error = %v, want nil", err)
	}
	if education.Description != "" {
		t.Fatalf("education.Description = %q, want empty string", education.Description)
	}

	skill, err := scanResumeSkill(scanFunc(func(dest ...interface{}) error {
		*dest[0].(*uint64) = 1
		*dest[1].(*string) = "Backend"
		*dest[2].(*string) = "Go"
		*dest[3].(*int) = 80
		*dest[4].(*sql.NullString) = sql.NullString{}
		*dest[5].(*int) = 1
		return nil
	}))
	if err != nil {
		t.Fatalf("scanResumeSkill() error = %v, want nil", err)
	}
	if skill.Description != "" {
		t.Fatalf("skill.Description = %q, want empty string", skill.Description)
	}

	project, numericID, err := scanProjectItem(scanFunc(func(dest ...interface{}) error {
		*dest[0].(*uint64) = 7
		*dest[1].(*string) = "Project"
		*dest[2].(*sql.NullString) = sql.NullString{}
		*dest[3].(*sql.NullString) = sql.NullString{}
		*dest[4].(*sql.NullString) = sql.NullString{}
		*dest[5].(*sql.NullString) = sql.NullString{}
		*dest[6].(*sql.NullString) = sql.NullString{}
		*dest[7].(*sql.NullString) = sql.NullString{}
		*dest[8].(*sql.NullString) = sql.NullString{}
		*dest[9].(*int) = 0
		*dest[10].(*int) = 1
		return nil
	}))
	if err != nil {
		t.Fatalf("scanProjectItem() error = %v, want nil", err)
	}
	if numericID != 7 || project.ID != "7" {
		t.Fatalf("project id = %d/%q, want 7/%q", numericID, project.ID, "7")
	}
	if project.Summary != "" || project.Role != "" || project.Period != "" || project.CoverURL != "" || project.DemoURL != "" || project.RepoURL != "" || project.ContentMarkdown != "" {
		t.Fatalf("nullable project strings = %#v", project)
	}
}
