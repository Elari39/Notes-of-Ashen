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
