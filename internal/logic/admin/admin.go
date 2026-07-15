package admin

import (
	"context"
	"errors"
	"net"
	"strconv"
	"strings"
	"time"

	"golang.org/x/sync/errgroup"

	"notes-of-ashen/internal/authutil"
	apperrors "notes-of-ashen/internal/errors"
	"notes-of-ashen/internal/logicutil"
	"notes-of-ashen/internal/middleware"
	"notes-of-ashen/internal/svc"
	"notes-of-ashen/internal/types"
	"notes-of-ashen/internal/validator"
	"notes-of-ashen/model"
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
	if err := svcCtx.Store.UpdateUserStatusSafely(ctx, userID, currentID, req.Status); err != nil {
		return mapAdminUpdateError(err)
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
	if err := svcCtx.Store.UpdateUserRoleSafely(ctx, userID, currentID, req.Role); err != nil {
		return mapAdminUpdateError(err)
	}
	middleware.EvictAuthUserCache(ctx, svcCtx.AuthUserCache, userID)
	return nil
}

func ListLogs(ctx context.Context, svcCtx *svc.ServiceContext, req types.OperationLogListReq) (*types.ListResp[types.OperationLogResp], error) {
	if err := authutil.RequireAdmin(ctx); err != nil {
		return nil, err
	}
	filter, err := operationLogFilter(req)
	if err != nil {
		return nil, err
	}
	items, total, err := svcCtx.Store.ListOperationLogsFiltered(ctx, filter)
	if err != nil {
		return nil, err
	}
	resp := make([]types.OperationLogResp, 0, len(items))
	for _, item := range items {
		resp = append(resp, logicutil.OperationLogResp(item))
	}
	return &types.ListResp[types.OperationLogResp]{Items: resp, Total: total, Page: filter.Page, Size: filter.Size}, nil
}

func operationLogFilter(req types.OperationLogListReq) (model.OperationLogFilter, error) {
	page, size := logicutil.Page(req.Page, req.Size)
	filter := model.OperationLogFilter{
		Page:      page,
		Size:      size,
		EventType: strings.TrimSpace(req.EventType),
		IP:        strings.TrimSpace(req.IP),
	}
	if filter.EventType != "" {
		if err := validator.Length(filter.EventType, "eventType", 1, 64); err != nil {
			return model.OperationLogFilter{}, err
		}
	}

	actor := strings.TrimSpace(req.Actor)
	if actor != "" {
		if err := validator.Length(actor, "actor", 1, 64); err != nil {
			return model.OperationLogFilter{}, err
		}
		if isDigits(actor) {
			userID, err := strconv.ParseUint(actor, 10, 64)
			if err != nil || userID == 0 {
				return model.OperationLogFilter{}, apperrors.BadRequest("actor is invalid")
			}
			filter.UserID = userID
		} else {
			filter.UserAccount = actor
		}
	}

	if filter.IP != "" && net.ParseIP(filter.IP) == nil {
		return model.OperationLogFilter{}, apperrors.BadRequest("ip format is invalid")
	}

	startAt, err := optionalRFC3339(req.StartAt, "startAt")
	if err != nil {
		return model.OperationLogFilter{}, err
	}
	endAt, err := optionalRFC3339(req.EndAt, "endAt")
	if err != nil {
		return model.OperationLogFilter{}, err
	}
	if startAt != nil && endAt != nil && !startAt.Before(*endAt) {
		return model.OperationLogFilter{}, apperrors.BadRequest("startAt must be before endAt")
	}
	filter.StartAt = startAt
	filter.EndAt = endAt
	return filter, nil
}

func optionalRFC3339(value, field string) (*time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return nil, apperrors.BadRequest(field + " format is invalid")
	}
	utc := parsed.UTC()
	return &utc, nil
}

func isDigits(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func Stats(ctx context.Context, svcCtx *svc.ServiceContext) (*types.AdminStatsResp, error) {
	if err := authutil.RequireContentManager(ctx); err != nil {
		return nil, err
	}
	// 仪表盘各数据源互相无依赖，并行查询以减少串行 DB 往返。
	var (
		stats       *model.AdminStats
		popular     []model.Article
		recent      []model.Article
		logs        []model.OperationLog
		trend       []model.TrafficTrendPoint
		topReferers []model.RefererStat
		today       model.TrafficTrendPoint
	)
	var g errgroup.Group
	g.Go(func() (err error) {
		stats, err = svcCtx.Store.AdminStats(ctx)
		return err
	})
	g.Go(func() (err error) {
		popular, err = cachedPopularArticles(ctx, svcCtx, 5)
		return err
	})
	g.Go(func() (err error) {
		recent, err = cachedRecentArticles(ctx, svcCtx, 5)
		return err
	})
	g.Go(func() (err error) {
		logs, _, err = svcCtx.Store.ListOperationLogs(ctx, 1, 5)
		return err
	})
	g.Go(func() (err error) {
		trend, err = svcCtx.Store.TrafficTrend(ctx, 30)
		return err
	})
	g.Go(func() (err error) {
		topReferers, err = svcCtx.Store.TopReferers(ctx, 30, 8)
		return err
	})
	g.Go(func() (err error) {
		today, err = svcCtx.Store.TodayTraffic(ctx, logicutil.TodayDate())
		return err
	})
	if err := g.Wait(); err != nil {
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

func mapAdminUpdateError(err error) error {
	switch {
	case errors.Is(err, model.ErrCannotDisableSelf):
		return apperrors.Forbidden("cannot disable yourself")
	case errors.Is(err, model.ErrCannotDowngradeSelf):
		return apperrors.Forbidden("cannot downgrade yourself")
	case errors.Is(err, model.ErrLastActiveAdmin):
		return apperrors.Forbidden("at least one active admin is required")
	default:
		return logicutil.MapError(err)
	}
}
