package traffic

import "testing"

func TestClassifyReferer(t *testing.T) {
	tests := []struct {
		name       string
		referrer   string
		host       string
		sourceType string
		sourceName string
	}{
		{name: "direct", sourceType: "direct", sourceName: "direct"},
		{name: "internal absolute", referrer: "https://example.com/article/1", host: "example.com", sourceType: "internal", sourceName: "site"},
		{name: "internal path", referrer: "/article/1", host: "example.com", sourceType: "internal", sourceName: "site"},
		{name: "google", referrer: "https://www.google.com/search?q=go", host: "example.com", sourceType: "search", sourceName: "google"},
		{name: "baidu", referrer: "https://m.baidu.com/s?wd=go", host: "example.com", sourceType: "search", sourceName: "baidu"},
		{name: "external", referrer: "https://news.example.org/post", host: "example.com", sourceType: "external", sourceName: "news.example.org"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sourceType, sourceName := classifyReferer(tt.referrer, tt.host)
			if sourceType != tt.sourceType || sourceName != tt.sourceName {
				t.Fatalf("classifyReferer() = (%q, %q), want (%q, %q)", sourceType, sourceName, tt.sourceType, tt.sourceName)
			}
		})
	}
}

func TestVisitorDailyHash(t *testing.T) {
	got := visitorDailyHash("2026-06-07", "127.0.0.1", "ua")
	if got == "" {
		t.Fatal("visitorDailyHash() returned empty value")
	}
	if got != visitorDailyHash("2026-06-07", "127.0.0.1", "ua") {
		t.Fatal("visitorDailyHash() is not stable")
	}
	if got == visitorDailyHash("2026-06-08", "127.0.0.1", "ua") {
		t.Fatal("visitorDailyHash() should change across dates")
	}
	if got != visitorDailyHash("2026-06-07", "127.0.0.1", "ua", "") {
		t.Fatal("missing visitor id should preserve the IP/UA fallback hash")
	}
	if got == visitorDailyHash("2026-06-07", "127.0.0.1", "ua", "abcdef12-3456-7890-abcd-ef1234567890") {
		t.Fatal("visitor id should participate in the daily UV hash")
	}
	if visitorDailyHash("2026-06-07", "127.0.0.1", "ua", "visitor-a-123456") == visitorDailyHash("2026-06-07", "127.0.0.1", "ua", "visitor-b-123456") {
		t.Fatal("different visitor ids behind the same NAT should not share a UV hash")
	}
}

func TestIsPublicTrafficPath(t *testing.T) {
	if !isPublicTrafficPath("/article/1") {
		t.Fatal("/article/1 should be public")
	}
	if isPublicTrafficPath("/admin/dashboard") {
		t.Fatal("/admin/dashboard should be ignored")
	}
	if isPublicTrafficPath("/login") {
		t.Fatal("/login should be ignored")
	}
}
