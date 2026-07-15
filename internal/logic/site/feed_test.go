package site

import (
	"strconv"
	"testing"
	"time"

	"notes-of-ashen/model"
)

func TestSitemapURLsIncludesMoreThanOneHundredArticles(t *testing.T) {
	updatedAt := time.Date(2026, 7, 15, 0, 0, 0, 0, time.UTC)
	articles := make([]model.PublicArticleEntry, 101)
	for i := range articles {
		articles[i] = model.PublicArticleEntry{ID: uint64(i + 1), UpdatedAt: updatedAt}
	}

	urls := sitemapURLs("https://example.com", true, articles)
	if len(urls) != 105 {
		t.Fatalf("len(urls) = %d, want 105", len(urls))
	}
	last := urls[len(urls)-1]
	if last.Loc != "https://example.com/article/101" || last.LastMod != "2026-07-15" {
		t.Fatalf("last URL = %#v", last)
	}
}

func TestSitemapURLsRespectsSingleFileLimit(t *testing.T) {
	articles := make([]model.PublicArticleEntry, maxSitemapURLs)
	for i := range articles {
		articles[i].ID = uint64(i + 1)
	}

	urls := sitemapURLs("https://example.com", false, articles)
	if len(urls) != maxSitemapURLs {
		t.Fatalf("len(urls) = %d, want %d", len(urls), maxSitemapURLs)
	}
	wantLast := "https://example.com/article/" + strconv.Itoa(maxSitemapURLs-baseSitemapURLs)
	if urls[len(urls)-1].Loc != wantLast {
		t.Fatalf("last URL = %q, want %q", urls[len(urls)-1].Loc, wantLast)
	}
}
