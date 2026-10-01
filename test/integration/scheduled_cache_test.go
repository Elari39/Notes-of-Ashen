//go:build integration

package integration

import (
	"net/http"
	"testing"
	"time"
)

func TestCoreScheduledPublicationInvalidatesWarmList(t *testing.T) {
	h := newHarness(t)
	admin := h.registerUser(t, "scheduledcache")
	scheduled := time.Now().UTC().Add(5 * time.Second).Truncate(time.Second)
	created, err := h.jsonRequest(t.Context(), http.MethodPost, "/api/v1/articles", map[string]any{
		"title": "Scheduled cache boundary", "slug": h.uniqueSlug("schedule-cache"),
		"content": "The list and detail must become public together.", "status": "published",
		"scheduledAt": scheduled.Format(time.RFC3339),
	}, admin.accessToken, "")
	if err != nil {
		t.Fatal(err)
	}
	expectAPISuccess(t, created)
	item := decodeData[article](t, created)
	listPath := "/api/v1/articles?page=1&size=10"
	for range 2 {
		before, err := h.jsonRequest(t.Context(), http.MethodGet, listPath, nil, "", "")
		if err != nil {
			t.Fatal(err)
		}
		expectAPISuccess(t, before)
		list := decodeData[struct {
			Items []article `json:"items"`
			Total int       `json:"total"`
		}](t, before)
		if list.Total != 0 || len(list.Items) != 0 {
			t.Fatal("article visible before its scheduled publication")
		}
	}
	keys, err := h.redis.Keys(t.Context(), "article:list:*").Result()
	if err != nil {
		t.Fatal(err)
	}
	if len(keys) != 1 {
		t.Fatalf("expected one warmed list cache, got %v", keys)
	}
	if !time.Now().Before(scheduled) {
		t.Fatal("test setup crossed publication time before cache verification")
	}
	select {
	case <-time.After(time.Until(scheduled) + 50*time.Millisecond):
	case <-t.Context().Done():
		t.Fatal(t.Context().Err())
	}
	h.publicArticle(t, item.ID)
	after, err := h.jsonRequest(t.Context(), http.MethodGet, listPath, nil, "", "")
	if err != nil {
		t.Fatal(err)
	}
	expectAPISuccess(t, after)
	list := decodeData[struct {
		Items []article `json:"items"`
		Total int       `json:"total"`
	}](t, after)
	if list.Total != 1 || len(list.Items) != 1 || list.Items[0].ID != item.ID {
		t.Fatalf("warm list did not publish at scheduled time: %+v", list)
	}
}
