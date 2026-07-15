package search

import (
	"context"
	"strings"
	"time"

	"notes-of-ashen/internal/authutil"
	apperrors "notes-of-ashen/internal/errors"
	articlelogic "notes-of-ashen/internal/logic/article"
	searchclient "notes-of-ashen/internal/search"
	"notes-of-ashen/internal/svc"
	"notes-of-ashen/internal/types"
	"notes-of-ashen/model"

	"github.com/zeromicro/go-zero/core/logx"
)

func Reindex(ctx context.Context, svcCtx *svc.ServiceContext) (*types.SearchReindexResp, error) {
	if err := authutil.RequireContentManager(ctx); err != nil {
		return nil, err
	}
	if svcCtx.Search == nil || !svcCtx.Search.Enabled() {
		return &types.SearchReindexResp{Indexed: 0, Enabled: false}, nil
	}
	indexed, err := articlelogic.ReindexSearch(ctx, svcCtx)
	if err != nil {
		return nil, err
	}
	return &types.SearchReindexResp{Indexed: indexed, Enabled: true}, nil
}

func Suggestions(ctx context.Context, svcCtx *svc.ServiceContext, query string, limit int) (*types.SearchSuggestionsResp, error) {
	query = strings.TrimSpace(query)
	if len([]rune(query)) < 2 || len([]rune(query)) > 80 {
		return nil, apperrors.BadRequest("q length must be between 2 and 80")
	}
	if limit < 1 {
		limit = 8
	}
	if limit > 10 {
		limit = 10
	}

	items := make([]types.SearchSuggestionResp, 0, limit)
	articleLimit := min(limit, 4)
	articles, err := suggestionArticles(ctx, svcCtx, query, articleLimit)
	if err != nil {
		return nil, err
	}
	for _, item := range articles {
		items = append(items, types.SearchSuggestionResp{Kind: "article", ID: item.ID, Label: item.Title})
	}

	remaining := limit - len(items)
	if remaining > 0 {
		categories, err := svcCtx.Store.SuggestCategories(ctx, query, min(remaining, 3))
		if err != nil {
			return nil, err
		}
		for _, item := range categories {
			items = append(items, types.SearchSuggestionResp{Kind: "category", ID: item.ID, Label: item.Name, ArticleCount: item.ArticleCount})
		}
	}
	remaining = limit - len(items)
	if remaining > 0 {
		tags, err := svcCtx.Store.SuggestTags(ctx, query, remaining)
		if err != nil {
			return nil, err
		}
		for _, item := range tags {
			items = append(items, types.SearchSuggestionResp{Kind: "tag", ID: item.ID, Label: item.Name, ArticleCount: item.ArticleCount})
		}
	}
	return &types.SearchSuggestionsResp{Items: items}, nil
}

func suggestionArticles(ctx context.Context, svcCtx *svc.ServiceContext, query string, limit int) ([]model.Article, error) {
	if svcCtx.Search != nil && svcCtx.Search.Enabled() {
		result, err := svcCtx.Search.Search(ctx, searchclient.SearchRequest{Query: query, Page: 1, Size: limit, Now: time.Now()})
		if err == nil && len(result.IDs) > 0 {
			items, loadErr := svcCtx.Store.FindArticlesByIDs(ctx, result.IDs)
			if loadErr == nil {
				now := time.Now()
				visible := make([]model.Article, 0, len(items))
				for _, item := range items {
					if model.IsArticlePubliclyVisible(item, now) {
						visible = append(visible, item)
					}
				}
				return visible, nil
			}
			logx.Errorf("load meilisearch suggestion ids failed, fallback to mysql: %v", loadErr)
		}
		if err != nil {
			logx.Errorf("meilisearch suggestions failed, fallback to mysql: %v", err)
		}
	}
	items, _, err := svcCtx.Store.ListArticles(ctx, model.ArticleFilter{Status: model.ArticleStatusPublished, Query: query, Page: 1, Size: limit})
	return items, err
}
