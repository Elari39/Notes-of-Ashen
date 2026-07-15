package model

import (
	"context"
	"database/sql"
	"strings"
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

type OperationLogFilter struct {
	Page        int
	Size        int
	EventType   string
	UserID      uint64
	UserAccount string
	IP          string
	StartAt     *time.Time
	EndAt       *time.Time
}

func (s *Store) ListOperationLogs(ctx context.Context, page, size int) ([]OperationLog, int64, error) {
	return s.ListOperationLogsFiltered(ctx, OperationLogFilter{Page: page, Size: size})
}

func (s *Store) ListOperationLogsFiltered(ctx context.Context, filter OperationLogFilter) ([]OperationLog, int64, error) {
	where, args := operationLogWhere(filter)
	countBaseSQL := " FROM operation_logs o "
	if filter.UserAccount != "" {
		countBaseSQL += "LEFT JOIN users u ON o.user_id = u.id "
	}
	var total int64
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*)"+countBaseSQL+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	offset := (filter.Page - 1) * filter.Size
	queryArgs := append(append([]interface{}{}, args...), filter.Size, offset)
	selectBaseSQL := " FROM operation_logs o LEFT JOIN users u ON o.user_id = u.id " + where
	rows, err := s.db.QueryContext(ctx, `
SELECT o.id, o.user_id, COALESCE(u.account, ''), o.event_type, o.resource_type, o.resource_id,
       COALESCE(CAST(o.metadata AS CHAR), ''), o.ip, o.user_agent, o.created_at
`+selectBaseSQL+`
ORDER BY o.id DESC LIMIT ? OFFSET ?`, queryArgs...)
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

func operationLogWhere(filter OperationLogFilter) (string, []interface{}) {
	clauses := make([]string, 0, 6)
	args := make([]interface{}, 0, 6)
	if filter.EventType != "" {
		clauses = append(clauses, "o.event_type = ?")
		args = append(args, filter.EventType)
	}
	if filter.UserID > 0 {
		clauses = append(clauses, "o.user_id = ?")
		args = append(args, filter.UserID)
	} else if filter.UserAccount != "" {
		clauses = append(clauses, "u.account LIKE ? ESCAPE '!'")
		args = append(args, "%"+escapeLike(filter.UserAccount)+"%")
	}
	if filter.IP != "" {
		clauses = append(clauses, "o.ip = ?")
		args = append(args, filter.IP)
	}
	if filter.StartAt != nil {
		clauses = append(clauses, "o.created_at >= ?")
		args = append(args, *filter.StartAt)
	}
	if filter.EndAt != nil {
		clauses = append(clauses, "o.created_at < ?")
		args = append(args, *filter.EndAt)
	}
	if len(clauses) == 0 {
		return "", args
	}
	return "WHERE " + strings.Join(clauses, " AND "), args
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
