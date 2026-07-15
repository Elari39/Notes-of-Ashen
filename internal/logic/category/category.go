package category

import (
	"context"
	"strings"

	"notes-of-ashen/internal/authutil"
	apperrors "notes-of-ashen/internal/errors"
	articlelogic "notes-of-ashen/internal/logic/article"
	"notes-of-ashen/internal/logicutil"
	"notes-of-ashen/internal/svc"
	"notes-of-ashen/internal/types"
	"notes-of-ashen/internal/validator"
	"notes-of-ashen/model"
)

func List(ctx context.Context, svcCtx *svc.ServiceContext, page, size int) (*types.ListResp[types.CategoryResp], error) {
	page, size = logicutil.Page(page, size)
	items, total, err := svcCtx.Store.ListCategories(ctx, page, size, true)
	if err != nil {
		return nil, err
	}
	resp := make([]types.CategoryResp, 0, len(items))
	for _, item := range items {
		resp = append(resp, logicutil.CategoryResp(item))
	}
	return &types.ListResp[types.CategoryResp]{Items: resp, Total: total, Page: page, Size: size}, nil
}

func AdminList(ctx context.Context, svcCtx *svc.ServiceContext, page, size int) (*types.ListResp[types.CategoryResp], error) {
	if err := authutil.RequireContentManager(ctx); err != nil {
		return nil, err
	}
	page, size = logicutil.Page(page, size)
	items, total, err := svcCtx.Store.ListCategories(ctx, page, size, false)
	if err != nil {
		return nil, err
	}
	resp := make([]types.CategoryResp, 0, len(items))
	for _, item := range items {
		resp = append(resp, logicutil.CategoryResp(item))
	}
	return &types.ListResp[types.CategoryResp]{Items: resp, Total: total, Page: page, Size: size}, nil
}

func Create(ctx context.Context, svcCtx *svc.ServiceContext, req types.TaxonomyReq) (*types.CategoryResp, error) {
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
	id, err := svcCtx.Store.CreateCategory(ctx, model.TaxonomyCreate{
		Name:        strings.TrimSpace(req.Name),
		Slug:        logicutil.NormalizeSlug(req.Slug),
		Description: strings.TrimSpace(req.Description),
		CreatedBy:   userID,
	})
	if err != nil {
		if logicutil.IsDuplicate(err) {
			return nil, apperrors.Conflict("category already exists")
		}
		return nil, err
	}
	item, err := svcCtx.Store.FindCategory(ctx, id)
	if err != nil {
		return nil, logicutil.MapError(err)
	}
	resp := logicutil.CategoryResp(*item)
	return &resp, nil
}

func Update(ctx context.Context, svcCtx *svc.ServiceContext, id uint64, req types.TaxonomyReq) (*types.CategoryResp, error) {
	if err := authutil.RequireContentManager(ctx); err != nil {
		return nil, err
	}
	if err := validate(req); err != nil {
		return nil, err
	}
	err := svcCtx.Store.UpdateCategory(ctx, id, model.TaxonomyUpdate{
		Name:        strings.TrimSpace(req.Name),
		Slug:        logicutil.NormalizeSlug(req.Slug),
		Description: strings.TrimSpace(req.Description),
	})
	if err != nil {
		if logicutil.IsDuplicate(err) {
			return nil, apperrors.Conflict("category already exists")
		}
		return nil, logicutil.MapError(err)
	}
	item, err := svcCtx.Store.FindCategory(ctx, id)
	if err != nil {
		return nil, logicutil.MapError(err)
	}
	articlelogic.RefreshDerivedPublicState(ctx, svcCtx)
	resp := logicutil.CategoryResp(*item)
	return &resp, nil
}

func Delete(ctx context.Context, svcCtx *svc.ServiceContext, id uint64) error {
	if err := authutil.RequireContentManager(ctx); err != nil {
		return err
	}
	if err := svcCtx.Store.DeleteCategory(ctx, id); err != nil {
		return logicutil.MapError(err)
	}
	articlelogic.RefreshDerivedPublicState(ctx, svcCtx)
	return nil
}

func validate(req types.TaxonomyReq) error {
	if err := validator.Length(strings.TrimSpace(req.Name), "name", 1, 64); err != nil {
		return err
	}
	if err := validator.Length(logicutil.NormalizeSlug(req.Slug), "slug", 1, 96); err != nil {
		return err
	}
	return nil
}
