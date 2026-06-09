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
		item, err := scanOperationLog(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	return items, total, rows.Err()
}

type operationLogScanner interface {
	Scan(dest ...interface{}) error
}

func scanOperationLog(scanner operationLogScanner) (OperationLog, error) {
	var item OperationLog
	var userID sql.NullInt64
	var resourceID sql.NullInt64
	var ip sql.NullString
	var userAgent sql.NullString
	if err := scanner.Scan(&item.ID, &userID, &item.EventType, &item.ResourceType, &resourceID, &item.Metadata, &ip, &userAgent, &item.CreatedAt); err != nil {
		return OperationLog{}, err
	}
	item.UserID = uint64FromNull(userID)
	item.ResourceID = uint64FromNull(resourceID)
	item.IP = stringFromNull(ip)
	item.UserAgent = stringFromNull(userAgent)
	return item, nil
}
