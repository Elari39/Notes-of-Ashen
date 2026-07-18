package site

import (
	"context"
	"fmt"
	"strings"

	"notes-of-ashen/internal/authutil"
	"notes-of-ashen/internal/errors"
	"notes-of-ashen/internal/logicutil"
	"notes-of-ashen/internal/svc"
	"notes-of-ashen/internal/types"
	"notes-of-ashen/internal/validator"
	"notes-of-ashen/model"
)

const (
	maxPageContentLength    = 200000
	maxProjectsCount        = 50
	maxProjectContentLength = 50000
	maxProjectTagsCount     = 12
)

func Settings(ctx context.Context, svcCtx *svc.ServiceContext) (*types.SiteSettingsResp, error) {
	settings, err := cachedSiteSettings(ctx, svcCtx)
	if err != nil {
		return nil, err
	}
	total, err := svcCtx.Store.CountUsers(ctx)
	if err != nil {
		return nil, err
	}
	isFirstUser := total == 0
	return siteSettingsResp(settings, isFirstUser, logicutil.RegistrationEmailCodeRequired(isFirstUser, svcCtx.Config.Email.Enabled)), nil
}

func UpdateSettings(ctx context.Context, svcCtx *svc.ServiceContext, req types.UpdateSiteSettingsReq) (*types.SiteSettingsResp, error) {
	if err := authutil.RequireAdmin(ctx); err != nil {
		return nil, err
	}
	currentSettings, err := cachedSiteSettings(ctx, svcCtx)
	if err != nil {
		return nil, err
	}
	nextSettings, err := siteSettingsForUpdate(*currentSettings, req)
	if err != nil {
		return nil, err
	}
	if err := svcCtx.Store.UpdateSiteSettings(ctx, nextSettings); err != nil {
		return nil, err
	}
	evictSiteSettingsCache(ctx, svcCtx)
	settings, err := cachedSiteSettings(ctx, svcCtx)
	if err != nil {
		return nil, err
	}
	total, err := svcCtx.Store.CountUsers(ctx)
	if err != nil {
		return nil, err
	}
	isFirstUser := total == 0
	return siteSettingsResp(settings, isFirstUser, logicutil.RegistrationEmailCodeRequired(isFirstUser, svcCtx.Config.Email.Enabled)), nil
}

func siteSettingsForUpdate(currentSettings model.SiteSettings, req types.UpdateSiteSettingsReq) (model.SiteSettings, error) {
	layout := currentSettings.HomeArticleLayout
	if req.HomeArticleLayout != nil && strings.TrimSpace(*req.HomeArticleLayout) != "" {
		layout = strings.TrimSpace(*req.HomeArticleLayout)
	}
	if !isValidHomeArticleLayout(layout) {
		return model.SiteSettings{}, errors.BadRequest("homeArticleLayout is invalid")
	}
	siteTitle := currentSettings.SiteTitle
	if req.SiteTitle != nil && strings.TrimSpace(*req.SiteTitle) != "" {
		siteTitle = strings.TrimSpace(*req.SiteTitle)
	}
	if err := validator.Length(siteTitle, "siteTitle", 1, 160); err != nil {
		return model.SiteSettings{}, err
	}
	siteDescription := currentSettings.SiteDescription
	if req.SiteDescription != nil && strings.TrimSpace(*req.SiteDescription) != "" {
		siteDescription = strings.TrimSpace(*req.SiteDescription)
	}
	if err := validator.Length(siteDescription, "siteDescription", 1, 255); err != nil {
		return model.SiteSettings{}, err
	}
	siteKeywords := currentSettings.SiteKeywords
	if req.SiteKeywords != nil && strings.TrimSpace(*req.SiteKeywords) != "" {
		siteKeywords = strings.TrimSpace(*req.SiteKeywords)
	}
	if err := validator.Length(siteKeywords, "siteKeywords", 1, 255); err != nil {
		return model.SiteSettings{}, err
	}
	siteBaseURL := currentSettings.SiteBaseURL
	if req.SiteBaseURL != nil {
		siteBaseURL = strings.TrimRight(strings.TrimSpace(*req.SiteBaseURL), "/")
	}
	if err := validator.OptionalHTTPURL(siteBaseURL, "siteBaseUrl"); err != nil {
		return model.SiteSettings{}, err
	}
	registrationEnabled := registrationEnabledForUpdate(currentSettings.RegistrationEnabled, req.RegistrationEnabled)
	homeCTAHidden := boolForUpdate(currentSettings.HomeCTAHidden, req.HomeCtaHidden)
	projectsPageEnabled := boolForUpdate(currentSettings.ProjectsPageEnabled, req.ProjectsPageEnabled)
	projectsNavHidden := boolForUpdate(currentSettings.ProjectsNavHidden, req.ProjectsNavHidden)
	return model.SiteSettings{
		RegistrationEnabled: registrationEnabled,
		HomeArticleLayout:   layout,
		HomeCTAHidden:       homeCTAHidden,
		SiteTitle:           siteTitle,
		SiteDescription:     siteDescription,
		SiteKeywords:        siteKeywords,
		SiteBaseURL:         siteBaseURL,
		ProjectsPageEnabled: projectsPageEnabled,
		ProjectsNavHidden:   projectsNavHidden,
	}, nil
}

func ProjectsPage(ctx context.Context, svcCtx *svc.ServiceContext) (*types.ProjectsPageResp, error) {
	settings, err := cachedSiteSettings(ctx, svcCtx)
	if err != nil {
		return nil, err
	}
	if !settings.ProjectsPageEnabled {
		return nil, errors.Forbidden("feature disabled")
	}
	content, err := cachedProjectsPageContent(ctx, svcCtx)
	if err != nil {
		return nil, err
	}
	return projectsPageResp(content), nil
}

func AdminProjectsPage(ctx context.Context, svcCtx *svc.ServiceContext) (*types.ProjectsPageResp, error) {
	if err := authutil.RequireAdmin(ctx); err != nil {
		return nil, err
	}
	content, err := cachedProjectsPageContent(ctx, svcCtx)
	if err != nil {
		return nil, err
	}
	return projectsPageResp(content), nil
}

func UpdateProjectsPage(ctx context.Context, svcCtx *svc.ServiceContext, req types.UpdateProjectsPageReq) (*types.ProjectsPageResp, error) {
	if err := authutil.RequireAdmin(ctx); err != nil {
		return nil, err
	}
	content, err := validateProjectsPageReq(req)
	if err != nil {
		return nil, err
	}
	if err := svcCtx.Store.EnsureTagsExist(ctx, projectTagIDs(content.Items)); err != nil {
		if err == model.ErrNotFound {
			return nil, errors.NotFound("tag not found")
		}
		return nil, err
	}
	if err := svcCtx.Store.UpdateProjectsPageContent(ctx, content); err != nil {
		return nil, err
	}
	evictProjectsPageCache(ctx, svcCtx)
	saved, err := cachedProjectsPageContent(ctx, svcCtx)
	if err != nil {
		return nil, err
	}
	return projectsPageResp(saved), nil
}

func registrationEnabledForUpdate(current bool, requested *bool) bool {
	return boolForUpdate(current, requested)
}

func boolForUpdate(current bool, requested *bool) bool {
	if requested == nil {
		return current
	}
	return *requested
}

func siteSettingsResp(settings *model.SiteSettings, forceRegistrationEnabled bool, registrationEmailCodeRequired bool) *types.SiteSettingsResp {
	return &types.SiteSettingsResp{
		RegistrationEnabled:           forceRegistrationEnabled || settings.RegistrationEnabled,
		RegistrationEmailCodeRequired: registrationEmailCodeRequired,
		HomeArticleLayout:             settings.HomeArticleLayout,
		HomeCtaHidden:                 settings.HomeCTAHidden,
		SiteTitle:                     settings.SiteTitle,
		SiteDescription:               settings.SiteDescription,
		SiteKeywords:                  settings.SiteKeywords,
		SiteBaseURL:                   settings.SiteBaseURL,
		ProjectsPageEnabled:           settings.ProjectsPageEnabled,
		ProjectsNavHidden:             settings.ProjectsNavHidden,
	}
}

func isValidHomeArticleLayout(layout string) bool {
	return layout == model.HomeArticleLayoutStandard || layout == model.HomeArticleLayoutAlternating
}

func validateProjectsPageReq(req types.UpdateProjectsPageReq) (model.ProjectsPageContent, error) {
	title := strings.TrimSpace(req.Title)
	if err := validator.Length(title, "title", 1, 160); err != nil {
		return model.ProjectsPageContent{}, err
	}
	subtitle := strings.TrimSpace(req.Subtitle)
	if err := validator.Length(subtitle, "subtitle", 0, 255); err != nil {
		return model.ProjectsPageContent{}, err
	}
	if len(req.Items) > maxProjectsCount {
		return model.ProjectsPageContent{}, errors.BadRequest("items count is invalid")
	}

	items := make([]model.ProjectItem, 0, len(req.Items))
	for _, item := range req.Items {
		items = append(items, model.ProjectItem{
			ID:              item.ID,
			TagIDs:          item.TagIDs,
			Title:           item.Title,
			Summary:         item.Summary,
			Role:            item.Role,
			Period:          item.Period,
			Tags:            item.Tags,
			CoverURL:        item.CoverURL,
			DemoURL:         item.DemoURL,
			RepoURL:         item.RepoURL,
			ContentMarkdown: item.ContentMarkdown,
			Featured:        item.Featured,
		})
	}
	items = model.NormalizeProjectItems(items)

	seenIDs := make(map[string]struct{}, len(items))
	for index, item := range items {
		field := func(name string) string {
			return fmt.Sprintf("items.%d.%s", index, name)
		}
		if err := validator.Length(item.ID, field("id"), 1, 64); err != nil {
			return model.ProjectsPageContent{}, err
		}
		if _, ok := seenIDs[item.ID]; ok {
			return model.ProjectsPageContent{}, errors.BadRequest("items id is duplicated")
		}
		seenIDs[item.ID] = struct{}{}
		if err := validator.Length(item.Title, field("title"), 1, 120); err != nil {
			return model.ProjectsPageContent{}, err
		}
		if err := validator.Length(item.Summary, field("summary"), 0, 500); err != nil {
			return model.ProjectsPageContent{}, err
		}
		if err := validator.Length(item.Role, field("role"), 0, 80); err != nil {
			return model.ProjectsPageContent{}, err
		}
		if err := validator.Length(item.Period, field("period"), 0, 80); err != nil {
			return model.ProjectsPageContent{}, err
		}
		if err := validateProjectURL(item.CoverURL, field("coverUrl")); err != nil {
			return model.ProjectsPageContent{}, err
		}
		if err := validateProjectURL(item.DemoURL, field("demoUrl")); err != nil {
			return model.ProjectsPageContent{}, err
		}
		if err := validateProjectURL(item.RepoURL, field("repoUrl")); err != nil {
			return model.ProjectsPageContent{}, err
		}
		if err := validator.Length(item.ContentMarkdown, field("contentMarkdown"), 0, maxProjectContentLength); err != nil {
			return model.ProjectsPageContent{}, err
		}
		if len(item.Tags) > maxProjectTagsCount {
			return model.ProjectsPageContent{}, errors.BadRequest(field("tags") + " count is invalid")
		}
		for tagIndex, tag := range item.Tags {
			if err := validator.Length(tag, fmt.Sprintf("%s.%d", field("tags"), tagIndex), 1, 32); err != nil {
				return model.ProjectsPageContent{}, err
			}
		}
	}

	return model.ProjectsPageContent{
		Title:    title,
		Subtitle: subtitle,
		Items:    items,
	}, nil
}

func projectTagIDs(items []model.ProjectItem) []uint64 {
	ids := make([]uint64, 0)
	for _, item := range items {
		ids = append(ids, item.TagIDs...)
	}
	return ids
}

func validateProjectURL(value, field string) error {
	if err := validator.Length(value, field, 0, 255); err != nil {
		return err
	}
	if strings.HasSuffix(field, ".coverUrl") {
		return validator.OptionalImageURL(value, field)
	}
	return validator.OptionalHTTPURL(value, field)
}

func projectsPageResp(content *model.ProjectsPageContent) *types.ProjectsPageResp {
	items := make([]types.ProjectItem, 0, len(content.Items))
	for _, item := range content.Items {
		items = append(items, types.ProjectItem{
			ID:              item.ID,
			TagIDs:          item.TagIDs,
			Title:           item.Title,
			Summary:         item.Summary,
			Role:            item.Role,
			Period:          item.Period,
			Tags:            item.Tags,
			CoverURL:        item.CoverURL,
			DemoURL:         item.DemoURL,
			RepoURL:         item.RepoURL,
			ContentMarkdown: item.ContentMarkdown,
			Featured:        item.Featured,
		})
	}
	return &types.ProjectsPageResp{
		Title:    content.Title,
		Subtitle: content.Subtitle,
		Items:    items,
	}
}
