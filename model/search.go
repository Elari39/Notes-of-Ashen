package model

import (
	"context"
	"strings"
	"time"
)

type ArticleSearchDocument struct {
	ID          uint64
	Title       string
	Summary     string
	Content     string
	Status      string
	VisibleAt   int64
	CategoryID  uint64
	Category    string
	TagIDs      []uint64
	Tags        []string
	CreatedAt   int64
	PublishedAt int64
}

func (s *Store) ListArticleSearchDocuments(ctx context.Context) ([]ArticleSearchDocument, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT "+articleSelectFields+" FROM articles WHERE status = 'published'")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	articles := make([]Article, 0)
	for rows.Next() {
		item, err := scanArticleRows(rows)
		if err != nil {
			return nil, err
		}
		articles = append(articles, *item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// 批量加载 tags 与分类，避免逐篇 ArticleTags / FindCategory 的 N+1 查询。
	articleIDs := make([]uint64, 0, len(articles))
	categoryIDs := make([]uint64, 0, len(articles))
	for _, item := range articles {
		articleIDs = append(articleIDs, item.ID)
		if item.CategoryID > 0 {
			categoryIDs = append(categoryIDs, item.CategoryID)
		}
	}
	tagsByArticle, err := s.ArticleTagsBatch(ctx, articleIDs)
	if err != nil {
		return nil, err
	}
	categories, err := s.FindCategoriesByIDs(ctx, categoryIDs)
	if err != nil {
		return nil, err
	}

	docs := make([]ArticleSearchDocument, 0, len(articles))
	for _, item := range articles {
		doc := articleSearchDocumentFromLoaded(item, tagsByArticle[item.ID], categories)
		docs = append(docs, doc)
	}
	return docs, nil
}

func (s *Store) FindArticleSearchDocument(ctx context.Context, id uint64) (*ArticleSearchDocument, error) {
	item, err := s.FindArticle(ctx, id)
	if err != nil {
		return nil, err
	}
	if item.Status != ArticleStatusPublished {
		return nil, ErrNotFound
	}
	return s.articleSearchDocument(ctx, *item)
}

func (s *Store) FindArticlesByIDs(ctx context.Context, ids []uint64) ([]Article, error) {
	ids = uniqueUint64(ids)
	if len(ids) == 0 {
		return []Article{}, nil
	}
	placeholders := strings.TrimRight(strings.Repeat("?,", len(ids)), ",")
	args := make([]interface{}, 0, len(ids))
	for _, id := range ids {
		args = append(args, id)
	}
	rows, err := s.db.QueryContext(ctx, "SELECT "+articleSelectFields+" FROM articles WHERE id IN ("+placeholders+")", args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	byID := make(map[uint64]Article, len(ids))
	for rows.Next() {
		item, err := scanArticleRows(rows)
		if err != nil {
			return nil, err
		}
		byID[item.ID] = *item
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	items := make([]Article, 0, len(ids))
	for _, id := range ids {
		if item, ok := byID[id]; ok {
			items = append(items, item)
		}
	}
	return items, nil
}

func (s *Store) articleSearchDocument(ctx context.Context, item Article) (*ArticleSearchDocument, error) {
	tags, err := s.ArticleTags(ctx, item.ID)
	if err != nil {
		return nil, err
	}
	var category *Category
	if item.CategoryID > 0 {
		category, err = s.FindCategory(ctx, item.CategoryID)
		if err != nil && err != ErrNotFound {
			return nil, err
		}
	}
	return buildArticleSearchDocument(item, tags, category), nil
}

// articleSearchDocumentFromLoaded 在已批量加载 tags（按文章分组）与分类 map 的前提下
// 纯内存组装搜索文档，避免重建期逐篇查询。
func articleSearchDocumentFromLoaded(item Article, tags []Tag, categories map[uint64]Category) ArticleSearchDocument {
	var category *Category
	if item.CategoryID > 0 {
		if c, ok := categories[item.CategoryID]; ok {
			category = &c
		}
	}
	return *buildArticleSearchDocument(item, tags, category)
}

// buildArticleSearchDocument 是 articleSearchDocument 与批量组装共用的纯内存组装逻辑。
func buildArticleSearchDocument(item Article, tags []Tag, category *Category) *ArticleSearchDocument {
	tagNames := make([]string, 0, len(tags))
	tagIDs := make([]uint64, 0, len(tags))
	for _, tag := range tags {
		tagNames = append(tagNames, tag.Name)
		tagIDs = append(tagIDs, tag.ID)
	}
	categoryName := ""
	if category != nil {
		categoryName = category.Name
	}
	visibleAt := item.CreatedAt
	if item.PublishedAt != nil {
		visibleAt = *item.PublishedAt
	}
	if item.ScheduledAt != nil {
		visibleAt = *item.ScheduledAt
	}
	doc := &ArticleSearchDocument{
		ID:         item.ID,
		Title:      item.Title,
		Summary:    item.Summary,
		Content:    item.Content,
		Status:     item.Status,
		VisibleAt:  unixOrZero(visibleAt),
		CategoryID: item.CategoryID,
		Category:   categoryName,
		TagIDs:     tagIDs,
		Tags:       tagNames,
		CreatedAt:  unixOrZero(item.CreatedAt),
	}
	if item.PublishedAt != nil {
		doc.PublishedAt = unixOrZero(*item.PublishedAt)
	}
	return doc
}

func unixOrZero(value time.Time) int64 {
	if value.IsZero() {
		return 0
	}
	return value.Unix()
}
