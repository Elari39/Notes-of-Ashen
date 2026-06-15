package httphelper

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"notes-of-ashen/internal/types"
)

func TestMetaIgnoresForwardedHeadersByDefault(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/traffic/visit", nil)
	req.RemoteAddr = "203.0.113.10:54321"
	req.Host = "api.local"
	req.Header.Set("X-Forwarded-For", "198.51.100.20")
	req.Header.Set("X-Real-IP", "198.51.100.30")
	req.Header.Set("X-Forwarded-Host", "example.com")

	meta := Meta(req)

	if meta.IP != "203.0.113.10" {
		t.Fatalf("IP = %q, want remote addr", meta.IP)
	}
	if meta.Host != "api.local" {
		t.Fatalf("Host = %q, want request host", meta.Host)
	}
}

func TestMetaUsesForwardedHeadersFromTrustedProxy(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/traffic/visit", nil)
	req.RemoteAddr = "172.18.0.5:54321"
	req.Host = "api.local"
	req.Header.Set("X-Forwarded-For", "198.51.100.20, 172.18.0.5")
	req.Header.Set("X-Real-IP", "198.51.100.30")
	req.Header.Set("X-Forwarded-Host", "blog.example.com")

	meta := Meta(req, ForwardedOptions{TrustedProxyCIDRs: "172.18.0.0/16"})

	if meta.IP != "198.51.100.20" {
		t.Fatalf("IP = %q, want forwarded client ip", meta.IP)
	}
	if meta.Host != "blog.example.com" {
		t.Fatalf("Host = %q, want forwarded host", meta.Host)
	}
}

func TestMetaFallsBackWhenForwardedIPIsInvalid(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/traffic/visit", nil)
	req.RemoteAddr = "172.18.0.5:54321"
	req.Header.Set("X-Forwarded-For", "unknown")
	req.Header.Set("X-Real-IP", "198.51.100.30")

	meta := Meta(req, ForwardedOptions{TrustedProxyCIDRs: "172.18.0.0/16"})

	if meta.IP != "198.51.100.30" {
		t.Fatalf("IP = %q, want valid x-real-ip fallback", meta.IP)
	}
}

func TestRequestBaseURLUsesForwardedHeadersOnlyForTrustedProxy(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/rss.xml", nil)
	req.RemoteAddr = "203.0.113.10:54321"
	req.Host = "api.local"
	req.Header.Set("X-Forwarded-Proto", "https")
	req.Header.Set("X-Forwarded-Host", "blog.example.com")

	if got := RequestBaseURL(req); got != "http://api.local" {
		t.Fatalf("RequestBaseURL() = %q, want untrusted host", got)
	}

	req.RemoteAddr = "172.18.0.5:54321"
	got := RequestBaseURL(req, ForwardedOptions{TrustedProxyCIDRs: "172.18.0.0/16"})
	if got != "https://blog.example.com" {
		t.Fatalf("RequestBaseURL() = %q, want trusted forwarded url", got)
	}
}

func TestRequestBaseURLRejectsInvalidForwardedProtoAndHost(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/rss.xml", nil)
	req.RemoteAddr = "172.18.0.5:54321"
	req.Host = "api.local"
	req.Header.Set("X-Forwarded-Proto", "javascript")
	req.Header.Set("X-Forwarded-Host", "bad host")

	got := RequestBaseURL(req, ForwardedOptions{TrustedProxyCIDRs: "172.18.0.0/16"})
	if got != "http://api.local" {
		t.Fatalf("RequestBaseURL() = %q, want fallback url", got)
	}
}

func TestParseUpdateResumePageAllowsMissingItemIDs(t *testing.T) {
	body := []byte(`{
		"title": "简介",
		"subtitle": "AI应用开发工程师",
		"contentMarkdown": "- 熟练掌握 Go 与 Python",
		"experiences": [{
			"role": "后端开发",
			"organization": "Example",
			"location": "",
			"startDate": "2025",
			"endDate": "至今",
			"description": "",
			"highlights": ["RESTful API"],
			"displayOrder": 1
		}],
		"educations": [{
			"school": "Example University",
			"degree": "",
			"major": "CS",
			"location": "",
			"startDate": "",
			"endDate": "",
			"description": "",
			"highlights": [],
			"displayOrder": 1
		}],
		"skills": [{
			"category": "Go",
			"name": "Golang",
			"level": 60,
			"description": "",
			"displayOrder": 1
		}]
	}`)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/site/resume", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	var parsed types.UpdateResumePageReq
	if err := Parse(req, &parsed); err != nil {
		t.Fatalf("Parse(UpdateResumePageReq) returned error: %v", err)
	}
	if len(parsed.Experiences) != 1 || len(parsed.Educations) != 1 || len(parsed.Skills) != 1 {
		t.Fatalf("parsed resume item counts = %d/%d/%d, want 1/1/1", len(parsed.Experiences), len(parsed.Educations), len(parsed.Skills))
	}
	if parsed.Experiences[0].ID != 0 || parsed.Educations[0].ID != 0 || parsed.Skills[0].ID != 0 {
		t.Fatalf("missing ids should parse as zero values: %#v %#v %#v", parsed.Experiences[0].ID, parsed.Educations[0].ID, parsed.Skills[0].ID)
	}
}
