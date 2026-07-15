package model

import (
	"context"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestListPublicArticleEntriesUsesLightweightFields(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock: %v", err)
	}
	defer db.Close()

	now := time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, title, summary, published_at, created_at, updated_at FROM articles WHERE status = 'published' AND (scheduled_at IS NULL OR scheduled_at <= NOW()) ORDER BY " + articleTimeOrder + " LIMIT ?")).
		WithArgs(50).
		WillReturnRows(sqlmock.NewRows([]string{"id", "title", "summary", "published_at", "created_at", "updated_at"}).
			AddRow(1, "Title", "Summary", now, now.Add(-time.Hour), now))

	items, err := NewStore(db).ListPublicArticleEntries(context.Background(), 50)
	if err != nil {
		t.Fatalf("ListPublicArticleEntries() error = %v", err)
	}
	if len(items) != 1 || items[0].Title != "Title" || items[0].PublishedAt == nil {
		t.Fatalf("items = %#v", items)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}

func TestArticleWherePublicFilters(t *testing.T) {
	where, args := articleWhere(ArticleFilter{
		Query:      "go",
		CategoryID: 2,
		TagID:      3,
	}, queryFulltext)

	assertContains(t, where, "status = 'published'")
	assertContains(t, where, "(scheduled_at IS NULL OR scheduled_at <= NOW())")
	assertContains(t, where, "MATCH(title, content) AGAINST(? IN NATURAL LANGUAGE MODE)")
	assertContains(t, where, "category_id = ?")
	assertContains(t, where, "EXISTS (SELECT 1 FROM article_tags at WHERE at.article_id = articles.id AND at.tag_id = ?)")

	wantArgs := []interface{}{"go", uint64(2), uint64(3)}
	if len(args) != len(wantArgs) {
		t.Fatalf("args length = %d, want %d: %#v", len(args), len(wantArgs), args)
	}
	for i, want := range wantArgs {
		if args[i] != want {
			t.Fatalf("args[%d] = %#v, want %#v", i, args[i], want)
		}
	}
}

func TestArticleWhereFulltextMissFallsBackToTitleLike(t *testing.T) {
	where, args := articleWhere(ArticleFilter{Query: "go"}, queryLike)

	assertContains(t, where, "title LIKE ? ESCAPE '!'")
	if strings.Contains(where, "MATCH(") {
		t.Fatalf("like mode should not include MATCH clause: %s", where)
	}
	wantArgs := []interface{}{"%go%"}
	if len(args) != len(wantArgs) || args[0] != wantArgs[0] {
		t.Fatalf("args = %#v, want %#v", args, wantArgs)
	}
}

func TestArticleWhereAdminStatus(t *testing.T) {
	where, args := articleWhere(ArticleFilter{
		Role:   "admin",
		Status: "draft",
	}, queryNone)

	assertContains(t, where, "status = ?")
	if len(args) != 1 || args[0] != "draft" {
		t.Fatalf("args = %#v, want draft status arg", args)
	}
}

func TestArticleWhereContentRoleScheduledStatus(t *testing.T) {
	where, args := articleWhere(ArticleFilter{
		Role:   "editor",
		Status: ArticleStatusScheduled,
	}, queryNone)

	assertContains(t, where, "status = 'published' AND scheduled_at > NOW()")
	if len(args) != 0 {
		t.Fatalf("args = %#v, want empty args", args)
	}
}

func TestArticleDisplayOrder(t *testing.T) {
	assertContains(t, articleDisplayOrder, "is_pinned DESC")
	assertContains(t, articleDisplayOrder, "display_priority DESC")
	assertContains(t, articleDisplayOrder, "COALESCE(published_at, created_at) DESC")
	assertContains(t, articleDisplayOrder, "id DESC")
}

func TestArticleVersionSelectFieldsIncludeDisplayFields(t *testing.T) {
	assertContains(t, articleVersionSelectFields, "is_pinned")
	assertContains(t, articleVersionSelectFields, "display_priority")
}

func TestIsArticlePubliclyVisible(t *testing.T) {
	now := time.Date(2026, 6, 6, 12, 0, 0, 0, time.UTC)
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)

	tests := []struct {
		name string
		item Article
		want bool
	}{
		{
			name: "published without schedule is visible",
			item: Article{Status: ArticleStatusPublished},
			want: true,
		},
		{
			name: "draft is hidden",
			item: Article{Status: ArticleStatusDraft},
			want: false,
		},
		{
			name: "future scheduled published article is hidden",
			item: Article{Status: ArticleStatusPublished, ScheduledAt: &future},
			want: false,
		},
		{
			name: "past scheduled published article is visible",
			item: Article{Status: ArticleStatusPublished, ScheduledAt: &past},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsArticlePubliclyVisible(tt.item, now); got != tt.want {
				t.Fatalf("IsArticlePubliclyVisible() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEscapeLike(t *testing.T) {
	got := escapeLike("100%_! done")
	want := "100!%!_!! done"
	if got != want {
		t.Fatalf("escapeLike() = %q, want %q", got, want)
	}
}

func assertContains(t *testing.T, value, want string) {
	t.Helper()
	if !strings.Contains(value, want) {
		t.Fatalf("%q does not contain %q", value, want)
	}
}
