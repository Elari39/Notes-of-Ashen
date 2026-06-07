package model

import "context"

type AdminStats struct {
	ArticleTotal   int64
	PublishedTotal int64
	DraftTotal     int64
	ArchivedTotal  int64
	ScheduledTotal int64
	ViewTotal      uint64
	UserTotal      int64
	CategoryTotal  int64
	TagTotal       int64
}

func (s *Store) AdminStats(ctx context.Context) (*AdminStats, error) {
	var stats AdminStats
	if err := s.db.QueryRowContext(ctx, `
SELECT
  COUNT(*),
  COALESCE(SUM(CASE WHEN status = 'published' AND (scheduled_at IS NULL OR scheduled_at <= NOW()) THEN 1 ELSE 0 END), 0),
  COALESCE(SUM(CASE WHEN status = 'draft' THEN 1 ELSE 0 END), 0),
  COALESCE(SUM(CASE WHEN status = 'archived' THEN 1 ELSE 0 END), 0),
  COALESCE(SUM(CASE WHEN status = 'published' AND scheduled_at > NOW() THEN 1 ELSE 0 END), 0),
  COALESCE(SUM(view_count), 0)
FROM articles`).Scan(&stats.ArticleTotal, &stats.PublishedTotal, &stats.DraftTotal, &stats.ArchivedTotal, &stats.ScheduledTotal, &stats.ViewTotal); err != nil {
		return nil, err
	}
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM users").Scan(&stats.UserTotal); err != nil {
		return nil, err
	}
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM categories").Scan(&stats.CategoryTotal); err != nil {
		return nil, err
	}
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM tags").Scan(&stats.TagTotal); err != nil {
		return nil, err
	}
	return &stats, nil
}

func (s *Store) PopularArticles(ctx context.Context, limit int) ([]Article, error) {
	if limit < 1 {
		limit = 5
	}
	rows, err := s.db.QueryContext(ctx, "SELECT "+articleSelectFields+" FROM articles ORDER BY view_count DESC, id DESC LIMIT ?", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]Article, 0)
	for rows.Next() {
		item, err := scanArticleRows(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	return items, rows.Err()
}

func (s *Store) RecentArticles(ctx context.Context, limit int) ([]Article, error) {
	if limit < 1 {
		limit = 5
	}
	rows, err := s.db.QueryContext(ctx, "SELECT "+articleSelectFields+" FROM articles ORDER BY "+articleTimeOrder+" LIMIT ?", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]Article, 0)
	for rows.Next() {
		item, err := scanArticleRows(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	return items, rows.Err()
}
