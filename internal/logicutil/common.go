package logicutil

import (
	"errors"
	"strings"
	"time"

	"notes-of-ashen/internal/contentstats"
	apperrors "notes-of-ashen/internal/errors"
	"notes-of-ashen/internal/types"
	"notes-of-ashen/model"

	"github.com/go-sql-driver/mysql"
)

const MaxPage = 1000

func Page(page, size int) (int, int) {
	if page < 1 {
		page = 1
	}
	if page > MaxPage {
		page = MaxPage
	}
	if size < 1 {
		size = 10
	}
	if size > 100 {
		size = 100
	}
	return page, size
}

func RegistrationEmailCodeRequired(isFirstUser bool, emailEnabled bool) bool {
	return !isFirstUser || emailEnabled
}

func MapError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, model.ErrNotFound) {
		return apperrors.NotFound("resource not found")
	}
	var mysqlErr *mysql.MySQLError
	if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
		return apperrors.Conflict("resource already exists")
	}
	return err
}

func IsDuplicate(err error) bool {
	var mysqlErr *mysql.MySQLError
	return errors.As(err, &mysqlErr) && mysqlErr.Number == 1062
}

// nonNilUint64Slice 保证 nil 切片返回空切片，避免 JSON omitempty 省略字段
// 导致前端非可选数组类型收到 undefined。
func nonNilUint64Slice(v []uint64) []uint64 {
	if v == nil {
		return []uint64{}
	}
	return v
}

func UserResp(u model.User) types.UserResp {
	return types.UserResp{
		ID:        u.ID,
		Account:   u.Account,
		Email:     u.Email,
		AvatarURL: u.AvatarURL,
		Nickname:  u.Nickname,
		Role:      u.Role,
		Status:    u.Status,
		CreatedAt: utcTime(u.CreatedAt),
		UpdatedAt: utcTime(u.UpdatedAt),
	}
}

func CategoryResp(c model.Category) types.CategoryResp {
	return types.CategoryResp{
		ID:           c.ID,
		Name:         c.Name,
		Slug:         c.Slug,
		Description:  c.Description,
		CreatedBy:    c.CreatedBy,
		CreatedAt:    utcTime(c.CreatedAt),
		UpdatedAt:    utcTime(c.UpdatedAt),
		ArticleCount: c.ArticleCount,
	}
}

func TagResp(t model.Tag) types.TagResp {
	return types.TagResp{
		ID:           t.ID,
		Name:         t.Name,
		Slug:         t.Slug,
		Description:  t.Description,
		CreatedBy:    t.CreatedBy,
		CreatedAt:    utcTime(t.CreatedAt),
		UpdatedAt:    utcTime(t.UpdatedAt),
		ArticleCount: t.ArticleCount,
	}
}

func ArticleResp(a model.Article, tags []model.Tag, category *model.Category, includeContent bool) types.ArticleResp {
	stats := contentstats.Analyze(a.Content)
	resp := types.ArticleResp{
		ID:                 a.ID,
		AuthorID:           a.AuthorID,
		CategoryID:         a.CategoryID,
		Title:              a.Title,
		Slug:               a.Slug,
		Summary:            a.Summary,
		CoverURL:           a.CoverURL,
		Status:             a.Status,
		ViewCount:          a.ViewCount,
		LikeCount:          a.LikeCount,
		WordCount:          stats.WordCount,
		ReadingTimeMinutes: stats.ReadingTimeMinutes,
		ScheduledAt:        utcTimePtr(a.ScheduledAt),
		PublishedAt:        utcTimePtr(a.PublishedAt),
		IsPinned:           a.IsPinned,
		DisplayPriority:    a.DisplayPriority,
		SEOTitle:           a.SEOTitle,
		SEODescription:     a.SEODescription,
		SEOKeywords:        a.SEOKeywords,
		CreatedAt:          utcTime(a.CreatedAt),
		UpdatedAt:          utcTime(a.UpdatedAt),
	}
	if includeContent {
		resp.Content = a.Content
	}
	if category != nil {
		categoryResp := CategoryResp(*category)
		resp.Category = &categoryResp
	}
	resp.Tags = make([]types.TagResp, 0, len(tags))
	for _, tag := range tags {
		resp.Tags = append(resp.Tags, TagResp(tag))
	}
	return resp
}

func ArticleVersionResp(v model.ArticleVersion, includeContent bool) types.ArticleVersionResp {
	resp := types.ArticleVersionResp{
		ID:                v.ID,
		ArticleID:         v.ArticleID,
		VersionNo:         v.VersionNo,
		ChangedBy:         v.ChangedBy,
		AuthorID:          v.AuthorID,
		CategoryID:        v.CategoryID,
		Title:             v.Title,
		Slug:              v.Slug,
		Summary:           v.Summary,
		CoverURL:          v.CoverURL,
		Status:            v.Status,
		ViewCount:         v.ViewCount,
		LikeCount:         v.LikeCount,
		ScheduledAt:       utcTimePtr(v.ScheduledAt),
		PublishedAt:       utcTimePtr(v.PublishedAt),
		IsPinned:          v.IsPinned,
		DisplayPriority:   v.DisplayPriority,
		SEOTitle:          v.SEOTitle,
		SEODescription:    v.SEODescription,
		SEOKeywords:       v.SEOKeywords,
		TagIDs:            nonNilUint64Slice(v.TagIDs),
		OriginalCreatedAt: utcTimePtr(v.OriginalCreatedAt),
		OriginalUpdatedAt: utcTimePtr(v.OriginalUpdatedAt),
		CreatedAt:         utcTime(v.CreatedAt),
	}
	if includeContent {
		resp.Content = v.Content
	}
	return resp
}

func OperationLogResp(l model.OperationLog) types.OperationLogResp {
	return types.OperationLogResp{
		ID:           l.ID,
		UserID:       l.UserID,
		UserAccount:  l.UserAccount,
		EventType:    l.EventType,
		ResourceType: l.ResourceType,
		ResourceID:   l.ResourceID,
		Metadata:     l.Metadata,
		IP:           l.IP,
		UserAgent:    l.UserAgent,
		CreatedAt:    utcTime(l.CreatedAt),
	}
}

// utcTime 将 time.Time 归一化为 UTC，确保 JSON 序列化带 Z 后缀，
// 避免前端跨时区/旧浏览器对 +08:00 偏移字符串解析不一致。
func utcTime(t time.Time) time.Time {
	return t.UTC()
}

// utcTimePtr 将 *time.Time 归一化为 UTC，nil 原样返回（保留 omitempty 语义）。
func utcTimePtr(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}
	utc := t.UTC()
	return &utc
}

func NormalizeSlug(slug string) string {
	return strings.ToLower(strings.TrimSpace(slug))
}

func TodayDate() string {
	return time.Now().Format("2006-01-02")
}
