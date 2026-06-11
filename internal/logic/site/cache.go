package site

import (
	"context"
	"time"

	"notes-of-ashen/internal/svc"
	"notes-of-ashen/model"

	"github.com/zeromicro/go-zero/core/logx"
)

const (
	siteSettingsCacheKey = "site:settings"
	resumePageCacheKey   = "site:resume"
	projectsPageCacheKey = "site:projects"
	siteCacheTTL         = 10 * time.Minute
)

func cachedSiteSettings(ctx context.Context, svcCtx *svc.ServiceContext) (*model.SiteSettings, error) {
	var settings model.SiteSettings
	if hit, err := svcCtx.Cache.Get(ctx, siteSettingsCacheKey, &settings); err == nil && hit {
		return &settings, nil
	} else if err != nil {
		logx.Errorf("site settings cache read failed: %v", err)
	}
	fresh, err := svcCtx.Store.SiteSettings(ctx)
	if err != nil {
		return nil, err
	}
	if err := svcCtx.Cache.Set(ctx, siteSettingsCacheKey, fresh, siteCacheTTL); err != nil {
		logx.Errorf("site settings cache write failed: %v", err)
	}
	return fresh, nil
}

func cachedResumePageContent(ctx context.Context, svcCtx *svc.ServiceContext) (*model.ResumePageContent, error) {
	var content model.ResumePageContent
	if hit, err := svcCtx.Cache.Get(ctx, resumePageCacheKey, &content); err == nil && hit {
		return &content, nil
	} else if err != nil {
		logx.Errorf("resume page cache read failed: %v", err)
	}
	fresh, err := svcCtx.Store.ResumePageContent(ctx)
	if err != nil {
		return nil, err
	}
	if err := svcCtx.Cache.Set(ctx, resumePageCacheKey, fresh, siteCacheTTL); err != nil {
		logx.Errorf("resume page cache write failed: %v", err)
	}
	return fresh, nil
}

func cachedProjectsPageContent(ctx context.Context, svcCtx *svc.ServiceContext) (*model.ProjectsPageContent, error) {
	var content model.ProjectsPageContent
	if hit, err := svcCtx.Cache.Get(ctx, projectsPageCacheKey, &content); err == nil && hit {
		return &content, nil
	} else if err != nil {
		logx.Errorf("projects page cache read failed: %v", err)
	}
	fresh, err := svcCtx.Store.ProjectsPageContent(ctx)
	if err != nil {
		return nil, err
	}
	if err := svcCtx.Cache.Set(ctx, projectsPageCacheKey, fresh, siteCacheTTL); err != nil {
		logx.Errorf("projects page cache write failed: %v", err)
	}
	return fresh, nil
}

func evictSiteSettingsCache(ctx context.Context, svcCtx *svc.ServiceContext) {
	if err := svcCtx.Cache.Delete(ctx, siteSettingsCacheKey); err != nil {
		logx.Errorf("site settings cache eviction failed: %v", err)
	}
}

func evictResumePageCache(ctx context.Context, svcCtx *svc.ServiceContext) {
	if err := svcCtx.Cache.Delete(ctx, resumePageCacheKey); err != nil {
		logx.Errorf("resume page cache eviction failed: %v", err)
	}
}

func evictProjectsPageCache(ctx context.Context, svcCtx *svc.ServiceContext) {
	if err := svcCtx.Cache.Delete(ctx, projectsPageCacheKey); err != nil {
		logx.Errorf("projects page cache eviction failed: %v", err)
	}
}

func EvictProjectsPageCache(ctx context.Context, svcCtx *svc.ServiceContext) {
	evictProjectsPageCache(ctx, svcCtx)
}
