package admin

import (
	"context"
	"time"

	"notes-of-ashen/internal/svc"
	"notes-of-ashen/model"

	"github.com/zeromicro/go-zero/core/logx"
)

const dashboardArticleCacheTTL = time.Minute

func cachedPopularArticles(ctx context.Context, svcCtx *svc.ServiceContext, limit int) ([]model.Article, error) {
	key := "article:popular:5"
	var items []model.Article
	if limit == 5 {
		if hit, err := svcCtx.Cache.Get(ctx, key, &items); err == nil && hit {
			return items, nil
		} else if err != nil {
			logx.Errorf("popular articles cache read failed: %v", err)
		}
	}
	fresh, err := svcCtx.Store.PopularArticles(ctx, limit)
	if err != nil {
		return nil, err
	}
	if limit == 5 {
		if err := svcCtx.Cache.Set(ctx, key, fresh, dashboardArticleCacheTTL); err != nil {
			logx.Errorf("popular articles cache write failed: %v", err)
		}
	}
	return fresh, nil
}

func cachedRecentArticles(ctx context.Context, svcCtx *svc.ServiceContext, limit int) ([]model.Article, error) {
	key := "article:recent:5"
	var items []model.Article
	if limit == 5 {
		if hit, err := svcCtx.Cache.Get(ctx, key, &items); err == nil && hit {
			return items, nil
		} else if err != nil {
			logx.Errorf("recent articles cache read failed: %v", err)
		}
	}
	fresh, err := svcCtx.Store.RecentArticles(ctx, limit)
	if err != nil {
		return nil, err
	}
	if limit == 5 {
		if err := svcCtx.Cache.Set(ctx, key, fresh, dashboardArticleCacheTTL); err != nil {
			logx.Errorf("recent articles cache write failed: %v", err)
		}
	}
	return fresh, nil
}
