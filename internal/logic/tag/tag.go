package tag

import (
	"context"
	"strings"

	"notes-of-ashen/internal/authutil"
	apperrors "notes-of-ashen/internal/errors"
	articlelogic "notes-of-ashen/internal/logic/article"
	sitelogic "notes-of-ashen/internal/logic/site"
	"notes-of-ashen/internal/logicutil"
	"notes-of-ashen/internal/svc"
	"notes-of-ashen/internal/types"
	"notes-of-ashen/internal/validator"
	"notes-of-ashen/model"
)

const maxTagDescriptionBytes = 65535

func List(ctx context.Context, svcCtx *svc.ServiceContext, page, size int) (*types.ListResp[types.TagResp], error) {
	page, size = logicutil.Page(page, size)
	items, total, err := svcCtx.Store.ListTags(ctx, page, size, true)
	if err != nil {
		return nil, err
	}
	resp := make([]types.TagResp, 0, len(items))
	for _, item := range items {
		resp = append(resp, logicutil.TagResp(item))
	}
	return &types.ListResp[types.TagResp]{Items: resp, Total: total, Page: page, Size: size}, nil
}

func AdminList(ctx context.Context, svcCtx *svc.ServiceContext, page, size int) (*types.ListResp[types.TagResp], error) {
	if err := authutil.RequireContentManager(ctx); err != nil {
		return nil, err
	}
	page, size = logicutil.Page(page, size)
	items, total, err := svcCtx.Store.ListTags(ctx, page, size, false)
	if err != nil {
		return nil, err
	}
	resp := make([]types.TagResp, 0, len(items))
	for _, item := range items {
		resp = append(resp, logicutil.TagResp(item))
	}
	return &types.ListResp[types.TagResp]{Items: resp, Total: total, Page: page, Size: size}, nil
}

func Create(ctx context.Context, svcCtx *svc.ServiceContext, req types.TaxonomyReq) (*types.TagResp, error) {
	userID, err := authutil.UserID(ctx)
	if err != nil {
		return nil, err
	}
	if err := authutil.RequireContentManager(ctx); err != nil {
		return nil, err
	}
	if err := validate(req); err != nil {
		return nil, err
	}
	id, err := svcCtx.Store.CreateTag(ctx, model.TaxonomyCreate{
		Name:        strings.TrimSpace(req.Name),
		Slug:        logicutil.NormalizeSlug(req.Slug),
		Description: strings.TrimSpace(req.Description),
		CreatedBy:   userID,
	})
	if err != nil {
		if logicutil.IsDuplicate(err) {
			return nil, apperrors.Conflict("tag already exists")
		}
		return nil, err
	}
	item, err := svcCtx.Store.FindTag(ctx, id)
	if err != nil {
		return nil, logicutil.MapError(err)
	}
	resp := logicutil.TagResp(*item)
	return &resp, nil
}

func Update(ctx context.Context, svcCtx *svc.ServiceContext, id uint64, req types.TaxonomyReq) (*types.TagResp, error) {
	if err := authutil.RequireContentManager(ctx); err != nil {
		return nil, err
	}
	if err := validate(req); err != nil {
		return nil, err
	}
	err := svcCtx.Store.UpdateTag(ctx, id, model.TaxonomyUpdate{
		Name:        strings.TrimSpace(req.Name),
		Slug:        logicutil.NormalizeSlug(req.Slug),
		Description: strings.TrimSpace(req.Description),
	})
	if err != nil {
		if logicutil.IsDuplicate(err) {
			return nil, apperrors.Conflict("tag already exists")
		}
		return nil, logicutil.MapError(err)
	}
	item, err := svcCtx.Store.FindTag(ctx, id)
	if err != nil {
		return nil, logicutil.MapError(err)
	}
	articlelogic.RefreshDerivedPublicState(ctx, svcCtx)
	sitelogic.EvictProjectsPageCache(ctx, svcCtx)
	resp := logicutil.TagResp(*item)
	return &resp, nil
}

func Delete(ctx context.Context, svcCtx *svc.ServiceContext, id uint64) error {
	if err := authutil.RequireContentManager(ctx); err != nil {
		return err
	}
	if err := svcCtx.Store.DeleteTag(ctx, id); err != nil {
		return logicutil.MapError(err)
	}
	articlelogic.RefreshDerivedPublicState(ctx, svcCtx)
	sitelogic.EvictProjectsPageCache(ctx, svcCtx)
	return nil
}

func validate(req types.TaxonomyReq) error {
	if err := validator.Length(strings.TrimSpace(req.Name), "name", 1, 64); err != nil {
		return err
	}
	if err := validator.Length(logicutil.NormalizeSlug(req.Slug), "slug", 1, 96); err != nil {
		return err
	}
	if err := validator.ByteLength(strings.TrimSpace(req.Description), "description", 0, maxTagDescriptionBytes); err != nil {
		return err
	}
	return nil
}
