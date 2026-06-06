package model

import (
	"context"
	"database/sql"
	"time"
)

type OperationLog struct {
	ID           uint64
	UserID       uint64
	EventType    string
	ResourceType string
	ResourceID   uint64
	Metadata     string
	IP           string
	UserAgent    string
	CreatedAt    time.Time
}

func (s *Store) ListOperationLogs(ctx context.Context, page, size int) ([]OperationLog, int64, error) {
	offset := (page - 1) * size
	var total int64
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM operation_logs").Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT id, user_id, event_type, resource_type, resource_id, COALESCE(CAST(metadata AS CHAR), ''), ip, user_agent, created_at
FROM operation_logs ORDER BY id DESC LIMIT ? OFFSET ?`, size, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items := make([]OperationLog, 0)
	for rows.Next() {
		var item OperationLog
		var userID sql.NullInt64
		var resourceID sql.NullInt64
		if err := rows.Scan(&item.ID, &userID, &item.EventType, &item.ResourceType, &resourceID, &item.Metadata, &item.IP, &item.UserAgent, &item.CreatedAt); err != nil {
			return nil, 0, err
		}
		item.UserID = uint64FromNull(userID)
		item.ResourceID = uint64FromNull(resourceID)
		items = append(items, item)
	}
	return items, total, rows.Err()
}
