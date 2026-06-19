package admin

import (
	"context"

	"notes-of-ashen/internal/authutil"
	apperrors "notes-of-ashen/internal/errors"
	"notes-of-ashen/internal/logicutil"
	"notes-of-ashen/internal/middleware"
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
	currentID, err := authutil.UserID(ctx)
	if err != nil {
		return err
	}
	if err := validator.Status(req.Status, userStatuses, "status"); err != nil {
		return err
	}
	target, err := svcCtx.Store.FindUserByID(ctx, userID)
	if err != nil {
		return logicutil.MapError(err)
	}
	if target.ID == currentID && req.Status != "active" {
		return apperrors.Forbidden("cannot disable yourself")
	}
	if target.Role == authutil.RoleAdmin && target.Status == "active" && req.Status != "active" {
		if err := ensureAnotherActiveAdmin(ctx, svcCtx); err != nil {
			return err
		}
	}
	if err := svcCtx.Store.UpdateUserStatus(ctx, userID, req.Status); err != nil {
		return logicutil.MapError(err)
	}
	middleware.EvictAuthUserCache(ctx, svcCtx.AuthUserCache, userID)
	return nil
}

func UpdateUserRole(ctx context.Context, svcCtx *svc.ServiceContext, userID uint64, req types.UserRoleReq) error {
	if err := authutil.RequireAdmin(ctx); err != nil {
		return err
	}
	currentID, err := authutil.UserID(ctx)
	if err != nil {
		return err
	}
	if !authutil.IsValidRole(req.Role) {
		return apperrors.BadRequest("role is invalid")
	}
	target, err := svcCtx.Store.FindUserByID(ctx, userID)
	if err != nil {
		return logicutil.MapError(err)
	}
	if target.ID == currentID && req.Role != authutil.RoleAdmin {
		return apperrors.Forbidden("cannot downgrade yourself")
	}
	if target.Role == authutil.RoleAdmin && req.Role != authutil.RoleAdmin && target.Status == "active" {
		if err := ensureAnotherActiveAdmin(ctx, svcCtx); err != nil {
			return err
		}
	}
	if err := svcCtx.Store.UpdateUserRole(ctx, userID, req.Role); err != nil {
		return logicutil.MapError(err)
	}
	middleware.EvictAuthUserCache(ctx, svcCtx.AuthUserCache, userID)
	return nil
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

func Stats(ctx context.Context, svcCtx *svc.ServiceContext) (*types.AdminStatsResp, error) {
	if err := authutil.RequireContentManager(ctx); err != nil {
		return nil, err
	}
	stats, err := svcCtx.Store.AdminStats(ctx)
	if err != nil {
		return nil, err
	}
	popular, err := cachedPopularArticles(ctx, svcCtx, 5)
	if err != nil {
		return nil, err
	}
	recent, err := cachedRecentArticles(ctx, svcCtx, 5)
	if err != nil {
		return nil, err
	}
	logs, _, err := svcCtx.Store.ListOperationLogs(ctx, 1, 5)
	if err != nil {
		return nil, err
	}
	trend, err := svcCtx.Store.TrafficTrend(ctx, 30)
	if err != nil {
		return nil, err
	}
	topReferers, err := svcCtx.Store.TopReferers(ctx, 30, 8)
	if err != nil {
		return nil, err
	}
	today, err := svcCtx.Store.TodayTraffic(ctx, logicutil.TodayDate())
	if err != nil {
		return nil, err
	}

	resp := &types.AdminStatsResp{
		ArticleTotal:    stats.ArticleTotal,
		PublishedTotal:  stats.PublishedTotal,
		DraftTotal:      stats.DraftTotal,
		ArchivedTotal:   stats.ArchivedTotal,
		ScheduledTotal:  stats.ScheduledTotal,
		ViewTotal:       stats.ViewTotal,
		LikeTotal:       stats.LikeTotal,
		TodayPV:         today.PV,
		TodayUV:         today.UV,
		UserTotal:       stats.UserTotal,
		CategoryTotal:   stats.CategoryTotal,
		TagTotal:        stats.TagTotal,
		TrafficTrend:    make([]types.TrafficTrendPointResp, 0, len(trend)),
		TopReferers:     make([]types.RefererStatResp, 0, len(topReferers)),
		PopularArticles: make([]types.ArticleResp, 0, len(popular)),
		RecentArticles:  make([]types.ArticleResp, 0, len(recent)),
		RecentLogs:      make([]types.OperationLogResp, 0, len(logs)),
	}
	for _, item := range trend {
		resp.TrafficTrend = append(resp.TrafficTrend, types.TrafficTrendPointResp{
			Date: item.Date,
			PV:   item.PV,
			UV:   item.UV,
		})
	}
	for _, item := range topReferers {
		resp.TopReferers = append(resp.TopReferers, types.RefererStatResp{
			SourceType: item.SourceType,
			SourceName: item.SourceName,
			PV:         item.PV,
		})
	}
	for _, item := range popular {
		resp.PopularArticles = append(resp.PopularArticles, logicutil.ArticleResp(item, nil, nil, false))
	}
	for _, item := range recent {
		resp.RecentArticles = append(resp.RecentArticles, logicutil.ArticleResp(item, nil, nil, false))
	}
	for _, item := range logs {
		resp.RecentLogs = append(resp.RecentLogs, logicutil.OperationLogResp(item))
	}
	return resp, nil
}

func ensureAnotherActiveAdmin(ctx context.Context, svcCtx *svc.ServiceContext) error {
	count, err := svcCtx.Store.CountActiveAdmins(ctx)
	if err != nil {
		return err
	}
	if count <= 1 {
		return apperrors.Forbidden("at least one active admin is required")
	}
	return nil
}
