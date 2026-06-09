package article

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"strings"
	"time"

	"notes-of-ashen/internal/aiclient"
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
	model.ArticleStatusDraft:     {},
	model.ArticleStatusPublished: {},
	model.ArticleStatusArchived:  {},
}

var listStatuses = map[string]struct{}{
	model.ArticleStatusDraft:     {},
	model.ArticleStatusPublished: {},
	model.ArticleStatusArchived:  {},
	model.ArticleStatusScheduled: {},
}

var aiActions = map[string]struct{}{
	"metadata":  {},
	"proofread": {},
	"polish":    {},
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
	if err := authutil.RequireContentManager(ctx); err != nil {
		return nil, err
	}
	return listWithFilter(ctx, svcCtx, req, model.ArticleFilter{Role: authutil.Role(ctx)})
}

func list(ctx context.Context, svcCtx *svc.ServiceContext, req types.ArticleListReq) (*types.ArticleListResp, error) {
	return listWithFilter(ctx, svcCtx, req, model.ArticleFilter{})
}

func listWithFilter(ctx context.Context, svcCtx *svc.ServiceContext, req types.ArticleListReq, filter model.ArticleFilter) (*types.ArticleListResp, error) {
	page, size := logicutil.Page(req.Page, req.Size)
	status := strings.TrimSpace(req.Status)
	query := strings.TrimSpace(req.Query)
	if status != "" {
		if err := validator.Status(status, listStatuses, "status"); err != nil {
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
	if !model.IsArticlePubliclyVisible(*item, time.Now()) {
		return nil, apperrors.NotFound("article not found")
	}
	if err := svcCtx.Store.IncreaseArticleView(ctx, id); err == nil {
		item.ViewCount++
	}
	resp := articleResp(ctx, svcCtx, *item, true)
	return &resp, nil
}

func Preview(ctx context.Context, svcCtx *svc.ServiceContext, id uint64) (*types.ArticleResp, error) {
	userID, role, err := currentActor(ctx)
	if err != nil {
		return nil, err
	}
	item, err := svcCtx.Store.FindArticle(ctx, id)
	if err != nil {
		return nil, logicutil.MapError(err)
	}
	if err := canManageArticle(userID, role, *item); err != nil {
		return nil, err
	}
	resp := articleResp(ctx, svcCtx, *item, true)
	return &resp, nil
}

func Context(ctx context.Context, svcCtx *svc.ServiceContext, id uint64) (*types.ArticleContextResp, error) {
	item, err := svcCtx.Store.FindArticle(ctx, id)
	if err != nil {
		return nil, logicutil.MapError(err)
	}
	if !model.IsArticlePubliclyVisible(*item, time.Now()) {
		return nil, apperrors.NotFound("article not found")
	}
	previous, next, related, err := svcCtx.Store.PublicArticleContext(ctx, *item, 3)
	if err != nil {
		return nil, err
	}
	resp := &types.ArticleContextResp{Related: make([]types.ArticleResp, 0, len(related))}
	if previous != nil {
		prevResp := articleResp(ctx, svcCtx, *previous, false)
		resp.Previous = &prevResp
	}
	if next != nil {
		nextResp := articleResp(ctx, svcCtx, *next, false)
		resp.Next = &nextResp
	}
	for _, relatedItem := range related {
		resp.Related = append(resp.Related, articleResp(ctx, svcCtx, relatedItem, false))
	}
	return resp, nil
}

func Like(ctx context.Context, svcCtx *svc.ServiceContext, id uint64, meta types.RequestMeta) (*types.ArticleLikeResp, error) {
	item, err := svcCtx.Store.FindArticle(ctx, id)
	if err != nil {
		return nil, logicutil.MapError(err)
	}
	if !model.IsArticlePubliclyVisible(*item, time.Now()) {
		return nil, apperrors.NotFound("article not found")
	}
	likeCount, liked, err := svcCtx.Store.LikeArticle(ctx, id, articleLikeVisitorHash(meta.IP, meta.UserAgent))
	if err != nil {
		return nil, err
	}
	return &types.ArticleLikeResp{Liked: liked, LikeCount: likeCount}, nil
}

func Create(ctx context.Context, svcCtx *svc.ServiceContext, req types.ArticleReq, meta types.RequestMeta) (*types.ArticleResp, error) {
	if err := requireArticleCreatePermission(ctx); err != nil {
		return nil, err
	}
	userID, err := authutil.UserID(ctx)
	if err != nil {
		return nil, err
	}
	if req.Status == "" {
		req.Status = model.ArticleStatusDraft
	}
	if err := validateArticle(ctx, svcCtx, req); err != nil {
		return nil, err
	}
	isPinned := false
	if req.IsPinned != nil {
		isPinned = *req.IsPinned
	}
	displayPriority := 0
	if req.DisplayPriority != nil {
		displayPriority = *req.DisplayPriority
	}
	id, err := svcCtx.Store.CreateArticle(ctx, model.ArticleCreate{
		AuthorID:        userID,
		CategoryID:      req.CategoryID,
		Title:           strings.TrimSpace(req.Title),
		Slug:            logicutil.NormalizeSlug(req.Slug),
		Summary:         strings.TrimSpace(req.Summary),
		Content:         req.Content,
		CoverURL:        strings.TrimSpace(req.CoverURL),
		Status:          req.Status,
		ScheduledAt:     normalizeScheduledAt(req.ScheduledAt),
		IsPinned:        isPinned,
		DisplayPriority: displayPriority,
		SEOTitle:        strings.TrimSpace(req.SEOTitle),
		SEODescription:  strings.TrimSpace(req.SEODescription),
		SEOKeywords:     strings.TrimSpace(req.SEOKeywords),
		TagIDs:          req.TagIDs,
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
	isPinned := current.IsPinned
	if req.IsPinned != nil {
		isPinned = *req.IsPinned
	}
	displayPriority := current.DisplayPriority
	if req.DisplayPriority != nil {
		displayPriority = *req.DisplayPriority
	}
	err = svcCtx.Store.UpdateArticle(ctx, id, model.ArticleUpdate{
		CategoryID:      req.CategoryID,
		Title:           strings.TrimSpace(req.Title),
		Slug:            logicutil.NormalizeSlug(req.Slug),
		Summary:         strings.TrimSpace(req.Summary),
		Content:         req.Content,
		CoverURL:        strings.TrimSpace(req.CoverURL),
		Status:          req.Status,
		ScheduledAt:     normalizeScheduledAt(req.ScheduledAt),
		IsPinned:        isPinned,
		DisplayPriority: displayPriority,
		SEOTitle:        strings.TrimSpace(req.SEOTitle),
		SEODescription:  strings.TrimSpace(req.SEODescription),
		SEOKeywords:     strings.TrimSpace(req.SEOKeywords),
		TagIDs:          req.TagIDs,
	}, userID)
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
	if err := svcCtx.Store.UpdateArticleStatus(ctx, id, req.Status, userID); err != nil {
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

func AIAssist(ctx context.Context, svcCtx *svc.ServiceContext, req types.AIAssistReq) (*types.AIAssistResp, error) {
	if err := authutil.RequireContentManager(ctx); err != nil {
		return nil, err
	}
	if !svcCtx.Config.AI.Enabled {
		return nil, apperrors.Forbidden("ai assistant is disabled")
	}
	if strings.TrimSpace(svcCtx.Config.AI.BaseURL) == "" || strings.TrimSpace(svcCtx.Config.AI.APIKey) == "" || strings.TrimSpace(svcCtx.Config.AI.Model) == "" {
		return nil, apperrors.BadRequest("ai assistant is not configured")
	}
	req.Action = strings.TrimSpace(req.Action)
	if _, ok := aiActions[req.Action]; !ok {
		return nil, apperrors.BadRequest("action is invalid")
	}
	if err := validator.Length(strings.TrimSpace(req.Content), "content", 1, 30000); err != nil {
		return nil, err
	}
	if err := validatorOptionalLength(req.Title, "title", 0, 160); err != nil {
		return nil, err
	}
	resp, err := aiclient.Assist(ctx, svcCtx.Config.AI, aiclient.Request{
		Action:  req.Action,
		Title:   req.Title,
		Content: req.Content,
	})
	if err != nil {
		return nil, err
	}
	out := &types.AIAssistResp{
		Summary:        trimRunes(strings.TrimSpace(resp.Summary), 500),
		SEODescription: trimRunes(strings.TrimSpace(resp.SEODescription), 255),
		SEOKeywords:    trimRunes(strings.TrimSpace(resp.SEOKeywords), 255),
		RevisedContent: strings.TrimSpace(resp.RevisedContent),
		Suggestions:    resp.Suggestions,
	}
	if req.Action == "metadata" && out.Summary == "" && out.SEODescription == "" && out.SEOKeywords == "" {
		return nil, apperrors.BadRequest("ai response is invalid")
	}
	if (req.Action == "proofread" || req.Action == "polish") && out.RevisedContent == "" {
		return nil, apperrors.BadRequest("ai response is invalid")
	}
	return out, nil
}

func ListVersions(ctx context.Context, svcCtx *svc.ServiceContext, articleID uint64, page, size int) (*types.ArticleVersionListResp, error) {
	userID, role, err := currentActor(ctx)
	if err != nil {
		return nil, err
	}
	current, err := svcCtx.Store.FindArticle(ctx, articleID)
	if err != nil {
		return nil, logicutil.MapError(err)
	}
	if err := canManageArticle(userID, role, *current); err != nil {
		return nil, err
	}
	page, size = logicutil.Page(page, size)
	items, total, err := svcCtx.Store.ListArticleVersions(ctx, articleID, page, size)
	if err != nil {
		return nil, err
	}
	resp := make([]types.ArticleVersionResp, 0, len(items))
	for _, item := range items {
		resp = append(resp, logicutil.ArticleVersionResp(item, false))
	}
	return &types.ArticleVersionListResp{Items: resp, Total: total, Page: page, Size: size}, nil
}

func VersionDetail(ctx context.Context, svcCtx *svc.ServiceContext, articleID uint64, versionNo int) (*types.ArticleVersionResp, error) {
	userID, role, err := currentActor(ctx)
	if err != nil {
		return nil, err
	}
	current, err := svcCtx.Store.FindArticle(ctx, articleID)
	if err != nil {
		return nil, logicutil.MapError(err)
	}
	if err := canManageArticle(userID, role, *current); err != nil {
		return nil, err
	}
	item, err := svcCtx.Store.FindArticleVersion(ctx, articleID, versionNo)
	if err != nil {
		return nil, logicutil.MapError(err)
	}
	resp := logicutil.ArticleVersionResp(*item, true)
	return &resp, nil
}

func RestoreVersion(ctx context.Context, svcCtx *svc.ServiceContext, articleID uint64, versionNo int, meta types.RequestMeta) (*types.ArticleResp, error) {
	userID, role, err := currentActor(ctx)
	if err != nil {
		return nil, err
	}
	current, err := svcCtx.Store.FindArticle(ctx, articleID)
	if err != nil {
		return nil, logicutil.MapError(err)
	}
	if err := canManageArticle(userID, role, *current); err != nil {
		return nil, err
	}
	if err := svcCtx.Store.RestoreArticleVersion(ctx, articleID, versionNo, userID); err != nil {
		if logicutil.IsDuplicate(err) {
			return nil, apperrors.Conflict("article slug already exists")
		}
		return nil, logicutil.MapError(err)
	}
	item, err := svcCtx.Store.FindArticle(ctx, articleID)
	if err != nil {
		return nil, logicutil.MapError(err)
	}
	publishEvent(ctx, svcCtx, mq.Event{
		UserID:       userID,
		EventType:    "article.version_restored",
		ResourceType: "article",
		ResourceID:   articleID,
		Metadata:     map[string]string{"versionNo": strconv.Itoa(versionNo)},
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
	if err := validator.OptionalHTTPURL(req.CoverURL, "coverUrl"); err != nil {
		return err
	}
	if err := validator.Status(req.Status, statuses, "status"); err != nil {
		return err
	}
	if err := validateDisplayPriority(req.DisplayPriority); err != nil {
		return err
	}
	if err := validatorOptionalLength(req.SEOTitle, "seoTitle", 0, 160); err != nil {
		return err
	}
	if err := validatorOptionalLength(req.SEODescription, "seoDescription", 0, 255); err != nil {
		return err
	}
	if err := validatorOptionalLength(req.SEOKeywords, "seoKeywords", 0, 255); err != nil {
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

func validateDisplayPriority(value *int) error {
	if value == nil {
		return nil
	}
	if *value < 0 || *value > 9999 {
		return apperrors.BadRequest("displayPriority is invalid")
	}
	return nil
}

func validatorOptionalLength(value, field string, min, max int) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return validator.Length(value, field, min, max)
}

func trimRunes(value string, max int) string {
	if max <= 0 {
		return ""
	}
	runes := []rune(value)
	if len(runes) <= max {
		return value
	}
	return string(runes[:max])
}

func normalizeScheduledAt(value *time.Time) *time.Time {
	if value == nil || value.IsZero() {
		return nil
	}
	return value
}

func currentActor(ctx context.Context) (uint64, string, error) {
	userID, err := authutil.UserID(ctx)
	if err != nil {
		return 0, "", err
	}
	return userID, authutil.Role(ctx), nil
}

func requireArticleCreatePermission(ctx context.Context) error {
	return authutil.RequireContentManager(ctx)
}

func canManageArticle(userID uint64, role string, item model.Article) error {
	if authutil.CanManageContent(role) || item.AuthorID == userID {
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

func articleLikeVisitorHash(ip, userAgent string) string {
	sum := sha256.Sum256([]byte("article-like|" + strings.TrimSpace(ip) + "|" + strings.TrimSpace(userAgent)))
	return hex.EncodeToString(sum[:])
}
