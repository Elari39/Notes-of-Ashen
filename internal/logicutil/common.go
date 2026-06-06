package logicutil

import (
	"errors"
	"strings"

	apperrors "notes-of-ashen/internal/errors"
	"notes-of-ashen/internal/types"
	"notes-of-ashen/model"

	"github.com/go-sql-driver/mysql"
)

func Page(page, size int) (int, int) {
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 10
	}
	if size > 100 {
		size = 100
	}
	return page, size
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

func UserResp(u model.User) types.UserResp {
	return types.UserResp{
		ID:        u.ID,
		Account:   u.Account,
		Email:     u.Email,
		AvatarURL: u.AvatarURL,
		Nickname:  u.Nickname,
		Role:      u.Role,
		Status:    u.Status,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
}

func CategoryResp(c model.Category) types.CategoryResp {
	return types.CategoryResp{
		ID:          c.ID,
		Name:        c.Name,
		Slug:        c.Slug,
		Description: c.Description,
		CreatedBy:   c.CreatedBy,
		CreatedAt:   c.CreatedAt,
		UpdatedAt:   c.UpdatedAt,
	}
}

func TagResp(t model.Tag) types.TagResp {
	return types.TagResp{
		ID:          t.ID,
		Name:        t.Name,
		Slug:        t.Slug,
		Description: t.Description,
		CreatedBy:   t.CreatedBy,
		CreatedAt:   t.CreatedAt,
		UpdatedAt:   t.UpdatedAt,
	}
}

func ArticleResp(a model.Article, tags []model.Tag, category *model.Category, includeContent bool) types.ArticleResp {
	resp := types.ArticleResp{
		ID:             a.ID,
		AuthorID:       a.AuthorID,
		CategoryID:     a.CategoryID,
		Title:          a.Title,
		Slug:           a.Slug,
		Summary:        a.Summary,
		CoverURL:       a.CoverURL,
		Status:         a.Status,
		ViewCount:      a.ViewCount,
		ScheduledAt:    a.ScheduledAt,
		PublishedAt:    a.PublishedAt,
		SEOTitle:       a.SEOTitle,
		SEODescription: a.SEODescription,
		SEOKeywords:    a.SEOKeywords,
		CreatedAt:      a.CreatedAt,
		UpdatedAt:      a.UpdatedAt,
	}
	if includeContent {
		resp.Content = a.Content
	}
	if category != nil {
		categoryResp := CategoryResp(*category)
		resp.Category = &categoryResp
	}
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
		ScheduledAt:       v.ScheduledAt,
		PublishedAt:       v.PublishedAt,
		SEOTitle:          v.SEOTitle,
		SEODescription:    v.SEODescription,
		SEOKeywords:       v.SEOKeywords,
		TagIDs:            v.TagIDs,
		OriginalCreatedAt: v.OriginalCreatedAt,
		OriginalUpdatedAt: v.OriginalUpdatedAt,
		CreatedAt:         v.CreatedAt,
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
		EventType:    l.EventType,
		ResourceType: l.ResourceType,
		ResourceID:   l.ResourceID,
		Metadata:     l.Metadata,
		IP:           l.IP,
		UserAgent:    l.UserAgent,
		CreatedAt:    l.CreatedAt,
	}
}

func NormalizeSlug(slug string) string {
	return strings.ToLower(strings.TrimSpace(slug))
}
