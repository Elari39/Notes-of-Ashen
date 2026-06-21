package admin

import (
	"context"
	"fmt"
	"time"

	"notes-of-ashen/internal/svc"
	"notes-of-ashen/model"

	"github.com/zeromicro/go-zero/core/logx"
)

const dashboardArticleCacheTTL = time.Minute

const (
	// popularArticlesCachePrefix / recentArticlesCachePrefix 作为缓存 key 前缀，
	// 驱逐时按前缀清空，避免硬编码 limit 导致 limit 变更后脏缓存（P4-12）。
	popularArticlesCachePrefix = "article:popular:"
	recentArticlesCachePrefix  = "article:recent:"
)

func cachedPopularArticles(ctx context.Context, svcCtx *svc.ServiceContext, limit int) ([]model.Article, error) {
	key := fmt.Sprintf("%s%d", popularArticlesCachePrefix, limit)
	var items []model.Article
	if hit, err := svcCtx.Cache.Get(ctx, key, &items); err == nil && hit {
		return items, nil
	} else if err != nil {
		logx.Errorf("popular articles cache read failed: %v", err)
	}
	fresh, err := svcCtx.Store.PopularArticles(ctx, limit)
	if err != nil {
		return nil, err
	}
	if err := svcCtx.Cache.Set(ctx, key, fresh, dashboardArticleCacheTTL); err != nil {
		logx.Errorf("popular articles cache write failed: %v", err)
	}
	return fresh, nil
}

func cachedRecentArticles(ctx context.Context, svcCtx *svc.ServiceContext, limit int) ([]model.Article, error) {
	key := fmt.Sprintf("%s%d", recentArticlesCachePrefix, limit)
	var items []model.Article
	if hit, err := svcCtx.Cache.Get(ctx, key, &items); err == nil && hit {
		return items, nil
	} else if err != nil {
		logx.Errorf("recent articles cache read failed: %v", err)
	}
	fresh, err := svcCtx.Store.RecentArticles(ctx, limit)
	if err != nil {
		return nil, err
	}
	if err := svcCtx.Cache.Set(ctx, key, fresh, dashboardArticleCacheTTL); err != nil {
		logx.Errorf("recent articles cache write failed: %v", err)
	}
	return fresh, nil
}
