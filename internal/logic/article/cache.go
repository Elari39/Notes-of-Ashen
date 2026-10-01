package article

import (
	"context"
	"time"

	appcache "notes-of-ashen/internal/cache"
	"notes-of-ashen/internal/svc"
	"notes-of-ashen/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

const (
	articleListCachePrefix = "article:list:"
	articleListCacheTTL    = 2 * time.Minute
)

type cachedArticleList struct {
	Response   types.ArticleListResp `json:"response"`
	ValidUntil time.Time             `json:"validUntil"`
}

func cacheablePublicArticleList(req types.ArticleListReq, filterRole string, filterUserID uint64, query string) bool {
	return filterRole == "" && filterUserID == 0 && query == ""
}

func publicArticleListCacheKey(req types.ArticleListReq, page, size int, status string) string {
	return appcache.HashKey(articleListCachePrefix+"v2:", page, size, status, req.CategoryID, req.TagID)
}

func getCachedArticleList(ctx context.Context, svcCtx *svc.ServiceContext, key string) (*types.ArticleListResp, bool) {
	var resp cachedArticleList
	hit, err := svcCtx.Cache.Get(ctx, key, &resp)
	if err != nil {
		logx.Errorf("article list cache read failed: %v", err)
		return nil, false
	}
	if !hit || !time.Now().Before(resp.ValidUntil) {
		return nil, false
	}
	return &resp.Response, true
}

func articleListCacheDeadline(started time.Time, nextPublication *time.Time) time.Time {
	deadline := started.Add(articleListCacheTTL)
	if nextPublication != nil && nextPublication.Before(deadline) {
		deadline = *nextPublication
	}
	return deadline
}

func setCachedArticleList(ctx context.Context, svcCtx *svc.ServiceContext, key string, resp *types.ArticleListResp, deadline time.Time) {
	// Redis 的非正 TTL 表示永久存储；到点或查询失败时必须直接不写缓存。
	ttl := time.Until(deadline)
	if ttl <= 0 {
		return
	}
	entry := cachedArticleList{Response: *resp, ValidUntil: deadline}
	if err := svcCtx.Cache.Set(ctx, key, entry, ttl); err != nil {
		logx.Errorf("article list cache write failed: %v", err)
	}
}

func evictArticleCaches(ctx context.Context, svcCtx *svc.ServiceContext) {
	if err := svcCtx.Cache.DeletePrefix(ctx, articleListCachePrefix); err != nil {
		logx.Errorf("article list cache eviction failed: %v", err)
	}
	if err := svcCtx.Cache.DeletePrefix(ctx, "article:popular:"); err != nil {
		logx.Errorf("article popular cache eviction failed: %v", err)
	}
	if err := svcCtx.Cache.DeletePrefix(ctx, "article:recent:"); err != nil {
		logx.Errorf("article recent cache eviction failed: %v", err)
	}
}

func RefreshDerivedPublicState(ctx context.Context, svcCtx *svc.ServiceContext) {
	evictArticleCaches(ctx, svcCtx)
	// 分类/标签变更只影响关联文章的搜索文档，全量 Reindex 较重，改为异步执行
	// 不阻塞分类/标签操作响应；索引最终一致（秒级延迟）。
	if svcCtx.Search == nil || !svcCtx.Search.Enabled() {
		return
	}
	go func() {
		reindexCtx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		if _, err := ReindexSearch(reindexCtx, svcCtx); err != nil {
			logx.Errorf("article search reindex after taxonomy change failed: %v", err)
		}
	}()
}
