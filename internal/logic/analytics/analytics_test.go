package analytics

import (
	"testing"
	"time"

	"notes-of-ashen/internal/types"
)

func TestParseRangeAndChange(t *testing.T) {
	from, to, previousFrom, previousTo, days, err := parseRange(types.AnalyticsRangeReq{From: "2026-07-01", To: "2026-07-30"})
	if err != nil {
		t.Fatalf("parseRange() error = %v", err)
	}
	if from != "2026-07-01" || to != "2026-07-30" || previousFrom != "2026-06-01" || previousTo != "2026-06-30" || days != 30 {
		t.Fatalf("parseRange() = %q %q %q %q %d", from, to, previousFrom, previousTo, days)
	}
	if change(10, 0) != nil {
		t.Fatal("change() previous zero should return nil")
	}
	got := change(15, 10)
	if got == nil || *got != 50 {
		t.Fatalf("change() = %v, want 50", got)
	}
}

func TestParseRangeRejectsInvalidOrOversizedRange(t *testing.T) {
	tests := []types.AnalyticsRangeReq{
		{From: "2026-07-31", To: "2026-07-01"},
		{From: "2025-01-01", To: "2026-07-01"},
		{From: "invalid", To: time.Now().Format("2006-01-02")},
	}
	for _, req := range tests {
		if _, _, _, _, _, err := parseRange(req); err == nil {
			t.Fatalf("parseRange(%+v) error = nil", req)
		}
	}
}
