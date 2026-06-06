package admin

import (
	"context"

	"notes-of-ashen/internal/authutil"
	"notes-of-ashen/internal/logicutil"
	"notes-of-ashen/internal/svc"
	"notes-of-ashen/internal/types"
	"notes-of-ashen/internal/validator"
)

var userStatuses = map[string]struct{}{
	"active":   {},
	"disabled": {},
}

func ListUsers(ctx context.Context, svcCtx *svc.ServiceContext, page, size int) (*types.ListResp[types.UserResp], error) {
	if err := authutil.RequireAdmin(ctx); err != nil {
		return nil, err
	}
	page, size = logicutil.Page(page, size)
	items, total, err := svcCtx.Store.ListUsers(ctx, page, size)
	if err != nil {
		return nil, err
	}
	resp := make([]types.UserResp, 0, len(items))
	for _, item := range items {
		resp = append(resp, logicutil.UserResp(item))
	}
	return &types.ListResp[types.UserResp]{Items: resp, Total: total, Page: page, Size: size}, nil
}

func UpdateUserStatus(ctx context.Context, svcCtx *svc.ServiceContext, userID uint64, req types.UserStatusReq) error {
	if err := authutil.RequireAdmin(ctx); err != nil {
		return err
	}
	if err := validator.Status(req.Status, userStatuses, "status"); err != nil {
		return err
	}
	return logicutil.MapError(svcCtx.Store.UpdateUserStatus(ctx, userID, req.Status))
}

func ListLogs(ctx context.Context, svcCtx *svc.ServiceContext, page, size int) (*types.ListResp[types.OperationLogResp], error) {
	if err := authutil.RequireAdmin(ctx); err != nil {
		return nil, err
	}
	page, size = logicutil.Page(page, size)
	items, total, err := svcCtx.Store.ListOperationLogs(ctx, page, size)
	if err != nil {
		return nil, err
	}
	resp := make([]types.OperationLogResp, 0, len(items))
	for _, item := range items {
		resp = append(resp, logicutil.OperationLogResp(item))
	}
	return &types.ListResp[types.OperationLogResp]{Items: resp, Total: total, Page: page, Size: size}, nil
}
