package site

import (
	"testing"

	"notes-of-ashen/internal/types"
	"notes-of-ashen/model"
)

func TestSiteSettingsResp(t *testing.T) {
	settings := &model.SiteSettings{
		RegistrationEnabled: false,
		HomeArticleLayout:   model.HomeArticleLayoutAlternating,
		ProjectsPageEnabled: true,
		ProjectsNavHidden:   true,
	}

	resp := siteSettingsResp(settings, true, false)
	if !resp.RegistrationEnabled {
		t.Fatal("siteSettingsResp should force registration enabled when no users exist")
	}
	if resp.RegistrationEmailCodeRequired {
		t.Fatal("siteSettingsResp should expose first-admin email code bypass when email service is disabled")
	}
	if resp.HomeArticleLayout != model.HomeArticleLayoutAlternating {
		t.Fatalf("HomeArticleLayout = %q, want %q", resp.HomeArticleLayout, model.HomeArticleLayoutAlternating)
	}
	if !resp.ProjectsPageEnabled || !resp.ProjectsNavHidden {
		t.Fatal("siteSettingsResp should expose public page visibility settings")
	}

	resp = siteSettingsResp(settings, false, true)
	if resp.RegistrationEnabled {
		t.Fatal("siteSettingsResp should keep stored registration flag when force flag is false")
	}
	if !resp.RegistrationEmailCodeRequired {
		t.Fatal("siteSettingsResp should expose email code requirement for normal registration")
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

func TestRegistrationEnabledForUpdate(t *testing.T) {
	if got := registrationEnabledForUpdate(true, nil); !got {
		t.Fatal("registrationEnabledForUpdate should keep true when request is nil")
	}
	if got := registrationEnabledForUpdate(false, nil); got {
		t.Fatal("registrationEnabledForUpdate should keep false when request is nil")
	}

	enabled := true
	if got := registrationEnabledForUpdate(false, &enabled); !got {
		t.Fatal("registrationEnabledForUpdate should use explicit true")
	}

	disabled := false
	if got := registrationEnabledForUpdate(true, &disabled); got {
		t.Fatal("registrationEnabledForUpdate should use explicit false")
	}
}

func TestValidateProjectsPageReq(t *testing.T) {
	content, err := validateProjectsPageReq(types.UpdateProjectsPageReq{
		Title:    " 项目 ",
		Subtitle: " 作品集 ",
		Items: []types.ProjectItem{
			{
				ID:              "one",
				Title:           " Go Blog ",
				Summary:         "A blog",
				Tags:            []string{"Go", "go", "React"},
				DemoURL:         "https://example.com",
				ContentMarkdown: "## Detail",
				Featured:        true,
			},
		},
	})
	if err != nil {
		t.Fatalf("validateProjectsPageReq returned error: %v", err)
	}
	if content.Title != "项目" || len(content.Items) != 1 {
		t.Fatalf("content = %#v, want normalized page and one item", content)
	}
	item := content.Items[0]
	if item.Title != "Go Blog" || len(item.Tags) != 2 || item.Tags[0] != "Go" || !item.Featured {
		t.Fatalf("item = %#v, want normalized project item", item)
	}
}

func TestValidateProjectsPageReqRejectsInvalidItems(t *testing.T) {
	if _, err := validateProjectsPageReq(types.UpdateProjectsPageReq{
		Title: "项目",
		Items: []types.ProjectItem{
			{ID: "same", Title: "One"},
			{ID: "same", Title: "Two"},
		},
	}); err == nil {
		t.Fatal("validateProjectsPageReq should reject duplicate ids")
	}

	if _, err := validateProjectsPageReq(types.UpdateProjectsPageReq{
		Title: "项目",
		Items: []types.ProjectItem{{ID: "one", Title: "One", RepoURL: "ftp://example.com/repo"}},
	}); err == nil {
		t.Fatal("validateProjectsPageReq should reject non-http URLs")
	}
}
