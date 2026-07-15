package article

import (
	"strings"
	"testing"

	"notes-of-ashen/internal/aiclient"
)

func TestAIActionsIncludeWritingActions(t *testing.T) {
	for _, action := range []string{"complete", "metadata", "proofread", "polish", "expand", "shorten", "translate"} {
		t.Run(action, func(t *testing.T) {
			if _, ok := aiActions[action]; !ok {
				t.Fatalf("aiActions missing %q", action)
			}
		})
	}
}

func TestNormalizeGeneratedSlug(t *testing.T) {
	tests := map[string]string{
		" Go 1.25 / AI Assist ": "go-1-25-ai-assist",
		"already--clean":        "already-clean",
		"中文标题":                  "",
		"":                      "",
	}
	for input, want := range tests {
		if got := normalizeGeneratedSlug(input, 180); got != want {
			t.Fatalf("normalizeGeneratedSlug(%q) = %q, want %q", input, got, want)
		}
	}
	if got := normalizeGeneratedSlug("long-slug", 5); got != "long" {
		t.Fatalf("limited slug = %q, want long", got)
	}
}

func TestNormalizeAISuggestions(t *testing.T) {
	got := normalizeAISuggestions([]string{" Go ", "go", "", "后端开发", "AI", "测试", "忽略"}, 4, 4)
	want := []string{"Go", "后端开发", "AI", "测试"}
	if len(got) != len(want) {
		t.Fatalf("suggestions = %#v, want %#v", got, want)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("suggestions = %#v, want %#v", got, want)
		}
	}
}

func TestNormalizeAIAssistResponseTrimsCompletionFields(t *testing.T) {
	response := normalizeAIAssistResponse(&aiclient.Response{
		Title:              strings.Repeat("题", 170),
		Slug:               " Go / AI ",
		Summary:            strings.Repeat("摘", 510),
		SEOTitle:           strings.Repeat("搜", 170),
		SEODescription:     strings.Repeat("描", 260),
		SEOKeywords:        strings.Repeat("词", 260),
		CategorySuggestion: strings.Repeat("分", 70),
		TagSuggestions:     []string{" Go ", "go", "AI"},
	})
	for name, value := range map[string]struct {
		text string
		max  int
	}{
		"title": {response.Title, 160}, "summary": {response.Summary, 500},
		"seoTitle": {response.SEOTitle, 160}, "seoDescription": {response.SEODescription, 255},
		"seoKeywords": {response.SEOKeywords, 255}, "category": {response.CategorySuggestion, 64},
	} {
		if got := len([]rune(value.text)); got != value.max {
			t.Fatalf("%s length = %d, want %d", name, got, value.max)
		}
	}
	if response.Slug != "go-ai" {
		t.Fatalf("slug = %q, want go-ai", response.Slug)
	}
	if len(response.TagSuggestions) != 2 || response.TagSuggestions[0] != "Go" || response.TagSuggestions[1] != "AI" {
		t.Fatalf("tag suggestions = %#v", response.TagSuggestions)
	}
}

func TestHasAICompletionRequiresUsableField(t *testing.T) {
	if hasAICompletion(normalizeAIAssistResponse(&aiclient.Response{})) {
		t.Fatal("empty completion should be rejected")
	}
	if !hasAICompletion(normalizeAIAssistResponse(&aiclient.Response{CategorySuggestion: "技术"})) {
		t.Fatal("taxonomy-only completion should remain usable")
	}
}
