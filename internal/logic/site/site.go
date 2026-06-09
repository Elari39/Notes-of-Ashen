package site

import (
	"context"
	"fmt"
	"strings"

	"notes-of-ashen/internal/authutil"
	"notes-of-ashen/internal/errors"
	"notes-of-ashen/internal/svc"
	"notes-of-ashen/internal/types"
	"notes-of-ashen/internal/validator"
	"notes-of-ashen/model"
)

const (
	maxPageContentLength     = 200000
	maxResumeItemsCount      = 30
	maxResumeSkillsCount     = 80
	maxResumeHighlightsCount = 12
	maxProjectsCount         = 50
	maxProjectContentLength  = 50000
	maxProjectTagsCount      = 12
)

func Settings(ctx context.Context, svcCtx *svc.ServiceContext) (*types.SiteSettingsResp, error) {
	settings, err := svcCtx.Store.SiteSettings(ctx)
	if err != nil {
		return nil, err
	}
	total, err := svcCtx.Store.CountUsers(ctx)
	if err != nil {
		return nil, err
	}
	return siteSettingsResp(settings, total == 0), nil
}

func UpdateSettings(ctx context.Context, svcCtx *svc.ServiceContext, req types.UpdateSiteSettingsReq) (*types.SiteSettingsResp, error) {
	if err := authutil.RequireAdmin(ctx); err != nil {
		return nil, err
	}
	currentSettings, err := svcCtx.Store.SiteSettings(ctx)
	if err != nil {
		return nil, err
	}
	layout := req.HomeArticleLayout
	if layout == "" {
		layout = currentSettings.HomeArticleLayout
	}
	if !isValidHomeArticleLayout(layout) {
		return nil, errors.BadRequest("homeArticleLayout is invalid")
	}
	siteTitle := strings.TrimSpace(req.SiteTitle)
	if siteTitle == "" {
		siteTitle = currentSettings.SiteTitle
	}
	if err := validator.Length(siteTitle, "siteTitle", 1, 160); err != nil {
		return nil, err
	}
	siteDescription := strings.TrimSpace(req.SiteDescription)
	if siteDescription == "" {
		siteDescription = currentSettings.SiteDescription
	}
	if err := validator.Length(siteDescription, "siteDescription", 1, 255); err != nil {
		return nil, err
	}
	siteKeywords := strings.TrimSpace(req.SiteKeywords)
	if siteKeywords == "" {
		siteKeywords = currentSettings.SiteKeywords
	}
	if err := validator.Length(siteKeywords, "siteKeywords", 1, 255); err != nil {
		return nil, err
	}
	siteBaseURL := strings.TrimRight(strings.TrimSpace(req.SiteBaseURL), "/")
	if err := validator.OptionalHTTPURL(siteBaseURL, "siteBaseUrl"); err != nil {
		return nil, err
	}
	registrationEnabled := registrationEnabledForUpdate(currentSettings.RegistrationEnabled, req.RegistrationEnabled)
	resumePageEnabled := boolForUpdate(currentSettings.ResumePageEnabled, req.ResumePageEnabled)
	resumeNavHidden := boolForUpdate(currentSettings.ResumeNavHidden, req.ResumeNavHidden)
	projectsPageEnabled := boolForUpdate(currentSettings.ProjectsPageEnabled, req.ProjectsPageEnabled)
	projectsNavHidden := boolForUpdate(currentSettings.ProjectsNavHidden, req.ProjectsNavHidden)
	if err := svcCtx.Store.UpdateSiteSettings(ctx, model.SiteSettings{
		RegistrationEnabled: registrationEnabled,
		HomeArticleLayout:   layout,
		SiteTitle:           siteTitle,
		SiteDescription:     siteDescription,
		SiteKeywords:        siteKeywords,
		SiteBaseURL:         siteBaseURL,
		ResumePageEnabled:   resumePageEnabled,
		ResumeNavHidden:     resumeNavHidden,
		ProjectsPageEnabled: projectsPageEnabled,
		ProjectsNavHidden:   projectsNavHidden,
	}); err != nil {
		return nil, err
	}
	settings, err := svcCtx.Store.SiteSettings(ctx)
	if err != nil {
		return nil, err
	}
	return siteSettingsResp(settings, false), nil
}

func ResumePage(ctx context.Context, svcCtx *svc.ServiceContext) (*types.ResumePageResp, error) {
	settings, err := svcCtx.Store.SiteSettings(ctx)
	if err != nil {
		return nil, err
	}
	if !settings.ResumePageEnabled {
		return nil, errors.Forbidden("feature disabled")
	}
	content, err := svcCtx.Store.ResumePageContent(ctx)
	if err != nil {
		return nil, err
	}
	return resumePageResp(content), nil
}

func AdminResumePage(ctx context.Context, svcCtx *svc.ServiceContext) (*types.ResumePageResp, error) {
	if err := authutil.RequireAdmin(ctx); err != nil {
		return nil, err
	}
	content, err := svcCtx.Store.ResumePageContent(ctx)
	if err != nil {
		return nil, err
	}
	return resumePageResp(content), nil
}

func UpdateResumePage(ctx context.Context, svcCtx *svc.ServiceContext, req types.UpdateResumePageReq) (*types.ResumePageResp, error) {
	if err := authutil.RequireAdmin(ctx); err != nil {
		return nil, err
	}
	content, err := validateResumePageReq(req)
	if err != nil {
		return nil, err
	}
	if err := svcCtx.Store.UpdateResumePageContent(ctx, content); err != nil {
		return nil, err
	}
	saved, err := svcCtx.Store.ResumePageContent(ctx)
	if err != nil {
		return nil, err
	}
	return resumePageResp(saved), nil
}

func ProjectsPage(ctx context.Context, svcCtx *svc.ServiceContext) (*types.ProjectsPageResp, error) {
	settings, err := svcCtx.Store.SiteSettings(ctx)
	if err != nil {
		return nil, err
	}
	if !settings.ProjectsPageEnabled {
		return nil, errors.Forbidden("feature disabled")
	}
	content, err := svcCtx.Store.ProjectsPageContent(ctx)
	if err != nil {
		return nil, err
	}
	return projectsPageResp(content), nil
}

func AdminProjectsPage(ctx context.Context, svcCtx *svc.ServiceContext) (*types.ProjectsPageResp, error) {
	if err := authutil.RequireAdmin(ctx); err != nil {
		return nil, err
	}
	content, err := svcCtx.Store.ProjectsPageContent(ctx)
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
	saved, err := svcCtx.Store.ProjectsPageContent(ctx)
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

func siteSettingsResp(settings *model.SiteSettings, forceRegistrationEnabled bool) *types.SiteSettingsResp {
	return &types.SiteSettingsResp{
		RegistrationEnabled: forceRegistrationEnabled || settings.RegistrationEnabled,
		HomeArticleLayout:   settings.HomeArticleLayout,
		SiteTitle:           settings.SiteTitle,
		SiteDescription:     settings.SiteDescription,
		SiteKeywords:        settings.SiteKeywords,
		SiteBaseURL:         settings.SiteBaseURL,
		ResumePageEnabled:   settings.ResumePageEnabled,
		ResumeNavHidden:     settings.ResumeNavHidden,
		ProjectsPageEnabled: settings.ProjectsPageEnabled,
		ProjectsNavHidden:   settings.ProjectsNavHidden,
	}
}

func isValidHomeArticleLayout(layout string) bool {
	return layout == model.HomeArticleLayoutStandard || layout == model.HomeArticleLayoutAlternating
}

func validateResumePageReq(req types.UpdateResumePageReq) (model.ResumePageContent, error) {
	title := strings.TrimSpace(req.Title)
	if err := validator.Length(title, "title", 1, 160); err != nil {
		return model.ResumePageContent{}, err
	}
	subtitle := strings.TrimSpace(req.Subtitle)
	if err := validator.Length(subtitle, "subtitle", 0, 255); err != nil {
		return model.ResumePageContent{}, err
	}
	if err := validator.Length(req.ContentMarkdown, "contentMarkdown", 0, maxPageContentLength); err != nil {
		return model.ResumePageContent{}, err
	}
	experiences, err := validateResumeExperiences(req.Experiences)
	if err != nil {
		return model.ResumePageContent{}, err
	}
	educations, err := validateResumeEducations(req.Educations)
	if err != nil {
		return model.ResumePageContent{}, err
	}
	skills, err := validateResumeSkills(req.Skills)
	if err != nil {
		return model.ResumePageContent{}, err
	}
	return model.ResumePageContent{
		Title:           title,
		Subtitle:        subtitle,
		ContentMarkdown: req.ContentMarkdown,
		Experiences:     experiences,
		Educations:      educations,
		Skills:          skills,
	}, nil
}

func validateResumeExperiences(items []types.ResumeExperience) ([]model.ResumeExperience, error) {
	if len(items) > maxResumeItemsCount {
		return nil, errors.BadRequest("experiences count is invalid")
	}
	out := make([]model.ResumeExperience, 0, len(items))
	for index, item := range items {
		field := func(name string) string {
			return fmt.Sprintf("experiences.%d.%s", index, name)
		}
		role := strings.TrimSpace(item.Role)
		organization := strings.TrimSpace(item.Organization)
		location := strings.TrimSpace(item.Location)
		startDate := strings.TrimSpace(item.StartDate)
		endDate := strings.TrimSpace(item.EndDate)
		description := strings.TrimSpace(item.Description)
		if err := validator.Length(role, field("role"), 1, 120); err != nil {
			return nil, err
		}
		if err := validator.Length(organization, field("organization"), 1, 120); err != nil {
			return nil, err
		}
		if err := validator.Length(location, field("location"), 0, 120); err != nil {
			return nil, err
		}
		if err := validator.Length(startDate, field("startDate"), 0, 32); err != nil {
			return nil, err
		}
		if err := validator.Length(endDate, field("endDate"), 0, 32); err != nil {
			return nil, err
		}
		if err := validator.Length(description, field("description"), 0, 1000); err != nil {
			return nil, err
		}
		highlights, err := validateHighlights(item.Highlights, field("highlights"))
		if err != nil {
			return nil, err
		}
		out = append(out, model.ResumeExperience{
			Role:         role,
			Organization: organization,
			Location:     location,
			StartDate:    startDate,
			EndDate:      endDate,
			Description:  description,
			Highlights:   highlights,
			DisplayOrder: index + 1,
		})
	}
	return out, nil
}

func validateResumeEducations(items []types.ResumeEducation) ([]model.ResumeEducation, error) {
	if len(items) > maxResumeItemsCount {
		return nil, errors.BadRequest("educations count is invalid")
	}
	out := make([]model.ResumeEducation, 0, len(items))
	for index, item := range items {
		field := func(name string) string {
			return fmt.Sprintf("educations.%d.%s", index, name)
		}
		school := strings.TrimSpace(item.School)
		degree := strings.TrimSpace(item.Degree)
		major := strings.TrimSpace(item.Major)
		location := strings.TrimSpace(item.Location)
		startDate := strings.TrimSpace(item.StartDate)
		endDate := strings.TrimSpace(item.EndDate)
		description := strings.TrimSpace(item.Description)
		if err := validator.Length(school, field("school"), 1, 120); err != nil {
			return nil, err
		}
		if err := validator.Length(degree, field("degree"), 0, 120); err != nil {
			return nil, err
		}
		if err := validator.Length(major, field("major"), 0, 120); err != nil {
			return nil, err
		}
		if err := validator.Length(location, field("location"), 0, 120); err != nil {
			return nil, err
		}
		if err := validator.Length(startDate, field("startDate"), 0, 32); err != nil {
			return nil, err
		}
		if err := validator.Length(endDate, field("endDate"), 0, 32); err != nil {
			return nil, err
		}
		if err := validator.Length(description, field("description"), 0, 1000); err != nil {
			return nil, err
		}
		highlights, err := validateHighlights(item.Highlights, field("highlights"))
		if err != nil {
			return nil, err
		}
		out = append(out, model.ResumeEducation{
			School:       school,
			Degree:       degree,
			Major:        major,
			Location:     location,
			StartDate:    startDate,
			EndDate:      endDate,
			Description:  description,
			Highlights:   highlights,
			DisplayOrder: index + 1,
		})
	}
	return out, nil
}

func validateResumeSkills(items []types.ResumeSkill) ([]model.ResumeSkill, error) {
	if len(items) > maxResumeSkillsCount {
		return nil, errors.BadRequest("skills count is invalid")
	}
	out := make([]model.ResumeSkill, 0, len(items))
	for index, item := range items {
		field := func(name string) string {
			return fmt.Sprintf("skills.%d.%s", index, name)
		}
		category := strings.TrimSpace(item.Category)
		name := strings.TrimSpace(item.Name)
		description := strings.TrimSpace(item.Description)
		if err := validator.Length(category, field("category"), 1, 80); err != nil {
			return nil, err
		}
		if err := validator.Length(name, field("name"), 1, 80); err != nil {
			return nil, err
		}
		if item.Level < 0 || item.Level > 100 {
			return nil, errors.BadRequest(field("level") + " is invalid")
		}
		if err := validator.Length(description, field("description"), 0, 255); err != nil {
			return nil, err
		}
		out = append(out, model.ResumeSkill{
			Category:     category,
			Name:         name,
			Level:        item.Level,
			Description:  description,
			DisplayOrder: index + 1,
		})
	}
	return out, nil
}

func validateHighlights(values []string, field string) ([]string, error) {
	if len(values) > maxResumeHighlightsCount {
		return nil, errors.BadRequest(field + " count is invalid")
	}
	out := make([]string, 0, len(values))
	for index, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if err := validator.Length(value, fmt.Sprintf("%s.%d", field, index), 1, 300); err != nil {
			return nil, err
		}
		out = append(out, value)
	}
	return out, nil
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
	return validator.OptionalHTTPURL(value, field)
}

func resumePageResp(content *model.ResumePageContent) *types.ResumePageResp {
	return &types.ResumePageResp{
		Title:           content.Title,
		Subtitle:        content.Subtitle,
		ContentMarkdown: content.ContentMarkdown,
		Experiences:     resumeExperienceResp(content.Experiences),
		Educations:      resumeEducationResp(content.Educations),
		Skills:          resumeSkillResp(content.Skills),
	}
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

func resumeExperienceResp(items []model.ResumeExperience) []types.ResumeExperience {
	out := make([]types.ResumeExperience, 0, len(items))
	for _, item := range items {
		out = append(out, types.ResumeExperience{
			ID:           item.ID,
			Role:         item.Role,
			Organization: item.Organization,
			Location:     item.Location,
			StartDate:    item.StartDate,
			EndDate:      item.EndDate,
			Description:  item.Description,
			Highlights:   item.Highlights,
			DisplayOrder: item.DisplayOrder,
		})
	}
	return out
}

func resumeEducationResp(items []model.ResumeEducation) []types.ResumeEducation {
	out := make([]types.ResumeEducation, 0, len(items))
	for _, item := range items {
		out = append(out, types.ResumeEducation{
			ID:           item.ID,
			School:       item.School,
			Degree:       item.Degree,
			Major:        item.Major,
			Location:     item.Location,
			StartDate:    item.StartDate,
			EndDate:      item.EndDate,
			Description:  item.Description,
			Highlights:   item.Highlights,
			DisplayOrder: item.DisplayOrder,
		})
	}
	return out
}

func resumeSkillResp(items []model.ResumeSkill) []types.ResumeSkill {
	out := make([]types.ResumeSkill, 0, len(items))
	for _, item := range items {
		out = append(out, types.ResumeSkill{
			ID:           item.ID,
			Category:     item.Category,
			Name:         item.Name,
			Level:        item.Level,
			Description:  item.Description,
			DisplayOrder: item.DisplayOrder,
		})
	}
	return out
}
