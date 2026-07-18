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
		HomeCTAHidden:       true,
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
	if !resp.HomeCtaHidden {
		t.Fatal("siteSettingsResp should expose home CTA visibility setting")
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

func TestSiteSettingsForUpdatePreservesMissingFieldsAndClearsBaseURL(t *testing.T) {
	current := model.SiteSettings{
		RegistrationEnabled: true,
		HomeArticleLayout:   model.HomeArticleLayoutAlternating,
		HomeCTAHidden:       false,
		SiteTitle:           "Title",
		SiteDescription:     "Description",
		SiteKeywords:        "go,blog",
		SiteBaseURL:         "https://1.1.1.1",
		ProjectsPageEnabled: true,
		ProjectsNavHidden:   false,
	}
	disabled := false
	next, err := siteSettingsForUpdate(current, types.UpdateSiteSettingsReq{RegistrationEnabled: &disabled})
	if err != nil {
		t.Fatalf("siteSettingsForUpdate() error = %v", err)
	}
	if next.SiteBaseURL != current.SiteBaseURL || next.SiteTitle != current.SiteTitle || next.HomeArticleLayout != current.HomeArticleLayout || next.HomeCTAHidden != current.HomeCTAHidden {
		t.Fatalf("missing fields were not preserved: %#v", next)
	}
	if next.RegistrationEnabled {
		t.Fatal("explicit false registrationEnabled was not applied")
	}

	empty := ""
	next, err = siteSettingsForUpdate(current, types.UpdateSiteSettingsReq{SiteBaseURL: &empty})
	if err != nil {
		t.Fatalf("siteSettingsForUpdate(clear base URL) error = %v", err)
	}
	if next.SiteBaseURL != "" {
		t.Fatalf("SiteBaseURL = %q, want empty", next.SiteBaseURL)
	}
	if next.SiteTitle != current.SiteTitle {
		t.Fatal("clearing siteBaseUrl changed unrelated fields")
	}

	hidden := true
	next, err = siteSettingsForUpdate(current, types.UpdateSiteSettingsReq{HomeCtaHidden: &hidden})
	if err != nil {
		t.Fatalf("siteSettingsForUpdate(hide home CTA) error = %v", err)
	}
	if !next.HomeCTAHidden {
		t.Fatal("explicit true homeCtaHidden was not applied")
	}

	visible := false
	next, err = siteSettingsForUpdate(current, types.UpdateSiteSettingsReq{HomeCtaHidden: &visible})
	if err != nil {
		t.Fatalf("siteSettingsForUpdate(show home CTA) error = %v", err)
	}
	if next.HomeCTAHidden {
		t.Fatal("explicit false homeCtaHidden was not applied")
	}
}

func TestValidateProjectsPageReq(t *testing.T) {
	content, err := validateProjectsPageReq(types.UpdateProjectsPageReq{
		Title:    " 项目 ",
		Subtitle: " 作品集 ",
		Items: []types.UpdateProjectItemReq{
			{
				ID:              "one",
				Title:           " Go Blog ",
				Summary:         "A blog",
				TagIDs:          []uint64{2, 2, 3},
				DemoURL:         "https://1.1.1.1",
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
	if item.Title != "Go Blog" || len(item.TagIDs) != 2 || item.TagIDs[0] != 2 || !item.Featured {
		t.Fatalf("item = %#v, want normalized project item", item)
	}
}

func TestValidateProjectsPageReqRejectsInvalidItems(t *testing.T) {
	if _, err := validateProjectsPageReq(types.UpdateProjectsPageReq{
		Title: "项目",
		Items: []types.UpdateProjectItemReq{{ID: "one", Title: "One", Tags: []string{"Go"}}},
	}); err == nil {
		t.Fatal("validateProjectsPageReq should reject legacy tags writes")
	}

	if _, err := validateProjectsPageReq(types.UpdateProjectsPageReq{
		Title: "项目",
		Items: []types.UpdateProjectItemReq{
			{ID: "same", Title: "One"},
			{ID: "same", Title: "Two"},
		},
	}); err == nil {
		t.Fatal("validateProjectsPageReq should reject duplicate ids")
	}

	if _, err := validateProjectsPageReq(types.UpdateProjectsPageReq{
		Title: "项目",
		Items: []types.UpdateProjectItemReq{{ID: "one", Title: "One", RepoURL: "ftp://example.com/repo"}},
	}); err == nil {
		t.Fatal("validateProjectsPageReq should reject non-http URLs")
	}
}
