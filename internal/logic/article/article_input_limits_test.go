package article

import (
	"context"
	"strings"
	"testing"

	"notes-of-ashen/internal/svc"
	"notes-of-ashen/internal/types"
)

func TestValidateArticleRejectsOversizedUTF8ContentAndSummary(t *testing.T) {
	tests := []struct {
		name string
		req  types.ArticleReq
	}{
		{
			name: "content exceeds byte limit with Chinese characters",
			req:  types.ArticleReq{Title: "title", Slug: "slug", Status: "draft", Content: strings.Repeat("你", maxArticleContentBytes/3+1)},
		},
		{
			name: "summary exceeds byte limit",
			req:  types.ArticleReq{Title: "title", Slug: "slug", Status: "draft", Content: "content", Summary: strings.Repeat("a", maxArticleSummaryBytes+1)},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := validateArticle(context.Background(), &svc.ServiceContext{}, tt.req); err == nil {
				t.Fatal("validateArticle() error = nil, want size validation error")
			}
		})
	}
}

func TestValidateArticleAcceptsExactByteLimits(t *testing.T) {
	req := types.ArticleReq{
		Title:   "title",
		Slug:    "slug",
		Status:  "draft",
		Content: strings.Repeat("a", maxArticleContentBytes),
		Summary: strings.Repeat("b", maxArticleSummaryBytes),
	}
	if err := validateArticle(context.Background(), &svc.ServiceContext{}, req); err != nil {
		t.Fatalf("validateArticle() error = %v, want nil at exact limits", err)
	}
}
