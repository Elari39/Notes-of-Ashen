package model

import (
	"context"
	"testing"
	"time"
)

// TestBuildArticleSearchDocument 验证批量重建路径共用的纯内存组装逻辑：
// tags 转换、分类名提取、visibleAt 取值优先级（scheduled > published > created）。
func TestBuildArticleSearchDocument(t *testing.T) {
	created := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	published := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	scheduled := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	category := Category{ID: 9, Name: "Go"}

	t.Run("tags and category assembled", func(t *testing.T) {
		item := Article{ID: 1, Title: "T", Summary: "S", Content: "C", Status: ArticleStatusPublished, CategoryID: 9, CreatedAt: created}
		tags := []Tag{{ID: 11, Name: "zero"}, {ID: 22, Name: "go"}}

		doc := buildArticleSearchDocument(item, tags, &category)

		if len(doc.Tags) != 2 || doc.Tags[0] != "zero" || doc.Tags[1] != "go" {
			t.Fatalf("tags = %#v, want [zero go]", doc.Tags)
		}
		if len(doc.TagIDs) != 2 || doc.TagIDs[0] != 11 || doc.TagIDs[1] != 22 {
			t.Fatalf("tagIDs = %#v, want [11 22]", doc.TagIDs)
		}
		if doc.Category != "Go" {
			t.Fatalf("category = %q, want Go", doc.Category)
		}
		if doc.VisibleAt != created.Unix() {
			t.Fatalf("visibleAt = %d, want created unix %d", doc.VisibleAt, created.Unix())
		}
		if doc.PublishedAt != 0 {
			t.Fatalf("publishedAt = %d, want 0 when PublishedAt nil", doc.PublishedAt)
		}
	})

	t.Run("publishedAt preferred over created", func(t *testing.T) {
		item := Article{ID: 2, CreatedAt: created, PublishedAt: &published}
		doc := buildArticleSearchDocument(item, nil, nil)
		if doc.VisibleAt != published.Unix() {
			t.Fatalf("visibleAt = %d, want published unix %d", doc.VisibleAt, published.Unix())
		}
		if doc.PublishedAt != published.Unix() {
			t.Fatalf("publishedAt = %d, want published unix %d", doc.PublishedAt, published.Unix())
		}
	})

	t.Run("scheduledAt preferred over published", func(t *testing.T) {
		item := Article{ID: 3, CreatedAt: created, PublishedAt: &published, ScheduledAt: &scheduled}
		doc := buildArticleSearchDocument(item, nil, nil)
		if doc.VisibleAt != scheduled.Unix() {
			t.Fatalf("visibleAt = %d, want scheduled unix %d", doc.VisibleAt, scheduled.Unix())
		}
	})

	t.Run("nil category yields empty name", func(t *testing.T) {
		item := Article{ID: 4, CategoryID: 99, CreatedAt: created}
		doc := buildArticleSearchDocument(item, nil, nil)
		if doc.Category != "" {
			t.Fatalf("category = %q, want empty", doc.Category)
		}
	})
}

// TestFindCategoriesByIDsEmptyNoQuery 与 TestArticleTagsBatchEmptyNoQuery 验证批量加载方法
// 在空入参时早返回空 map，不触碰数据库（db 为 nil 也不会 panic）。
func TestFindCategoriesByIDsEmptyNoQuery(t *testing.T) {
	s := &Store{}
	out, err := s.FindCategoriesByIDs(context.Background(), nil)
	if err != nil {
		t.Fatalf("err = %v, want nil", err)
	}
	if len(out) != 0 {
		t.Fatalf("len(out) = %d, want 0", len(out))
	}
}

func TestArticleTagsBatchEmptyNoQuery(t *testing.T) {
	s := &Store{}
	out, err := s.ArticleTagsBatch(context.Background(), nil)
	if err != nil {
		t.Fatalf("err = %v, want nil", err)
	}
	if len(out) != 0 {
		t.Fatalf("len(out) = %d, want 0", len(out))
	}
}
