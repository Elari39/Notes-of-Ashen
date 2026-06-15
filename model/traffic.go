package model

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

type TrafficRecord struct {
	Date        string
	VisitorHash string
	ArticleID   uint64
	SourceType  string
	SourceName  string
}

type TrafficTrendPoint struct {
	Date string
	PV   int64
	UV   int64
}

type RefererStat struct {
	SourceType string
	SourceName string
	PV         int64
}

func (s *Store) RecordTraffic(ctx context.Context, in TrafficRecord) error {
	return WithTx(ctx, s.db, func(tx *sql.Tx) error {
		uvIncrement := int64(0)
		res, err := tx.ExecContext(ctx, `
INSERT IGNORE INTO traffic_daily_visitors (stat_date, visitor_hash)
VALUES (?, ?)`, in.Date, in.VisitorHash)
		if err != nil {
			return err
		}
		affected, err := res.RowsAffected()
		if err != nil {
			return err
		}
		if affected > 0 {
			uvIncrement = 1
		}

		if _, err := tx.ExecContext(ctx, `
INSERT INTO traffic_daily_stats (stat_date, pv, uv)
VALUES (?, 1, ?)
ON DUPLICATE KEY UPDATE pv = pv + 1, uv = uv + VALUES(uv)`, in.Date, uvIncrement); err != nil {
			return err
		}

		_, err = tx.ExecContext(ctx, `
INSERT INTO traffic_referer_stats (stat_date, article_id, source_type, source_name, pv)
VALUES (?, ?, ?, ?, 1)
ON DUPLICATE KEY UPDATE pv = pv + 1`, in.Date, in.ArticleID, in.SourceType, in.SourceName)
		return err
	})
}

func (s *Store) TodayTraffic(ctx context.Context, date string) (TrafficTrendPoint, error) {
	var item TrafficTrendPoint
	item.Date = date
	err := s.db.QueryRowContext(ctx, `
SELECT COALESCE(pv, 0), COALESCE(uv, 0)
FROM traffic_daily_stats
WHERE stat_date = ?`, date).Scan(&item.PV, &item.UV)
	if err != nil {
		if errors.Is(scanErr(err), ErrNotFound) {
			return item, nil
		}
		return item, scanErr(err)
	}
	return item, nil
}

func (s *Store) TrafficTrend(ctx context.Context, days int) ([]TrafficTrendPoint, error) {
	if days < 1 {
		days = 30
	}
	if days > 90 {
		days = 90
	}
	now := time.Now()
	start := now.AddDate(0, 0, -(days - 1)).Format("2006-01-02")
	end := now.Format("2006-01-02")

	rows, err := s.db.QueryContext(ctx, `
SELECT DATE_FORMAT(stat_date, '%Y-%m-%d'), pv, uv
FROM traffic_daily_stats
WHERE stat_date BETWEEN ? AND ?
ORDER BY stat_date`, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	found := make(map[string]TrafficTrendPoint, days)
	for rows.Next() {
		var item TrafficTrendPoint
		if err := rows.Scan(&item.Date, &item.PV, &item.UV); err != nil {
			return nil, err
		}
		found[item.Date] = item
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	points := make([]TrafficTrendPoint, 0, days)
	for i := days - 1; i >= 0; i-- {
		date := now.AddDate(0, 0, -i).Format("2006-01-02")
		if item, ok := found[date]; ok {
			points = append(points, item)
			continue
		}
		points = append(points, TrafficTrendPoint{Date: date})
	}
	return points, nil
}

func (s *Store) TopReferers(ctx context.Context, days, limit int) ([]RefererStat, error) {
	if days < 1 {
		days = 30
	}
	if limit < 1 {
		limit = 8
	}
	if limit > 20 {
		limit = 20
	}
	start := time.Now().AddDate(0, 0, -(days - 1)).Format("2006-01-02")
	rows, err := s.db.QueryContext(ctx, `
SELECT source_type, source_name, SUM(pv) AS pv
FROM traffic_referer_stats
WHERE stat_date >= ?
GROUP BY source_type, source_name
ORDER BY pv DESC, source_type, source_name
LIMIT ?`, start, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]RefererStat, 0)
	for rows.Next() {
		var item RefererStat
		if err := rows.Scan(&item.SourceType, &item.SourceName, &item.PV); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
