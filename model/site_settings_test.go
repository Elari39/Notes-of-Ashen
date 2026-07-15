package model

import "testing"

func TestNormalizeHomeArticleLayout(t *testing.T) {
	tests := []struct {
		name   string
		layout string
		want   string
	}{
		{name: "standard", layout: HomeArticleLayoutStandard, want: HomeArticleLayoutStandard},
		{name: "alternating", layout: HomeArticleLayoutAlternating, want: HomeArticleLayoutAlternating},
		{name: "empty defaults to standard", layout: "", want: HomeArticleLayoutStandard},
		{name: "invalid defaults to standard", layout: "grid", want: HomeArticleLayoutStandard},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NormalizeHomeArticleLayout(tt.layout); got != tt.want {
				t.Fatalf("NormalizeHomeArticleLayout(%q) = %q, want %q", tt.layout, got, tt.want)
			}
		})
	}
}

func TestNormalizeAIAPIFormat(t *testing.T) {
	tests := []struct {
		value string
		want  string
	}{
		{value: "", want: AIAPIFormatOpenAI},
		{value: " OpenAI ", want: AIAPIFormatOpenAI},
		{value: "ANTHROPIC", want: AIAPIFormatAnthropic},
		{value: "unknown", want: AIAPIFormatOpenAI},
	}
	for _, tt := range tests {
		if got := NormalizeAIAPIFormat(tt.value); got != tt.want {
			t.Fatalf("NormalizeAIAPIFormat(%q) = %q, want %q", tt.value, got, tt.want)
		}
	}
}

func TestDecodeProjectItems(t *testing.T) {
	items, err := DecodeProjectItems(`[
		{
			"id": " first ",
			"title": " Project ",
			"summary": " Summary ",
			"tags": [" Go ", "go", "", "React"],
			"coverUrl": " https://example.com/cover.png "
		}
	]`)
	if err != nil {
		t.Fatalf("DecodeProjectItems returned error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
	item := items[0]
	if item.ID != "first" || item.Title != "Project" || item.Summary != "Summary" {
		t.Fatalf("project item was not normalized: %#v", item)
	}
	if len(item.Tags) != 2 || item.Tags[0] != "Go" || item.Tags[1] != "React" {
		t.Fatalf("tags = %#v, want [Go React]", item.Tags)
	}
	if item.CoverURL != "https://example.com/cover.png" {
		t.Fatalf("CoverURL = %q, want trimmed URL", item.CoverURL)
	}
}

func TestDecodeProjectItemsEmpty(t *testing.T) {
	items, err := DecodeProjectItems("")
	if err != nil {
		t.Fatalf("DecodeProjectItems returned error: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("len(items) = %d, want 0", len(items))
	}
}
