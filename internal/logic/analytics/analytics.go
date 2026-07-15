package analytics

import (
	"context"
	"math"
	"strings"
	"time"

	"notes-of-ashen/internal/authutil"
	apperrors "notes-of-ashen/internal/errors"
	"notes-of-ashen/internal/logicutil"
	"notes-of-ashen/internal/svc"
	"notes-of-ashen/internal/types"
	"notes-of-ashen/model"

	"golang.org/x/sync/errgroup"
)

func Overview(ctx context.Context, svcCtx *svc.ServiceContext, req types.AnalyticsRangeReq) (*types.AnalyticsOverviewResp, error) {
	if err := authutil.RequireContentManager(ctx); err != nil {
		return nil, err
	}
	from, to, previousFrom, previousTo, days, err := parseRange(req)
	if err != nil {
		return nil, err
	}
	var current, previous model.AnalyticsSummary
	var trend []model.TrafficTrendPoint
	var pages []model.PageAnalytics
	var referers []model.RefererStat
	group, groupCtx := errgroup.WithContext(ctx)
	group.Go(func() error { var e error; current, e = svcCtx.Store.AnalyticsSummary(groupCtx, from, to); return e })
	group.Go(func() error {
		var e error
		previous, e = svcCtx.Store.AnalyticsSummary(groupCtx, previousFrom, previousTo)
		return e
	})
	group.Go(func() error { var e error; trend, e = svcCtx.Store.TrafficTrendRange(groupCtx, from, to); return e })
	group.Go(func() error { var e error; pages, e = svcCtx.Store.TopPagesRange(groupCtx, from, to, 10); return e })
	group.Go(func() error {
		var e error
		referers, e = svcCtx.Store.TopReferersRange(groupCtx, from, to, 0, 10)
		return e
	})
	if err := group.Wait(); err != nil {
		return nil, err
	}
	resp := &types.AnalyticsOverviewResp{
		From: from, To: to, Summary: summaryResp(current, previous),
		Trend: fillTrend(from, days, trend), TopReferers: refererResp(referers),
		TopPages: make([]types.PageAnalyticsResp, 0, len(pages)),
	}
	for _, page := range pages {
		resp.TopPages = append(resp.TopPages, types.PageAnalyticsResp{
			RouteType: page.RouteType, Path: page.Path, ArticleID: page.ArticleID,
			Title: page.Title, PV: page.PV, UV: page.UV,
		})
	}
	return resp, nil
}

func Articles(ctx context.Context, svcCtx *svc.ServiceContext, req types.AnalyticsRangeReq, query string, page, size int) (*types.ListResp[types.ArticleAnalyticsResp], error) {
	if err := authutil.RequireContentManager(ctx); err != nil {
		return nil, err
	}
	from, to, _, _, _, err := parseRange(req)
	if err != nil {
		return nil, err
	}
	page, size = logicutil.Page(page, size)
	items, total, err := svcCtx.Store.ListArticleAnalytics(ctx, from, to, strings.TrimSpace(query), page, size)
	if err != nil {
		return nil, err
	}
	resp := make([]types.ArticleAnalyticsResp, 0, len(items))
	for _, item := range items {
		resp = append(resp, articleResp(item))
	}
	return &types.ListResp[types.ArticleAnalyticsResp]{Items: resp, Total: total, Page: page, Size: size}, nil
}

func Detail(ctx context.Context, svcCtx *svc.ServiceContext, id uint64, req types.AnalyticsRangeReq) (*types.ArticleAnalyticsDetailResp, error) {
	if err := authutil.RequireContentManager(ctx); err != nil {
		return nil, err
	}
	from, to, _, _, days, err := parseRange(req)
	if err != nil {
		return nil, err
	}
	article, points, err := svcCtx.Store.ArticleAnalyticsDetail(ctx, id, from, to)
	if err != nil {
		return nil, logicutil.MapError(err)
	}
	referers, err := svcCtx.Store.TopReferersRange(ctx, from, to, id, 10)
	if err != nil {
		return nil, err
	}
	byDate := make(map[string]model.ArticleAnalyticsPoint, len(points))
	for _, point := range points {
		byDate[point.Date] = point
	}
	trend := make([]types.ArticleAnalyticsPointResp, 0, days)
	start, _ := time.ParseInLocation("2006-01-02", from, time.Local)
	for i := 0; i < days; i++ {
		date := start.AddDate(0, 0, i).Format("2006-01-02")
		point := byDate[date]
		trend = append(trend, types.ArticleAnalyticsPointResp{Date: date, PV: point.PV, UV: point.UV, Likes: point.Likes})
	}
	return &types.ArticleAnalyticsDetailResp{
		Article: articleResp(article), From: from, To: to, Trend: trend, Referers: refererResp(referers),
	}, nil
}

func parseRange(req types.AnalyticsRangeReq) (string, string, string, string, int, error) {
	to := time.Now()
	var err error
	if strings.TrimSpace(req.To) != "" {
		to, err = time.ParseInLocation("2006-01-02", req.To, time.Local)
		if err != nil {
			return "", "", "", "", 0, apperrors.BadRequest("to is invalid")
		}
	}
	from := to.AddDate(0, 0, -29)
	if strings.TrimSpace(req.From) != "" {
		from, err = time.ParseInLocation("2006-01-02", req.From, time.Local)
		if err != nil {
			return "", "", "", "", 0, apperrors.BadRequest("from is invalid")
		}
	}
	if from.After(to) {
		return "", "", "", "", 0, apperrors.BadRequest("from must not be after to")
	}
	days := int(to.Sub(from).Hours()/24) + 1
	if days > 366 {
		return "", "", "", "", 0, apperrors.BadRequest("date range must not exceed 366 days")
	}
	previousTo := from.AddDate(0, 0, -1)
	previousFrom := previousTo.AddDate(0, 0, -(days - 1))
	return from.Format("2006-01-02"), to.Format("2006-01-02"), previousFrom.Format("2006-01-02"), previousTo.Format("2006-01-02"), days, nil
}

func change(current, previous int64) *float64 {
	if previous == 0 {
		return nil
	}
	value := math.Round((float64(current-previous)/float64(previous)*100)*100) / 100
	return &value
}

func summaryResp(current, previous model.AnalyticsSummary) types.AnalyticsSummaryResp {
	return types.AnalyticsSummaryResp{
		PV: current.PV, UV: current.UV, Likes: current.Likes,
		PreviousPV: previous.PV, PreviousUV: previous.UV, PreviousLikes: previous.Likes,
		PVChange: change(current.PV, previous.PV), UVChange: change(current.UV, previous.UV), LikesChange: change(current.Likes, previous.Likes),
	}
}

func fillTrend(from string, days int, items []model.TrafficTrendPoint) []types.TrafficTrendPointResp {
	byDate := make(map[string]model.TrafficTrendPoint, len(items))
	for _, item := range items {
		byDate[item.Date] = item
	}
	start, _ := time.ParseInLocation("2006-01-02", from, time.Local)
	out := make([]types.TrafficTrendPointResp, 0, days)
	for i := 0; i < days; i++ {
		date := start.AddDate(0, 0, i).Format("2006-01-02")
		item := byDate[date]
		out = append(out, types.TrafficTrendPointResp{Date: date, PV: item.PV, UV: item.UV})
	}
	return out
}

func refererResp(items []model.RefererStat) []types.RefererStatResp {
	out := make([]types.RefererStatResp, 0, len(items))
	for _, item := range items {
		out = append(out, types.RefererStatResp{SourceType: item.SourceType, SourceName: item.SourceName, PV: item.PV})
	}
	return out
}

func articleResp(item model.ArticleAnalytics) types.ArticleAnalyticsResp {
	return types.ArticleAnalyticsResp{
		ArticleID: item.ArticleID, Title: item.Title, Status: item.Status,
		PV: item.PV, UV: item.UV, Likes: item.Likes, TotalViews: item.TotalViews, TotalLikes: item.TotalLikes,
	}
}
