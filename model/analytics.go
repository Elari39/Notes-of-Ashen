package model

import (
	"context"
	"database/sql"
)

type AnalyticsSummary struct {
	PV    int64
	UV    int64
	Likes int64
}

type PageAnalytics struct {
	RouteType string
	Path      string
	ArticleID uint64
	Title     string
	PV        int64
	UV        int64
}

type ArticleAnalytics struct {
	ArticleID  uint64
	Title      string
	Status     string
	PV         int64
	UV         int64
	Likes      int64
	TotalViews uint64
	TotalLikes uint64
}

type ArticleAnalyticsPoint struct {
	Date  string
	PV    int64
	UV    int64
	Likes int64
}

func (s *Store) AnalyticsSummary(ctx context.Context, from, to string) (AnalyticsSummary, error) {
	var item AnalyticsSummary
	if err := s.db.QueryRowContext(ctx, `
SELECT COALESCE(SUM(pv), 0), COALESCE(SUM(uv), 0)
FROM traffic_daily_stats WHERE stat_date BETWEEN ? AND ?`, from, to).Scan(&item.PV, &item.UV); err != nil {
		return item, err
	}
	err := s.db.QueryRowContext(ctx, `
SELECT COUNT(*) FROM article_likes WHERE DATE(created_at) BETWEEN ? AND ?`, from, to).Scan(&item.Likes)
	return item, err
}

func (s *Store) TrafficTrendRange(ctx context.Context, from, to string) ([]TrafficTrendPoint, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT DATE_FORMAT(stat_date, '%Y-%m-%d'), pv, uv
FROM traffic_daily_stats WHERE stat_date BETWEEN ? AND ? ORDER BY stat_date`, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]TrafficTrendPoint, 0)
	for rows.Next() {
		var item TrafficTrendPoint
		if err := rows.Scan(&item.Date, &item.PV, &item.UV); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) TopPagesRange(ctx context.Context, from, to string, limit int) ([]PageAnalytics, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT s.route_type, s.path, s.article_id, COALESCE(a.title, ''), SUM(s.pv), SUM(s.uv)
FROM traffic_content_daily_stats s
LEFT JOIN articles a ON a.id = s.article_id
WHERE s.stat_date BETWEEN ? AND ?
GROUP BY s.route_type, s.path, s.article_id, a.title
ORDER BY SUM(s.pv) DESC, s.path LIMIT ?`, from, to, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]PageAnalytics, 0, limit)
	for rows.Next() {
		var item PageAnalytics
		if err := rows.Scan(&item.RouteType, &item.Path, &item.ArticleID, &item.Title, &item.PV, &item.UV); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) TopReferersRange(ctx context.Context, from, to string, articleID uint64, limit int) ([]RefererStat, error) {
	where := "stat_date BETWEEN ? AND ?"
	args := []interface{}{from, to}
	if articleID > 0 {
		where += " AND article_id = ?"
		args = append(args, articleID)
	}
	args = append(args, limit)
	rows, err := s.db.QueryContext(ctx, `
SELECT source_type, source_name, SUM(pv)
FROM traffic_referer_stats WHERE `+where+`
GROUP BY source_type, source_name ORDER BY SUM(pv) DESC, source_name LIMIT ?`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]RefererStat, 0, limit)
	for rows.Next() {
		var item RefererStat
		if err := rows.Scan(&item.SourceType, &item.SourceName, &item.PV); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) ListArticleAnalytics(ctx context.Context, from, to, query string, page, size int) ([]ArticleAnalytics, int64, error) {
	where := ""
	whereArgs := make([]interface{}, 0, 1)
	if query != "" {
		where = " WHERE a.title LIKE ?"
		whereArgs = append(whereArgs, "%"+query+"%")
	}
	var total int64
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM articles a"+where, whereArgs...).Scan(&total); err != nil {
		return nil, 0, err
	}
	args := []interface{}{from, to, from, to}
	args = append(args, whereArgs...)
	args = append(args, size, (page-1)*size)
	rows, err := s.db.QueryContext(ctx, `
SELECT a.id, a.title, a.status, COALESCE(SUM(s.pv),0), COALESCE(SUM(s.uv),0),
       (SELECT COUNT(*) FROM article_likes al WHERE al.article_id=a.id AND DATE(al.created_at) BETWEEN ? AND ?),
       a.view_count, a.like_count
FROM articles a
LEFT JOIN traffic_content_daily_stats s ON s.article_id=a.id AND s.stat_date BETWEEN ? AND ?`+where+`
GROUP BY a.id, a.title, a.status, a.view_count, a.like_count
ORDER BY COALESCE(SUM(s.pv),0) DESC, a.id DESC LIMIT ? OFFSET ?`, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := make([]ArticleAnalytics, 0, size)
	for rows.Next() {
		var item ArticleAnalytics
		if err := rows.Scan(&item.ArticleID, &item.Title, &item.Status, &item.PV, &item.UV, &item.Likes, &item.TotalViews, &item.TotalLikes); err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	return items, total, rows.Err()
}

func (s *Store) ArticleAnalyticsDetail(ctx context.Context, articleID uint64, from, to string) (ArticleAnalytics, []ArticleAnalyticsPoint, error) {
	var article ArticleAnalytics
	if err := s.db.QueryRowContext(ctx, `
SELECT id, title, status, view_count, like_count FROM articles WHERE id = ?`, articleID).
		Scan(&article.ArticleID, &article.Title, &article.Status, &article.TotalViews, &article.TotalLikes); err != nil {
		return article, nil, scanErr(err)
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT DATE_FORMAT(s.stat_date, '%Y-%m-%d'), SUM(s.pv), SUM(s.uv),
       (SELECT COUNT(*) FROM article_likes al WHERE al.article_id=? AND DATE(al.created_at)=s.stat_date)
FROM traffic_content_daily_stats s
WHERE s.article_id=? AND s.stat_date BETWEEN ? AND ?
GROUP BY s.stat_date ORDER BY s.stat_date`, articleID, articleID, from, to)
	if err != nil {
		return article, nil, err
	}
	defer rows.Close()
	items := make([]ArticleAnalyticsPoint, 0)
	for rows.Next() {
		var item ArticleAnalyticsPoint
		if err := rows.Scan(&item.Date, &item.PV, &item.UV, &item.Likes); err != nil {
			return article, nil, err
		}
		article.PV += item.PV
		article.UV += item.UV
		article.Likes += item.Likes
		items = append(items, item)
	}
	return article, items, rows.Err()
}

func (s *Store) CleanupTrafficVisitors(ctx context.Context, before string) error {
	return WithTx(ctx, s.db, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, "DELETE FROM traffic_daily_visitors WHERE stat_date < ?", before); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, "DELETE FROM traffic_content_daily_visitors WHERE stat_date < ?", before)
		return err
	})
}
