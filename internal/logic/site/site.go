package site

import (
	"context"
	"strings"

	"notes-of-ashen/internal/authutil"
	"notes-of-ashen/internal/errors"
	"notes-of-ashen/internal/svc"
	"notes-of-ashen/internal/types"
	"notes-of-ashen/internal/validator"
	"notes-of-ashen/model"
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
	if err := svcCtx.Store.UpdateSiteSettings(ctx, model.SiteSettings{
		RegistrationEnabled: registrationEnabled,
		HomeArticleLayout:   layout,
		SiteTitle:           siteTitle,
		SiteDescription:     siteDescription,
		SiteKeywords:        siteKeywords,
		SiteBaseURL:         siteBaseURL,
	}); err != nil {
		return nil, err
	}
	settings, err := svcCtx.Store.SiteSettings(ctx)
	if err != nil {
		return nil, err
	}
	return siteSettingsResp(settings, false), nil
}

func registrationEnabledForUpdate(current bool, requested *bool) bool {
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
	}
}

func isValidHomeArticleLayout(layout string) bool {
	return layout == model.HomeArticleLayoutStandard || layout == model.HomeArticleLayoutAlternating
}
