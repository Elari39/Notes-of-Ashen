package site

import (
	"testing"

	"notes-of-ashen/model"
)

func TestSiteSettingsResp(t *testing.T) {
	settings := &model.SiteSettings{
		RegistrationEnabled: false,
		HomeArticleLayout:   model.HomeArticleLayoutAlternating,
	}

	resp := siteSettingsResp(settings, true)
	if !resp.RegistrationEnabled {
		t.Fatal("siteSettingsResp should force registration enabled when no users exist")
	}
	if resp.HomeArticleLayout != model.HomeArticleLayoutAlternating {
		t.Fatalf("HomeArticleLayout = %q, want %q", resp.HomeArticleLayout, model.HomeArticleLayoutAlternating)
	}

	resp = siteSettingsResp(settings, false)
	if resp.RegistrationEnabled {
		t.Fatal("siteSettingsResp should keep stored registration flag when force flag is false")
	}
}

func TestIsValidHomeArticleLayout(t *testing.T) {
	tests := []struct {
		layout string
		want   bool
	}{
		{layout: model.HomeArticleLayoutStandard, want: true},
		{layout: model.HomeArticleLayoutAlternating, want: true},
		{layout: "", want: false},
		{layout: "grid", want: false},
	}

	for _, tt := range tests {
		if got := isValidHomeArticleLayout(tt.layout); got != tt.want {
			t.Fatalf("isValidHomeArticleLayout(%q) = %v, want %v", tt.layout, got, tt.want)
		}
	}
}
