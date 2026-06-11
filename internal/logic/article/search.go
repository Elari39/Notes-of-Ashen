package article

import (
	"context"
	"time"

	searchclient "notes-of-ashen/internal/search"
	"notes-of-ashen/internal/svc"
	"notes-of-ashen/internal/types"
	"notes-of-ashen/model"

	"github.com/zeromicro/go-zero/core/logx"
)

func searchPublicArticles(ctx context.Context, svcCtx *svc.ServiceContext, req types.ArticleListReq, page, size int) (*types.ArticleListResp, bool) {
	if svcCtx.Search == nil || !svcCtx.Search.Enabled() {
		return nil, false
	}
	result, err := svcCtx.Search.Search(ctx, searchclient.SearchRequest{
		Query:      req.Query,
		Page:       page,
		Size:       size,
		CategoryID: req.CategoryID,
		TagID:      req.TagID,
		Now:        time.Now(),
	})
	if err != nil {
		logx.Errorf("meilisearch query failed, fallback to mysql: %v", err)
		return nil, false
	}
	items, err := svcCtx.Store.FindArticlesByIDs(ctx, result.IDs)
	if err != nil {
		logx.Errorf("load meilisearch article ids failed, fallback to mysql: %v", err)
		return nil, false
	}
	resp := make([]types.ArticleResp, 0, len(items))
	for _, item := range items {
		if !model.IsArticlePubliclyVisible(item, time.Now()) {
			continue
		}
		article := articleResp(ctx, svcCtx, item, false)
		if highlight, ok := result.Highlights[item.ID]; ok {
			article.SearchHighlights = &types.ArticleSearchHighlights{
				Title:   highlight.Title,
				Summary: highlight.Summary,
				Content: highlight.Content,
			}
		}
		resp = append(resp, article)
	}
	return &types.ArticleListResp{Items: resp, Total: result.Total, Page: page, Size: size}, true
}

func ReindexSearch(ctx context.Context, svcCtx *svc.ServiceContext) (int, error) {
	if svcCtx.Search == nil || !svcCtx.Search.Enabled() {
		return 0, nil
	}
	docs, err := svcCtx.Store.ListArticleSearchDocuments(ctx)
	if err != nil {
		return 0, err
	}
	searchDocs := make([]searchclient.Document, 0, len(docs))
	for _, doc := range docs {
		searchDocs = append(searchDocs, searchDocument(doc))
	}
	if err := svcCtx.Search.Reindex(ctx, searchDocs); err != nil {
		return 0, err
	}
	return len(searchDocs), nil
}

func syncArticleSearch(ctx context.Context, svcCtx *svc.ServiceContext, articleID uint64) {
	if svcCtx.Search == nil || !svcCtx.Search.Enabled() {
		return
	}
	doc, err := svcCtx.Store.FindArticleSearchDocument(ctx, articleID)
	if err == model.ErrNotFound {
		if err := svcCtx.Search.Delete(ctx, articleID); err != nil {
			logx.Errorf("delete article from search index failed, article=%d, err=%v", articleID, err)
		}
		return
	}
	if err != nil {
		logx.Errorf("build article search document failed, article=%d, err=%v", articleID, err)
		return
	}
	if err := svcCtx.Search.Upsert(ctx, searchDocument(*doc)); err != nil {
		logx.Errorf("upsert article search index failed, article=%d, err=%v", articleID, err)
	}
}

func deleteArticleSearch(ctx context.Context, svcCtx *svc.ServiceContext, articleID uint64) {
	if svcCtx.Search == nil || !svcCtx.Search.Enabled() {
		return
	}
	if err := svcCtx.Search.Delete(ctx, articleID); err != nil {
		logx.Errorf("delete article search index failed, article=%d, err=%v", articleID, err)
	}
}

func searchDocument(doc model.ArticleSearchDocument) searchclient.Document {
	return searchclient.Document{
		ID:          doc.ID,
		Title:       doc.Title,
		Summary:     doc.Summary,
		Content:     doc.Content,
		Status:      doc.Status,
		VisibleAt:   doc.VisibleAt,
		CategoryID:  doc.CategoryID,
		Category:    doc.Category,
		TagIDs:      doc.TagIDs,
		Tags:        doc.Tags,
		CreatedAt:   doc.CreatedAt,
		PublishedAt: doc.PublishedAt,
	}
}
