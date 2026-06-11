package search

import (
	"context"

	"notes-of-ashen/internal/authutil"
	articlelogic "notes-of-ashen/internal/logic/article"
	"notes-of-ashen/internal/svc"
	"notes-of-ashen/internal/types"
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
