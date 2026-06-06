package article

import (
	"context"
	"strings"

	"notes-of-ashen/internal/authutil"
	apperrors "notes-of-ashen/internal/errors"
	"notes-of-ashen/internal/logicutil"
	"notes-of-ashen/internal/mq"
	"notes-of-ashen/internal/svc"
	"notes-of-ashen/internal/types"
	"notes-of-ashen/internal/validator"
	"notes-of-ashen/model"
)

var statuses = map[string]struct{}{
	"draft":     {},
	"published": {},
	"archived":  {},
}

func List(ctx context.Context, svcCtx *svc.ServiceContext, page, size int, status string) (*types.ArticleListResp, error) {
	return list(ctx, svcCtx, types.ArticleListReq{
		Page:   page,
		Size:   size,
		Status: status,
	})
}

func ListByFilter(ctx context.Context, svcCtx *svc.ServiceContext, req types.ArticleListReq) (*types.ArticleListResp, error) {
	return list(ctx, svcCtx, req)
}

func AdminList(ctx context.Context, svcCtx *svc.ServiceContext, req types.ArticleListReq) (*types.ArticleListResp, error) {
	if err := authutil.RequireAdmin(ctx); err != nil {
		return nil, err
	}
	return listWithFilter(ctx, svcCtx, req, model.ArticleFilter{Role: "admin"})
}

func list(ctx context.Context, svcCtx *svc.ServiceContext, req types.ArticleListReq) (*types.ArticleListResp, error) {
	return listWithFilter(ctx, svcCtx, req, model.ArticleFilter{})
}

func listWithFilter(ctx context.Context, svcCtx *svc.ServiceContext, req types.ArticleListReq, filter model.ArticleFilter) (*types.ArticleListResp, error) {
	page, size := logicutil.Page(req.Page, req.Size)
	status := strings.TrimSpace(req.Status)
	query := strings.TrimSpace(req.Query)
	if status != "" {
		if err := validator.Status(status, statuses, "status"); err != nil {
			return nil, err
		}
	}
	if query != "" {
		if err := validator.Length(query, "q", 1, 160); err != nil {
			return nil, err
		}
	}
	items, total, err := svcCtx.Store.ListArticles(ctx, model.ArticleFilter{
		UserID:     filter.UserID,
		Role:       filter.Role,
		Status:     status,
		Query:      query,
		CategoryID: req.CategoryID,
		TagID:      req.TagID,
		Page:       page,
		Size:       size,
	})
	if err != nil {
		return nil, err
	}
	resp := make([]types.ArticleResp, 0, len(items))
	for _, item := range items {
		resp = append(resp, articleResp(ctx, svcCtx, item, false))
	}
	return &types.ArticleListResp{Items: resp, Total: total, Page: page, Size: size}, nil
}

func Detail(ctx context.Context, svcCtx *svc.ServiceContext, id uint64) (*types.ArticleResp, error) {
	item, err := svcCtx.Store.FindArticle(ctx, id)
	if err != nil {
		return nil, logicutil.MapError(err)
	}
	if item.Status != "published" {
		return nil, apperrors.NotFound("article not found")
	}
	if err := svcCtx.Store.IncreaseArticleView(ctx, id); err == nil {
		item.ViewCount++
	}
	resp := articleResp(ctx, svcCtx, *item, true)
	return &resp, nil
}

func Create(ctx context.Context, svcCtx *svc.ServiceContext, req types.ArticleReq, meta types.RequestMeta) (*types.ArticleResp, error) {
	userID, err := authutil.UserID(ctx)
	if err != nil {
		return nil, err
	}
	if req.Status == "" {
		req.Status = "draft"
	}
	if err := validateArticle(ctx, svcCtx, req); err != nil {
		return nil, err
	}
	id, err := svcCtx.Store.CreateArticle(ctx, model.ArticleCreate{
		AuthorID:   userID,
		CategoryID: req.CategoryID,
		Title:      strings.TrimSpace(req.Title),
		Slug:       logicutil.NormalizeSlug(req.Slug),
		Summary:    strings.TrimSpace(req.Summary),
		Content:    req.Content,
		CoverURL:   strings.TrimSpace(req.CoverURL),
		Status:     req.Status,
		TagIDs:     req.TagIDs,
	})
	if err != nil {
		if logicutil.IsDuplicate(err) {
			return nil, apperrors.Conflict("article slug already exists")
		}
		return nil, logicutil.MapError(err)
	}
	item, err := svcCtx.Store.FindArticle(ctx, id)
	if err != nil {
		return nil, logicutil.MapError(err)
	}
	publishEvent(ctx, svcCtx, mq.Event{
		UserID:       userID,
		EventType:    "article.created",
		ResourceType: "article",
		ResourceID:   id,
		IP:           meta.IP,
		UserAgent:    meta.UserAgent,
	})
	resp := articleResp(ctx, svcCtx, *item, true)
	return &resp, nil
}

func Update(ctx context.Context, svcCtx *svc.ServiceContext, id uint64, req types.ArticleReq, meta types.RequestMeta) (*types.ArticleResp, error) {
	userID, role, err := currentActor(ctx)
	if err != nil {
		return nil, err
	}
	current, err := svcCtx.Store.FindArticle(ctx, id)
	if err != nil {
		return nil, logicutil.MapError(err)
	}
	if err := canManageArticle(userID, role, *current); err != nil {
		return nil, err
	}
	if req.Status == "" {
		req.Status = current.Status
	}
	if err := validateArticle(ctx, svcCtx, req); err != nil {
		return nil, err
	}
	err = svcCtx.Store.UpdateArticle(ctx, id, model.ArticleUpdate{
		CategoryID: req.CategoryID,
		Title:      strings.TrimSpace(req.Title),
		Slug:       logicutil.NormalizeSlug(req.Slug),
		Summary:    strings.TrimSpace(req.Summary),
		Content:    req.Content,
		CoverURL:   strings.TrimSpace(req.CoverURL),
		Status:     req.Status,
		TagIDs:     req.TagIDs,
	})
	if err != nil {
		if logicutil.IsDuplicate(err) {
			return nil, apperrors.Conflict("article slug already exists")
		}
		return nil, logicutil.MapError(err)
	}
	item, err := svcCtx.Store.FindArticle(ctx, id)
	if err != nil {
		return nil, logicutil.MapError(err)
	}
	publishEvent(ctx, svcCtx, mq.Event{
		UserID:       userID,
		EventType:    "article.updated",
		ResourceType: "article",
		ResourceID:   id,
		IP:           meta.IP,
		UserAgent:    meta.UserAgent,
	})
	resp := articleResp(ctx, svcCtx, *item, true)
	return &resp, nil
}

func Delete(ctx context.Context, svcCtx *svc.ServiceContext, id uint64, meta types.RequestMeta) error {
	userID, role, err := currentActor(ctx)
	if err != nil {
		return err
	}
	current, err := svcCtx.Store.FindArticle(ctx, id)
	if err != nil {
		return logicutil.MapError(err)
	}
	if err := canManageArticle(userID, role, *current); err != nil {
		return err
	}
	if err := svcCtx.Store.DeleteArticle(ctx, id); err != nil {
		return logicutil.MapError(err)
	}
	publishEvent(ctx, svcCtx, mq.Event{
		UserID:       userID,
		EventType:    "article.deleted",
		ResourceType: "article",
		ResourceID:   id,
		IP:           meta.IP,
		UserAgent:    meta.UserAgent,
	})
	return nil
}

func UpdateStatus(ctx context.Context, svcCtx *svc.ServiceContext, id uint64, req types.ArticleStatusReq, meta types.RequestMeta) (*types.ArticleResp, error) {
	userID, role, err := currentActor(ctx)
	if err != nil {
		return nil, err
	}
	req.Status = strings.TrimSpace(req.Status)
	if err := validator.Status(req.Status, statuses, "status"); err != nil {
		return nil, err
	}
	current, err := svcCtx.Store.FindArticle(ctx, id)
	if err != nil {
		return nil, logicutil.MapError(err)
	}
	if err := canManageArticle(userID, role, *current); err != nil {
		return nil, err
	}
	if err := svcCtx.Store.UpdateArticleStatus(ctx, id, req.Status); err != nil {
		return nil, logicutil.MapError(err)
	}
	item, err := svcCtx.Store.FindArticle(ctx, id)
	if err != nil {
		return nil, logicutil.MapError(err)
	}
	publishEvent(ctx, svcCtx, mq.Event{
		UserID:       userID,
		EventType:    "article.status_updated",
		ResourceType: "article",
		ResourceID:   id,
		Metadata:     map[string]string{"status": req.Status},
		IP:           meta.IP,
		UserAgent:    meta.UserAgent,
	})
	resp := articleResp(ctx, svcCtx, *item, true)
	return &resp, nil
}

func validateArticle(ctx context.Context, svcCtx *svc.ServiceContext, req types.ArticleReq) error {
	if err := validator.Length(strings.TrimSpace(req.Title), "title", 1, 160); err != nil {
		return err
	}
	if err := validator.Length(logicutil.NormalizeSlug(req.Slug), "slug", 1, 180); err != nil {
		return err
	}
	if err := validator.Required(req.Content, "content"); err != nil {
		return err
	}
	if err := validator.Status(req.Status, statuses, "status"); err != nil {
		return err
	}
	if req.CategoryID > 0 {
		if _, err := svcCtx.Store.FindCategory(ctx, req.CategoryID); err != nil {
			return logicutil.MapError(err)
		}
	}
	if err := svcCtx.Store.EnsureTagsExist(ctx, req.TagIDs); err != nil {
		if err == model.ErrNotFound {
			return apperrors.NotFound("tag not found")
		}
		return err
	}
	return nil
}

func currentActor(ctx context.Context) (uint64, string, error) {
	userID, err := authutil.UserID(ctx)
	if err != nil {
		return 0, "", err
	}
	return userID, authutil.Role(ctx), nil
}

func canManageArticle(userID uint64, role string, item model.Article) error {
	if role == "admin" || item.AuthorID == userID {
		return nil
	}
	return apperrors.Forbidden("cannot manage other user's article")
}

func articleResp(ctx context.Context, svcCtx *svc.ServiceContext, item model.Article, includeContent bool) types.ArticleResp {
	tags, err := svcCtx.Store.ArticleTags(ctx, item.ID)
	if err != nil {
		tags = nil
	}
	var category *model.Category
	if item.CategoryID > 0 {
		category, _ = svcCtx.Store.FindCategory(ctx, item.CategoryID)
	}
	return logicutil.ArticleResp(item, tags, category, includeContent)
}

func publishEvent(ctx context.Context, svcCtx *svc.ServiceContext, event mq.Event) {
	if svcCtx.Events != nil {
		svcCtx.Events.Publish(ctx, event)
	}
}
