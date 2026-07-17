package mq

import (
	"regexp"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestTruncateUserAgent(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "keeps 512 byte ASCII", input: strings.Repeat("a", maxUserAgentBytes), want: strings.Repeat("a", maxUserAgentBytes)},
		{name: "does not split UTF-8 rune", input: strings.Repeat("a", 510) + "中", want: strings.Repeat("a", 510)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateUserAgent(tt.input)
			if got != tt.want {
				t.Fatalf("truncateUserAgent() = %q, want %q", got, tt.want)
			}
			if len(got) > maxUserAgentBytes {
				t.Fatalf("truncated User-Agent length = %d, want <= %d", len(got), maxUserAgentBytes)
			}
			if !utf8.ValidString(got) {
				t.Fatalf("truncated User-Agent is not valid UTF-8: %q", got)
			}
		})
	}
}

func TestWriteOperationLogEventTruncatesUserAgent(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock: %v", err)
	}
	defer db.Close()

	createdAt := time.Date(2026, time.July, 17, 12, 0, 0, 0, time.UTC)
	wantUserAgent := strings.Repeat("a", 510)
	mock.ExpectExec(regexp.QuoteMeta(`
INSERT INTO operation_logs (user_id, event_type, resource_type, resource_id, metadata, ip, user_agent, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)`)).
		WithArgs(nil, "audit", "article", nil, nil, "127.0.0.1", wantUserAgent, createdAt).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = writeOperationLogEvent(db, Event{
		EventType:    "audit",
		ResourceType: "article",
		IP:           "127.0.0.1",
		UserAgent:    wantUserAgent + "中",
		CreatedAt:    createdAt,
	})
	if err != nil {
		t.Fatalf("write operation log event: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sqlmock expectations: %v", err)
	}
}
