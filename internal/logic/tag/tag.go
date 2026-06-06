package tag

import (
	"context"
	"strings"

	"notes-of-ashen/internal/authutil"
	apperrors "notes-of-ashen/internal/errors"
	"notes-of-ashen/internal/logicutil"
	"notes-of-ashen/internal/svc"
	"notes-of-ashen/internal/types"
	"notes-of-ashen/internal/validator"
	"notes-of-ashen/model"
)

func List(ctx context.Context, svcCtx *svc.ServiceContext, page, size int) (*types.ListResp[types.TagResp], error) {
	page, size = logicutil.Page(page, size)
	items, total, err := svcCtx.Store.ListTags(ctx, page, size)
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
	if err := authutil.RequireAdmin(ctx); err != nil {
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
	if err := authutil.RequireAdmin(ctx); err != nil {
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
	resp := logicutil.TagResp(*item)
	return &resp, nil
}

func Delete(ctx context.Context, svcCtx *svc.ServiceContext, id uint64) error {
	if err := authutil.RequireAdmin(ctx); err != nil {
		return err
	}
	return logicutil.MapError(svcCtx.Store.DeleteTag(ctx, id))
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
