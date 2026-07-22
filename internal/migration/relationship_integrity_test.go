package migration

import (
	"io/fs"
	"strings"
	"testing"

	"notes-of-ashen/deploy/mysql/migrations"
)

func TestRelationshipIntegrityMigrationContainsDeletePolicies(t *testing.T) {
	sqlText, err := fs.ReadFile(migrations.FS, "000024_add_relationship_integrity.sql")
	if err != nil {
		t.Fatalf("read relationship integrity migration: %v", err)
	}
	sql := strings.ToLower(string(sqlText))
	for _, fragment := range []string{
		"fk_articles_category",
		"fk_article_tags_article",
		"fk_refresh_tokens_user",
		"fk_operation_logs_user",
		"fk_media_assets_creator",
		"on delete cascade",
		"on delete set null",
		"on delete restrict",
		"delete article_tags",
		"delete project_tags",
	} {
		if !strings.Contains(sql, fragment) {
			t.Fatalf("relationship integrity migration missing %q", fragment)
		}
	}
}
