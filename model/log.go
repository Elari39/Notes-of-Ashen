package model

import (
	"context"
	"database/sql"
	"time"
)

type OperationLog struct {
	ID           uint64
	UserID       uint64
	UserAccount  string
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
SELECT o.id, o.user_id, COALESCE(u.account, ''), o.event_type, o.resource_type, o.resource_id,
       COALESCE(CAST(o.metadata AS CHAR), ''), o.ip, o.user_agent, o.created_at
FROM operation_logs o
LEFT JOIN users u ON o.user_id = u.id
ORDER BY o.id DESC LIMIT ? OFFSET ?`, size, offset)
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
	if err := scanner.Scan(&item.ID, &userID, &item.UserAccount, &item.EventType, &item.ResourceType, &resourceID, &item.Metadata, &ip, &userAgent, &item.CreatedAt); err != nil {
		return OperationLog{}, err
	}
	item.UserID = uint64FromNull(userID)
	item.ResourceID = uint64FromNull(resourceID)
	item.IP = stringFromNull(ip)
	item.UserAgent = stringFromNull(userAgent)
	return item, nil
}
