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

	docs := make([]ArticleSearchDocument, 0)
	for rows.Next() {
		item, err := scanArticleRows(rows)
		if err != nil {
			return nil, err
		}
		doc, err := s.articleSearchDocument(ctx, *item)
		if err != nil {
			return nil, err
		}
		docs = append(docs, *doc)
	}
	return docs, rows.Err()
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
	tagNames := make([]string, 0, len(tags))
	tagIDs := make([]uint64, 0, len(tags))
	for _, tag := range tags {
		tagNames = append(tagNames, tag.Name)
		tagIDs = append(tagIDs, tag.ID)
	}
	categoryName := ""
	if item.CategoryID > 0 {
		category, err := s.FindCategory(ctx, item.CategoryID)
		if err != nil && err != ErrNotFound {
			return nil, err
		}
		if category != nil {
			categoryName = category.Name
		}
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
	return doc, nil
}

func unixOrZero(value time.Time) int64 {
	if value.IsZero() {
		return 0
	}
	return value.Unix()
}
