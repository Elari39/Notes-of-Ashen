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
	ailogic "notes-of-ashen/internal/logic/ai"
	"notes-of-ashen/internal/logicutil"
	"notes-of-ashen/internal/mq"
	"notes-of-ashen/internal/svc"
	"notes-of-ashen/internal/types"
	"notes-of-ashen/internal/validator"
	"notes-of-ashen/model"

	"github.com/zeromicro/go-zero/core/logx"
)

var statuses = map[string]struct{}{
	model.ArticleStatusDraft:     {},
	model.ArticleStatusPublished: {},
	model.ArticleStatusArchived:  {},
}

// articleLikePerHourLimit 限制单篇文章每小时不同 visitor 的点赞数，防刷赞。
const articleLikePerHourLimit int64 = 500

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
	"expand":    {},
	"shorten":   {},
	"translate": {},
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
	req.Page, req.Size = page, size
	req.Status = status
	req.Query = query
	if filter.UserID == 0 && filter.Role == "" && query != "" {
		if resp, ok := searchPublicArticles(ctx, svcCtx, req, page, size); ok {
			return resp, nil
		}
	}
	if cacheablePublicArticleList(req, filter.Role, filter.UserID, query) {
		cacheKey := publicArticleListCacheKey(req, page, size, status)
		if cached, ok := getCachedArticleList(ctx, svcCtx, cacheKey); ok {
			return cached, nil
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
	resp := articlesResp(ctx, svcCtx, items, false)
	out := &types.ArticleListResp{Items: resp, Total: total, Page: page, Size: size}
	if cacheablePublicArticleList(req, filter.Role, filter.UserID, query) {
		setCachedArticleList(ctx, svcCtx, publicArticleListCacheKey(req, page, size, status), out)
	}
	return out, nil
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
	resp, err := articleResp(ctx, svcCtx, *item, true)
	if err != nil {
		return nil, logicutil.MapError(err)
	}
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
	resp, err := articleResp(ctx, svcCtx, *item, true)
	if err != nil {
		return nil, logicutil.MapError(err)
	}
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
	// 将 prev/next/related 合并批量组装，避免逐篇 articleResp 触发 N+1 查询。
	batchItems := make([]model.Article, 0, 2+len(related))
	if previous != nil {
		batchItems = append(batchItems, *previous)
	}
	if next != nil {
		batchItems = append(batchItems, *next)
	}
	batchItems = append(batchItems, related...)
	batchResp := articlesResp(ctx, svcCtx, batchItems, false)
	resp := &types.ArticleContextResp{Related: make([]types.ArticleResp, 0, len(related))}
	idx := 0
	if previous != nil {
		prevResp := batchResp[idx]
		resp.Previous = &prevResp
		idx++
	}
	if next != nil {
		nextResp := batchResp[idx]
		resp.Next = &nextResp
		idx++
	}
	resp.Related = append(resp.Related, batchResp[idx:]...)
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
	visitorHash := articleLikeVisitorHash(meta.IP, meta.UserAgent, meta.VisitorID)
	// 每篇文章每小时唯一点赞者去重：用 Redis SET 统计不同 visitor_hash 数量，
	// 超阈值视为刷赞并拒绝。Redis 异常时 fail-open，仅记日志不阻断正常点赞。
	if svcCtx.Redis != nil {
		hourKey := "article:like:hour:" + strconv.FormatUint(id, 10) + ":" + strconv.FormatInt(time.Now().Unix()/3600, 10)
		if err := svcCtx.Redis.SAdd(ctx, hourKey, visitorHash).Err(); err != nil {
			logx.Errorf("record article like hour set failed: %v", err)
		} else {
			_ = svcCtx.Redis.Expire(ctx, hourKey, time.Hour).Err()
			if cnt, err := svcCtx.Redis.SCard(ctx, hourKey).Result(); err != nil {
				logx.Errorf("count article like hour set failed: %v", err)
			} else if cnt > articleLikePerHourLimit {
				return nil, apperrors.TooManyRequests("too many likes for this article")
			}
		}
	}
	likeCount, liked, err := svcCtx.Store.LikeArticle(ctx, id, visitorHash)
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
	syncArticleSearch(ctx, svcCtx, id)
	evictArticleCaches(ctx, svcCtx)
	resp, err := articleResp(ctx, svcCtx, *item, true)
	if err != nil {
		return nil, logicutil.MapError(err)
	}
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
	syncArticleSearch(ctx, svcCtx, id)
	evictArticleCaches(ctx, svcCtx)
	resp, err := articleResp(ctx, svcCtx, *item, true)
	if err != nil {
		return nil, logicutil.MapError(err)
	}
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
	deleteArticleSearch(ctx, svcCtx, id)
	evictArticleCaches(ctx, svcCtx)
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
	syncArticleSearch(ctx, svcCtx, id)
	evictArticleCaches(ctx, svcCtx)
	resp, err := articleResp(ctx, svcCtx, *item, true)
	if err != nil {
		return nil, logicutil.MapError(err)
	}
	return &resp, nil
}

func AIAssist(ctx context.Context, svcCtx *svc.ServiceContext, req types.AIAssistReq) (*types.AIAssistResp, error) {
	if err := authutil.RequireContentManager(ctx); err != nil {
		return nil, err
	}
	aiConf, _, err := ailogic.EffectiveConfig(ctx, svcCtx)
	if err != nil {
		return nil, err
	}
	if !aiConf.Enabled {
		return nil, apperrors.Forbidden("ai assistant is disabled")
	}
	if strings.TrimSpace(aiConf.BaseURL) == "" || strings.TrimSpace(aiConf.APIKey) == "" || strings.TrimSpace(aiConf.Model) == "" {
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
	resp, err := aiclient.Assist(ctx, aiConf, aiclient.Request{
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
	if req.Action != "metadata" && out.RevisedContent == "" {
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
	syncArticleSearch(ctx, svcCtx, articleID)
	evictArticleCaches(ctx, svcCtx)
	resp, err := articleResp(ctx, svcCtx, *item, true)
	if err != nil {
		return nil, logicutil.MapError(err)
	}
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

func articleResp(ctx context.Context, svcCtx *svc.ServiceContext, item model.Article, includeContent bool) (types.ArticleResp, error) {
	tags, err := svcCtx.Store.ArticleTags(ctx, item.ID)
	if err != nil {
		return types.ArticleResp{}, err
	}
	var category *model.Category
	if item.CategoryID > 0 {
		category, err = svcCtx.Store.FindCategory(ctx, item.CategoryID)
		if err != nil {
			return types.ArticleResp{}, err
		}
	}
	return logicutil.ArticleResp(item, tags, category, includeContent), nil
}

// articlesResp 批量组装多篇文章响应，一次性加载 tags 与分类，
// 避免列表/搜索/上下文等场景逐篇 articleResp 触发的 N+1 查询。
// 容错语义与单条 articleResp 保持一致：批量查询失败时降级为空 tags/分类，
// 不向调用方返回错误（不引入新的 500），仅补一条日志便于排查。
func articlesResp(ctx context.Context, svcCtx *svc.ServiceContext, items []model.Article, includeContent bool) []types.ArticleResp {
	articleIDs := make([]uint64, 0, len(items))
	categoryIDs := make([]uint64, 0, len(items))
	for _, item := range items {
		articleIDs = append(articleIDs, item.ID)
		if item.CategoryID > 0 {
			categoryIDs = append(categoryIDs, item.CategoryID)
		}
	}

	tagsByArticle, err := svcCtx.Store.ArticleTagsBatch(ctx, articleIDs)
	if err != nil {
		logx.Errorf("batch load article tags failed, fallback to empty: articleCount=%d, err=%v", len(items), err)
		tagsByArticle = map[uint64][]model.Tag{}
	}
	categories, err := svcCtx.Store.FindCategoriesByIDs(ctx, categoryIDs)
	if err != nil {
		logx.Errorf("batch load article categories failed, fallback to empty: categoryCount=%d, err=%v", len(categoryIDs), err)
		categories = map[uint64]model.Category{}
	}

	resp := make([]types.ArticleResp, 0, len(items))
	for _, item := range items {
		var category *model.Category
		if item.CategoryID > 0 {
			if c, ok := categories[item.CategoryID]; ok {
				category = &c
			}
		}
		resp = append(resp, logicutil.ArticleResp(item, tagsByArticle[item.ID], category, includeContent))
	}
	return resp
}

func publishEvent(ctx context.Context, svcCtx *svc.ServiceContext, event mq.Event) {
	if svcCtx.Events != nil {
		svcCtx.Events.Publish(ctx, event)
	}
}

func articleLikeVisitorHash(ip, userAgent, visitorID string) string {
	sum := sha256.Sum256([]byte("article-like|" + strings.TrimSpace(visitorID) + "|" + strings.TrimSpace(ip) + "|" + strings.TrimSpace(userAgent)))
	return hex.EncodeToString(sum[:])
}
