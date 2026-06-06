package model

import (
	"strings"
	"testing"
)

func TestArticleWherePublicFilters(t *testing.T) {
	where, args := articleWhere(ArticleFilter{
		Query:      "go",
		CategoryID: 2,
		TagID:      3,
	})

	assertContains(t, where, "status = 'published'")
	assertContains(t, where, "MATCH(title, content) AGAINST(? IN NATURAL LANGUAGE MODE)")
	assertContains(t, where, "summary LIKE ? ESCAPE '!'")
	assertContains(t, where, "category_id = ?")
	assertContains(t, where, "EXISTS (SELECT 1 FROM article_tags at WHERE at.article_id = articles.id AND at.tag_id = ?)")

	wantArgs := []interface{}{"go", "%go%", "%go%", "%go%", uint64(2), uint64(3)}
	if len(args) != len(wantArgs) {
		t.Fatalf("args length = %d, want %d: %#v", len(args), len(wantArgs), args)
	}
	for i, want := range wantArgs {
		if args[i] != want {
			t.Fatalf("args[%d] = %#v, want %#v", i, args[i], want)
		}
	}
}

func TestArticleWhereAdminStatus(t *testing.T) {
	where, args := articleWhere(ArticleFilter{
		Role:   "admin",
		Status: "draft",
	})

	assertContains(t, where, "status = ?")
	if len(args) != 1 || args[0] != "draft" {
		t.Fatalf("args = %#v, want draft status arg", args)
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
