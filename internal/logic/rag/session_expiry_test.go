package rag

import (
	"testing"
	"time"

	"notes-of-ashen/model"
)

func TestIsExpiredSession(t *testing.T) {
	now := time.Date(2026, time.August, 13, 12, 0, 0, 0, time.UTC)
	if isExpiredSession(nil, now) {
		t.Fatal("nil session must not be treated as expired")
	}
	if isExpiredSession(&model.RAGChatSession{}, now) {
		t.Fatal("permanent session must not be treated as expired")
	}
	expired := now.Add(-time.Nanosecond)
	if !isExpiredSession(&model.RAGChatSession{ExpiresAt: &expired}, now) {
		t.Fatal("past expiry must be treated as expired")
	}
	boundary := now
	if !isExpiredSession(&model.RAGChatSession{ExpiresAt: &boundary}, now) {
		t.Fatal("expiry at current time must be treated as expired")
	}
	future := now.Add(time.Nanosecond)
	if isExpiredSession(&model.RAGChatSession{ExpiresAt: &future}, now) {
		t.Fatal("future expiry must remain accessible")
	}
}
