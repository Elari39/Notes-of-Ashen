package site

import (
	"context"

	"notes-of-ashen/internal/authutil"
	"notes-of-ashen/internal/svc"
	"notes-of-ashen/internal/types"
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
	return &types.SiteSettingsResp{RegistrationEnabled: total == 0 || settings.RegistrationEnabled}, nil
}

func UpdateSettings(ctx context.Context, svcCtx *svc.ServiceContext, req types.UpdateSiteSettingsReq) (*types.SiteSettingsResp, error) {
	if err := authutil.RequireAdmin(ctx); err != nil {
		return nil, err
	}
	if err := svcCtx.Store.UpdateRegistrationEnabled(ctx, req.RegistrationEnabled); err != nil {
		return nil, err
	}
	settings, err := svcCtx.Store.SiteSettings(ctx)
	if err != nil {
		return nil, err
	}
	return &types.SiteSettingsResp{RegistrationEnabled: settings.RegistrationEnabled}, nil
}
