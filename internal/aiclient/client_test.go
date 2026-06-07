package aiclient

import "testing"

func TestParseAssistantJSON(t *testing.T) {
	resp, err := ParseAssistantJSON("```json\n{\"summary\":\"摘要\",\"seoDescription\":\"描述\",\"seoKeywords\":\"go, blog\"}\n```")
	if err != nil {
		t.Fatalf("ParseAssistantJSON() error = %v", err)
	}
	if resp.Summary != "摘要" || resp.SEOKeywords != "go, blog" {
		t.Fatalf("unexpected response: %#v", resp)
	}
}

func TestParseAssistantJSONRejectsInvalidContent(t *testing.T) {
	if _, err := ParseAssistantJSON("not json"); err == nil {
		t.Fatal("ParseAssistantJSON() error = nil, want error")
	}
}
