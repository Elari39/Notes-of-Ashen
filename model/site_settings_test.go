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
