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

func TestChatCompletionsEndpoint(t *testing.T) {
	tests := []struct {
		name string
		base string
		want string
	}{
		{
			name: "base url",
			base: "https://api.example.com/v1",
			want: "https://api.example.com/v1/chat/completions",
		},
		{
			name: "base url with trailing slash",
			base: "https://api.example.com/v1/",
			want: "https://api.example.com/v1/chat/completions",
		},
		{
			name: "full endpoint",
			base: "https://api.example.com/v1/chat/completions",
			want: "https://api.example.com/v1/chat/completions",
		},
		{
			name: "full endpoint with trailing slash",
			base: "https://api.example.com/v1/chat/completions/",
			want: "https://api.example.com/v1/chat/completions",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := chatCompletionsEndpoint(tt.base); got != tt.want {
				t.Fatalf("chatCompletionsEndpoint() = %q, want %q", got, tt.want)
			}
		})
	}
}
