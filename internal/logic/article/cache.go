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

func cacheablePublicArticleList(req types.ArticleListReq, filterRole string, filterUserID uint64, query string) bool {
	return filterRole == "" && filterUserID == 0 && query == ""
}

func publicArticleListCacheKey(req types.ArticleListReq, page, size int, status string) string {
	return appcache.HashKey(articleListCachePrefix, page, size, status, req.CategoryID, req.TagID)
}

func getCachedArticleList(ctx context.Context, svcCtx *svc.ServiceContext, key string) (*types.ArticleListResp, bool) {
	var resp types.ArticleListResp
	hit, err := svcCtx.Cache.Get(ctx, key, &resp)
	if err != nil {
		logx.Errorf("article list cache read failed: %v", err)
		return nil, false
	}
	if !hit {
		return nil, false
	}
	return &resp, true
}

func setCachedArticleList(ctx context.Context, svcCtx *svc.ServiceContext, key string, resp *types.ArticleListResp) {
	if err := svcCtx.Cache.Set(ctx, key, resp, articleListCacheTTL); err != nil {
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
