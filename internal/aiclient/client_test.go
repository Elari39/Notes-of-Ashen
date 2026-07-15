package aiclient

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

const testAPIKey = "secret-test-api-key"

func TestParseAssistantJSON(t *testing.T) {
	resp, err := ParseAssistantJSON("```json\n{\"title\":\"标题\",\"slug\":\"article-title\",\"summary\":\"摘要\",\"seoTitle\":\"SEO 标题\",\"seoDescription\":\"描述\",\"seoKeywords\":\"go, blog\",\"categorySuggestion\":\"技术\",\"tagSuggestions\":[\"Go\",\"AI\"]}\n```")
	if err != nil {
		t.Fatalf("ParseAssistantJSON() error = %v", err)
	}
	if resp.Title != "标题" || resp.Slug != "article-title" || resp.Summary != "摘要" ||
		resp.SEOTitle != "SEO 标题" || resp.SEOKeywords != "go, blog" ||
		resp.CategorySuggestion != "技术" || len(resp.TagSuggestions) != 2 {
		t.Fatalf("unexpected response: %#v", resp)
	}
}

func TestParseAssistantJSONRejectsInvalidContent(t *testing.T) {
	if _, err := ParseAssistantJSON("not json"); err == nil {
		t.Fatal("ParseAssistantJSON() error = nil, want error")
	}
}

func TestNormalizeAPIFormat(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{input: "", want: APIFormatOpenAI},
		{input: " OpenAI ", want: APIFormatOpenAI},
		{input: "ANTHROPIC", want: APIFormatAnthropic},
	}
	for _, tt := range tests {
		got, err := NormalizeAPIFormat(tt.input)
		if err != nil {
			t.Fatalf("NormalizeAPIFormat(%q) error = %v", tt.input, err)
		}
		if got != tt.want {
			t.Fatalf("NormalizeAPIFormat(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
	if _, err := NormalizeAPIFormat("unknown"); err == nil {
		t.Fatal("NormalizeAPIFormat(unknown) error = nil, want error")
	}
}

func TestEndpointForSupportsBaseAndFullEndpoints(t *testing.T) {
	tests := []struct {
		name   string
		base   string
		format string
		kind   endpointKind
		want   string
	}{
		{
			name:   "openai base url",
			base:   "https://api.example.com/v1/",
			format: APIFormatOpenAI,
			kind:   endpointCompletion,
			want:   "https://api.example.com/v1/chat/completions",
		},
		{
			name:   "openai full completion endpoint",
			base:   "https://api.example.com/v1/chat/completions/",
			format: APIFormatOpenAI,
			kind:   endpointCompletion,
			want:   "https://api.example.com/v1/chat/completions",
		},
		{
			name:   "openai derives models from completion endpoint",
			base:   "https://api.example.com/v1/chat/completions?api-version=1",
			format: APIFormatOpenAI,
			kind:   endpointModels,
			want:   "https://api.example.com/v1/models?api-version=1",
		},
		{
			name:   "openai derives completion from models endpoint",
			base:   "https://api.example.com/v1/models",
			format: APIFormatOpenAI,
			kind:   endpointCompletion,
			want:   "https://api.example.com/v1/chat/completions",
		},
		{
			name:   "anthropic host base adds v1",
			base:   "https://api.anthropic.com",
			format: APIFormatAnthropic,
			kind:   endpointCompletion,
			want:   "https://api.anthropic.com/v1/messages",
		},
		{
			name:   "anthropic path prefix adds v1",
			base:   "https://api.example.com/anthropic",
			format: APIFormatAnthropic,
			kind:   endpointCompletion,
			want:   "https://api.example.com/anthropic/v1/messages",
		},
		{
			name:   "anthropic path prefix derives models and keeps query",
			base:   "https://api.example.com/anthropic?tenant=notes",
			format: APIFormatAnthropic,
			kind:   endpointModels,
			want:   "https://api.example.com/anthropic/v1/models?tenant=notes",
		},
		{
			name:   "anthropic version base",
			base:   "https://api.anthropic.com/v1",
			format: APIFormatAnthropic,
			kind:   endpointModels,
			want:   "https://api.anthropic.com/v1/models",
		},
		{
			name:   "anthropic derives models from messages endpoint",
			base:   "https://api.anthropic.com/v1/messages/",
			format: APIFormatAnthropic,
			kind:   endpointModels,
			want:   "https://api.anthropic.com/v1/models",
		},
		{
			name:   "anthropic keeps explicit root messages endpoint",
			base:   "https://proxy.example.com/messages",
			format: APIFormatAnthropic,
			kind:   endpointCompletion,
			want:   "https://proxy.example.com/messages",
		},
		{
			name:   "anthropic derives root models from root messages endpoint",
			base:   "https://proxy.example.com/messages",
			format: APIFormatAnthropic,
			kind:   endpointModels,
			want:   "https://proxy.example.com/models",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := endpointFor(tt.base, tt.format, tt.kind)
			if err != nil {
				t.Fatalf("endpointFor() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("endpointFor() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestChatCompletionsEndpoint(t *testing.T) {
	if got := chatCompletionsEndpoint("https://api.example.com/v1/"); got != "https://api.example.com/v1/chat/completions" {
		t.Fatalf("chatCompletionsEndpoint() = %q", got)
	}
}

func TestAssistOpenAI(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/chat/completions" {
			t.Errorf("request = %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer "+testAPIKey {
			t.Errorf("Authorization = %q", got)
		}
		var request openAIChatRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Errorf("decode request: %v", err)
		}
		if request.Model != "test-model" || request.ResponseFormat.Type != "json_object" {
			t.Errorf("request = %#v", request)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"choices":[{"message":{"role":"assistant","content":"{\"summary\":\"摘要\"}"}}]}`)
	}))
	defer server.Close()

	resp, err := Assist(context.Background(), testConfig(server.URL+"/v1", APIFormatOpenAI), Request{
		Action:  "metadata",
		Title:   "标题",
		Content: "正文",
	})
	if err != nil {
		t.Fatalf("Assist() error = %v", err)
	}
	if resp.Summary != "摘要" {
		t.Fatalf("Assist() = %#v", resp)
	}
}

func TestAssistAnthropic(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/messages" {
			t.Errorf("request = %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("x-api-key"); got != testAPIKey {
			t.Errorf("x-api-key = %q", got)
		}
		if got := r.Header.Get("anthropic-version"); got != "2023-06-01" {
			t.Errorf("anthropic-version = %q", got)
		}
		var request anthropicRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Errorf("decode request: %v", err)
		}
		if request.Model != "test-model" || len(request.Messages) != 1 || request.Messages[0].Role != "user" {
			t.Errorf("request = %#v", request)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"content":[{"type":"text","text":"{\"revisedContent\":\"修订\"}"}]}`)
	}))
	defer server.Close()

	resp, err := Assist(context.Background(), testConfig(server.URL+"/v1/messages", APIFormatAnthropic), Request{
		Action:  "proofread",
		Content: "正文",
	})
	if err != nil {
		t.Fatalf("Assist() error = %v", err)
	}
	if resp.RevisedContent != "修订" {
		t.Fatalf("Assist() = %#v", resp)
	}
}

func TestListModelsNormalizesResults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/v1/models" {
			t.Errorf("request = %s %s", r.Method, r.URL.Path)
		}
		if r.URL.Query().Has("limit") {
			t.Errorf("OpenAI models request unexpectedly has limit = %q", r.URL.Query().Get("limit"))
		}
		_, _ = fmt.Fprint(w, `{"data":[{"id":" z-model "},{"id":""},{"id":"a-model"},{"id":"z-model"}]}`)
	}))
	defer server.Close()

	conf := testConfig(server.URL+"/v1/chat/completions", APIFormatOpenAI)
	models, err := ListModels(context.Background(), conf)
	if err != nil {
		t.Fatalf("ListModels() error = %v", err)
	}
	want := []string{"a-model", "z-model"}
	if fmt.Sprint(models) != fmt.Sprint(want) {
		t.Fatalf("ListModels() = %v, want %v", models, want)
	}
}

func TestTestModelUsesJSONProbeBudget(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request openAIChatRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Errorf("decode request: %v", err)
		}
		if request.MaxTokens != modelProbeMaxTokens {
			t.Errorf("MaxTokens = %d, want %d", request.MaxTokens, modelProbeMaxTokens)
		}
		if request.Temperature != 0 || request.ResponseFormat.Type != "json_object" {
			t.Errorf("probe request = %#v", request)
		}
		if len(request.Messages) != 2 || request.Messages[1].Content != `Return exactly {"ok":true}.` {
			t.Errorf("probe messages = %#v", request.Messages)
		}
		_, _ = fmt.Fprint(w, `{"choices":[{"message":{"content":"{\"ok\":true}"}}]}`)
	}))
	defer server.Close()

	elapsed, err := TestModel(context.Background(), testConfig(server.URL+"/v1", APIFormatOpenAI))
	if err != nil {
		t.Fatalf("TestModel() error = %v", err)
	}
	if elapsed <= 0 {
		t.Fatalf("TestModel() elapsed = %v", elapsed)
	}
}

func TestTestModelAnthropicUsesJSONProbeBudget(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/messages" {
			t.Errorf("request = %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("x-api-key"); got != testAPIKey {
			t.Errorf("x-api-key = %q", got)
		}
		if got := r.Header.Get("anthropic-version"); got != "2023-06-01" {
			t.Errorf("anthropic-version = %q", got)
		}
		var request anthropicRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Errorf("decode request: %v", err)
		}
		if request.MaxTokens != modelProbeMaxTokens || request.Temperature != 0 {
			t.Errorf("probe request = %#v", request)
		}
		if request.System != "Return only one valid JSON object and no Markdown." {
			t.Errorf("probe system = %q", request.System)
		}
		if len(request.Messages) != 1 || request.Messages[0].Content != `Return exactly {"ok":true}.` {
			t.Errorf("probe messages = %#v", request.Messages)
		}
		_, _ = fmt.Fprint(w, `{"content":[{"type":"thinking","thinking":"Checking the response format."},{"type":"text","text":"{\"ok\":true}"}]}`)
	}))
	defer server.Close()

	elapsed, err := TestModel(context.Background(), testConfig(server.URL, APIFormatAnthropic))
	if err != nil {
		t.Fatalf("TestModel() error = %v", err)
	}
	if elapsed <= 0 {
		t.Fatalf("TestModel() elapsed = %v", elapsed)
	}
}

func TestValidateProbeJSONRequiresOKTrue(t *testing.T) {
	for _, content := range []string{`{"ok":false}`, `{"ok":"true"}`, `{"OK":true}`, `{}`} {
		if err := validateProbeJSON(content); err == nil {
			t.Fatalf("validateProbeJSON(%s) error = nil, want error", content)
		}
	}
}

func TestListModelsAnthropicAddsLimitAndKeepsQuery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/v1/models" {
			t.Errorf("request = %s %s", r.Method, r.URL.Path)
		}
		if got := r.URL.Query().Get("limit"); got != "1000" {
			t.Errorf("limit = %q, want 1000", got)
		}
		if got := r.URL.Query().Get("tenant"); got != "notes" {
			t.Errorf("tenant = %q, want notes", got)
		}
		_, _ = fmt.Fprint(w, `{"data":[{"id":"claude-test"}]}`)
	}))
	defer server.Close()

	conf := testConfig(server.URL+"/v1/messages?tenant=notes", APIFormatAnthropic)
	models, err := ListModels(context.Background(), conf)
	if err != nil {
		t.Fatalf("ListModels() error = %v", err)
	}
	if len(models) != 1 || models[0] != "claude-test" {
		t.Fatalf("ListModels() = %v", models)
	}
}

func TestClientDoesNotFollowRedirects(t *testing.T) {
	var redirectedRequests atomic.Int32
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		redirectedRequests.Add(1)
		_, _ = fmt.Fprint(w, `{"data":[]}`)
	}))
	defer target.Close()
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Redirect(w, &http.Request{}, target.URL+"/v1/models", http.StatusTemporaryRedirect)
	}))
	defer origin.Close()

	_, err := ListModels(context.Background(), testConfig(origin.URL+"/v1", APIFormatOpenAI))
	if err == nil {
		t.Fatal("ListModels() error = nil, want redirect status error")
	}
	var statusErr *HTTPStatusError
	if !errors.As(err, &statusErr) || statusErr.StatusCode != http.StatusTemporaryRedirect {
		t.Fatalf("ListModels() error = %v", err)
	}
	if got := redirectedRequests.Load(); got != 0 {
		t.Fatalf("redirect target requests = %d, want 0", got)
	}
}

func TestResponseBodyLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprint(w, strings.Repeat("x", maxResponseBodyBytes+1))
	}))
	defer server.Close()

	_, err := ListModels(context.Background(), testConfig(server.URL+"/v1", APIFormatOpenAI))
	if !errors.Is(err, errResponseBodyTooLarge) {
		t.Fatalf("ListModels() error = %v, want body limit error", err)
	}
}

func TestHTTPStatusErrorRedactsAPIKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = fmt.Fprintf(w, `{"error":{"message":"invalid key %s"}}`, testAPIKey)
	}))
	defer server.Close()

	_, err := ListModels(context.Background(), testConfig(server.URL+"/v1", APIFormatOpenAI))
	if err == nil {
		t.Fatal("ListModels() error = nil, want error")
	}
	if strings.Contains(err.Error(), testAPIKey) {
		t.Fatalf("error leaked API key: %v", err)
	}
	var statusErr *HTTPStatusError
	if !errors.As(err, &statusErr) || statusErr.StatusCode != http.StatusUnauthorized {
		t.Fatalf("ListModels() error = %v", err)
	}
	if strings.Contains(statusErr.Message, testAPIKey) {
		t.Fatalf("HTTPStatusError.Message leaked API key: %q", statusErr.Message)
	}
}

func TestTimeoutDefaults(t *testing.T) {
	conf := Config{}
	if got := firstByteTimeout(conf); got != 60*time.Second {
		t.Fatalf("firstByteTimeout() = %v, want 60s", got)
	}
	if got := nonStreamTimeout(conf); got != 600*time.Second {
		t.Fatalf("nonStreamTimeout() = %v, want 600s", got)
	}
}

func TestTimeoutOverrides(t *testing.T) {
	conf := Config{FirstByteTimeoutSeconds: 2, NonStreamTimeoutSeconds: 3}
	if got := firstByteTimeout(conf); got != 2*time.Second {
		t.Fatalf("firstByteTimeout() = %v, want 2s", got)
	}
	if got := nonStreamTimeout(conf); got != 3*time.Second {
		t.Fatalf("nonStreamTimeout() = %v, want 3s", got)
	}
}

func TestSystemPromptSupportsWritingActions(t *testing.T) {
	for _, action := range []string{"expand", "shorten", "translate"} {
		t.Run(action, func(t *testing.T) {
			if got := systemPrompt(action); got == "" || got == systemPrompt("unknown") {
				t.Fatalf("systemPrompt(%q) did not use a dedicated prompt", action)
			}
		})
	}
}

func TestSystemPromptSupportsArticleCompletion(t *testing.T) {
	prompt := systemPrompt("complete")
	for _, field := range []string{"title", "slug", "summary", "seoTitle", "seoDescription", "seoKeywords", "categorySuggestion", "tagSuggestions"} {
		if !strings.Contains(prompt, field) {
			t.Fatalf("complete prompt missing %q", field)
		}
	}
	if maxTokens("complete") != 1200 {
		t.Fatalf("complete max tokens = %d, want 1200", maxTokens("complete"))
	}
}

func testConfig(baseURL, format string) Config {
	return Config{
		Enabled:                 true,
		APIFormat:               format,
		BaseURL:                 baseURL,
		APIKey:                  testAPIKey,
		Model:                   "test-model",
		FirstByteTimeoutSeconds: 2,
		NonStreamTimeoutSeconds: 2,
	}
}
