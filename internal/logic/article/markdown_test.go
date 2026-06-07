package article

import (
	"strings"
	"testing"
)

func TestParseMarkdownImportFrontMatter(t *testing.T) {
	raw := `---
title: Front Matter Title
slug: custom-slug
summary: Short summary
category: Go
tags:
  - backend
  - redis
date: 2026-06-07
pinned: true
priority: 128
seo_keywords:
  - go
  - blog
---
# Ignored H1

Body content.
`

	doc, err := parseMarkdownImport("post.md", raw)
	if err != nil {
		t.Fatalf("parseMarkdownImport() error = %v", err)
	}
	if doc.Title != "Front Matter Title" || doc.Slug != "custom-slug" {
		t.Fatalf("title/slug = %q/%q", doc.Title, doc.Slug)
	}
	if doc.Category != "Go" || len(doc.Tags) != 2 || doc.Tags[0] != "backend" || doc.Tags[1] != "redis" {
		t.Fatalf("taxonomy = %q %#v", doc.Category, doc.Tags)
	}
	if doc.SEOKeywords != "go, blog" {
		t.Fatalf("SEOKeywords = %q", doc.SEOKeywords)
	}
	if !doc.IsPinned || doc.DisplayPriority != 128 {
		t.Fatalf("pin priority = %v/%d, want true/128", doc.IsPinned, doc.DisplayPriority)
	}
	if doc.ScheduledAt == nil {
		t.Fatal("ScheduledAt = nil")
	}
	if !strings.Contains(doc.Content, "# Ignored H1") {
		t.Fatalf("front matter title should not remove body h1, content = %q", doc.Content)
	}
}

func TestParseMarkdownImportUsesFirstH1(t *testing.T) {
	doc, err := parseMarkdownImport("fallback.md", "# Hello\n\nBody")
	if err != nil {
		t.Fatalf("parseMarkdownImport() error = %v", err)
	}
	if doc.Title != "Hello" {
		t.Fatalf("Title = %q, want Hello", doc.Title)
	}
	if strings.Contains(doc.Content, "# Hello") {
		t.Fatalf("Content should remove imported h1: %q", doc.Content)
	}
	if doc.Slug != "hello" {
		t.Fatalf("Slug = %q, want hello", doc.Slug)
	}
}

func TestSafeMarkdownFilename(t *testing.T) {
	if got := safeMarkdownFilename("Hello World!", "fallback"); got != "hello-world.md" {
		t.Fatalf("safeMarkdownFilename() = %q", got)
	}
}
