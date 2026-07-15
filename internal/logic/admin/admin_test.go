package admin

import (
	"strings"
	"testing"
	"time"

	"notes-of-ashen/internal/types"
)

func TestOperationLogFilterNormalizesValues(t *testing.T) {
	filter, err := operationLogFilter(types.OperationLogListReq{
		EventType: " article.updated ",
		Actor:     "42",
		IP:        "203.0.113.8",
		StartAt:   "2026-07-15T00:00:00+08:00",
		EndAt:     "2026-07-16T00:00:00+08:00",
	})
	if err != nil {
		t.Fatalf("operationLogFilter() error = %v", err)
	}
	if filter.Page != 1 || filter.Size != 10 {
		t.Fatalf("page/size = %d/%d, want 1/10", filter.Page, filter.Size)
	}
	if filter.EventType != "article.updated" || filter.UserID != 42 || filter.UserAccount != "" {
		t.Fatalf("event/actor filter = %#v", filter)
	}
	wantStart := time.Date(2026, 7, 14, 16, 0, 0, 0, time.UTC)
	if filter.StartAt == nil || !filter.StartAt.Equal(wantStart) {
		t.Fatalf("startAt = %v, want %v", filter.StartAt, wantStart)
	}
}

func TestOperationLogFilterUsesAccountForNonNumericActor(t *testing.T) {
	filter, err := operationLogFilter(types.OperationLogListReq{Actor: " ashen_admin "})
	if err != nil {
		t.Fatalf("operationLogFilter() error = %v", err)
	}
	if filter.UserID != 0 || filter.UserAccount != "ashen_admin" {
		t.Fatalf("actor filter = %#v", filter)
	}
}

func TestOperationLogFilterRejectsInvalidValues(t *testing.T) {
	tests := []struct {
		name string
		req  types.OperationLogListReq
	}{
		{name: "invalid ip", req: types.OperationLogListReq{IP: "not-an-ip"}},
		{name: "invalid start", req: types.OperationLogListReq{StartAt: "2026-07-15"}},
		{name: "invalid range", req: types.OperationLogListReq{StartAt: "2026-07-16T00:00:00Z", EndAt: "2026-07-15T00:00:00Z"}},
		{name: "zero user id", req: types.OperationLogListReq{Actor: "0"}},
		{name: "overflow user id", req: types.OperationLogListReq{Actor: "18446744073709551616"}},
		{name: "long event", req: types.OperationLogListReq{EventType: strings.Repeat("a", 65)}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := operationLogFilter(tt.req); err == nil {
				t.Fatal("operationLogFilter() error = nil, want validation error")
			}
		})
	}
}
