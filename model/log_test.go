package model

import (
	"database/sql"
	"testing"
	"time"
)

type fakeOperationLogScanner struct {
	createdAt time.Time
}

func (f fakeOperationLogScanner) Scan(dest ...interface{}) error {
	*dest[0].(*uint64) = 7
	*dest[1].(*sql.NullInt64) = sql.NullInt64{}
	*dest[2].(*string) = "alice"
	*dest[3].(*string) = "article.created"
	*dest[4].(*string) = "article"
	*dest[5].(*sql.NullInt64) = sql.NullInt64{Int64: 12, Valid: true}
	*dest[6].(*string) = "{}"
	*dest[7].(*sql.NullString) = sql.NullString{}
	*dest[8].(*sql.NullString) = sql.NullString{}
	*dest[9].(*time.Time) = f.createdAt
	return nil
}

func TestScanOperationLogAllowsNullNetworkFields(t *testing.T) {
	createdAt := time.Date(2026, 6, 9, 12, 0, 0, 0, time.UTC)
	item, err := scanOperationLog(fakeOperationLogScanner{createdAt: createdAt})
	if err != nil {
		t.Fatalf("scanOperationLog() error = %v", err)
	}
	if item.IP != "" || item.UserAgent != "" {
		t.Fatalf("IP/UserAgent = %q/%q, want empty strings", item.IP, item.UserAgent)
	}
	if item.UserID != 0 {
		t.Fatalf("UserID = %d, want 0", item.UserID)
	}
	if item.ResourceID != 12 {
		t.Fatalf("ResourceID = %d, want 12", item.ResourceID)
	}
	if !item.CreatedAt.Equal(createdAt) {
		t.Fatalf("CreatedAt = %v, want %v", item.CreatedAt, createdAt)
	}
}

func TestOperationLogWhereBuildsCombinedFilters(t *testing.T) {
	startAt := time.Date(2026, 7, 14, 16, 0, 0, 0, time.UTC)
	endAt := time.Date(2026, 7, 15, 16, 0, 0, 0, time.UTC)
	where, args := operationLogWhere(OperationLogFilter{
		EventType:   "article.updated",
		UserAccount: "ash%_!",
		IP:          "203.0.113.8",
		StartAt:     &startAt,
		EndAt:       &endAt,
	})
	wantWhere := "WHERE o.event_type = ? AND u.account LIKE ? ESCAPE '!' AND o.ip = ? AND o.created_at >= ? AND o.created_at < ?"
	if where != wantWhere {
		t.Fatalf("operationLogWhere() where = %q, want %q", where, wantWhere)
	}
	if len(args) != 5 {
		t.Fatalf("operationLogWhere() args length = %d, want 5", len(args))
	}
	if args[0] != "article.updated" || args[1] != "%ash!%!_!!%" || args[2] != "203.0.113.8" {
		t.Fatalf("operationLogWhere() string args = %#v", args[:3])
	}
	if !args[3].(time.Time).Equal(startAt) || !args[4].(time.Time).Equal(endAt) {
		t.Fatalf("operationLogWhere() time args = %#v", args[3:])
	}
}

func TestOperationLogWherePrefersUserID(t *testing.T) {
	where, args := operationLogWhere(OperationLogFilter{UserID: 42, UserAccount: "ignored"})
	if where != "WHERE o.user_id = ?" || len(args) != 1 || args[0] != uint64(42) {
		t.Fatalf("operationLogWhere() = %q, %#v", where, args)
	}
}

func TestOperationLogWhereAllowsNoFilters(t *testing.T) {
	where, args := operationLogWhere(OperationLogFilter{})
	if where != "" || len(args) != 0 {
		t.Fatalf("operationLogWhere() = %q, %#v, want empty", where, args)
	}
}
