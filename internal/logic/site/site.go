package site

import (
	"context"

	"notes-of-ashen/internal/authutil"
	"notes-of-ashen/internal/errors"
	"notes-of-ashen/internal/svc"
	"notes-of-ashen/internal/types"
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
	if err := svcCtx.Store.UpdateSiteSettings(ctx, model.SiteSettings{
		RegistrationEnabled: req.RegistrationEnabled,
		HomeArticleLayout:   layout,
	}); err != nil {
		return nil, err
	}
	settings, err := svcCtx.Store.SiteSettings(ctx)
	if err != nil {
		return nil, err
	}
	return siteSettingsResp(settings, false), nil
}

func siteSettingsResp(settings *model.SiteSettings, forceRegistrationEnabled bool) *types.SiteSettingsResp {
	return &types.SiteSettingsResp{
		RegistrationEnabled: forceRegistrationEnabled || settings.RegistrationEnabled,
		HomeArticleLayout:   settings.HomeArticleLayout,
	}
}

func isValidHomeArticleLayout(layout string) bool {
	return layout == model.HomeArticleLayoutStandard || layout == model.HomeArticleLayoutAlternating
}
