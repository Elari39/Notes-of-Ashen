package ragclient

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

const protocolTestAPIKey = "test-rag-api-key"

func TestEmbeddingUsesDashScopeCompatibleContract(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/compatible-mode/v1/embeddings" {
			t.Errorf("path = %q, want compatible embeddings endpoint", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer "+protocolTestAPIKey {
			t.Errorf("Authorization = %q", got)
		}

		var payload struct {
			Model      string   `json:"model"`
			Input      []string `json:"input"`
			Dimensions int      `json:"dimensions"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Errorf("decode request: %v", err)
			return
		}
		if payload.Model != "text-embedding-v4" {
			t.Errorf("model = %q", payload.Model)
		}
		if !reflect.DeepEqual(payload.Input, []string{"文章片段"}) {
			t.Errorf("input = %#v", payload.Input)
		}
		if payload.Dimensions != 4 {
			t.Errorf("dimensions = %d, want 4", payload.Dimensions)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"data":[{"embedding":[0.1,0.2,0.3,0.4]}]}`)
	}))
	defer server.Close()

	vector, err := Embedding(context.Background(), testDashScopeConfig(server), []string{"文章片段"})
	if err != nil {
		t.Fatalf("Embedding() error = %v", err)
	}
	if want := []float64{0.1, 0.2, 0.3, 0.4}; !reflect.DeepEqual(vector, want) {
		t.Fatalf("Embedding() = %v, want %v", vector, want)
	}
}

func TestEmbeddingRejectsUnexpectedDimensions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"data":[{"embedding":[0.1,0.2,0.3]}]}`)
	}))
	defer server.Close()

	_, err := Embedding(context.Background(), testDashScopeConfig(server), []string{"文章片段"})
	if err == nil || err.Error() != "embedding dimensions mismatch: got 3, want 4" {
		t.Fatalf("Embedding() error = %v, want dimension mismatch", err)
	}
}

func TestRerankUsesDashScopeNativeContractAndParsesOutput(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/api/v1/services/rerank/text-rerank/text-rerank" {
			t.Errorf("path = %q, want native rerank endpoint", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer "+protocolTestAPIKey {
			t.Errorf("Authorization = %q", got)
		}

		var payload map[string]json.RawMessage
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Errorf("decode request: %v", err)
			return
		}
		if _, exists := payload["query"]; exists {
			t.Error("native rerank request unexpectedly contains top-level query")
		}
		if _, exists := payload["documents"]; exists {
			t.Error("native rerank request unexpectedly contains top-level documents")
		}
		if got := string(payload["model"]); got != `"qwen3-vl-rerank"` {
			t.Errorf("model = %s", got)
		}

		var input struct {
			Query     string   `json:"query"`
			Documents []string `json:"documents"`
		}
		if err := json.Unmarshal(payload["input"], &input); err != nil {
			t.Errorf("decode input: %v", err)
		}
		if input.Query != "如何启用 RAG？" || !reflect.DeepEqual(input.Documents, []string{"第一篇", "第二篇"}) {
			t.Errorf("input = %#v", input)
		}
		var parameters struct {
			TopN       int  `json:"top_n"`
			ReturnDocs bool `json:"return_documents"`
		}
		if err := json.Unmarshal(payload["parameters"], &parameters); err != nil {
			t.Errorf("decode parameters: %v", err)
		}
		if parameters.TopN != 2 || parameters.ReturnDocs {
			t.Errorf("parameters = %#v", parameters)
		}

		w.Header().Set("Content-Type", "application/json")
		// relevance_score 为 0 仍是合法结果；不能误用代理兼容字段 score 覆盖它。
		_, _ = fmt.Fprint(w, `{"output":{"results":[{"index":1,"relevance_score":0.98},{"index":0,"relevance_score":0,"score":0.75}]}}`)
	}))
	defer server.Close()

	results, err := Rerank(context.Background(), testDashScopeConfig(server), "如何启用 RAG？", []string{"第一篇", "第二篇"})
	if err != nil {
		t.Fatalf("Rerank() error = %v", err)
	}
	want := []RerankResult{{Index: 1, Score: 0.98}, {Index: 0, Score: 0}}
	if !reflect.DeepEqual(results, want) {
		t.Fatalf("Rerank() = %#v, want %#v", results, want)
	}
}

func TestRerankAcceptsProxyResultShape(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"results":[{"index":0,"score":0.6}]}`)
	}))
	defer server.Close()

	results, err := Rerank(context.Background(), testDashScopeConfig(server), "问题", []string{"文档"})
	if err != nil {
		t.Fatalf("Rerank() error = %v", err)
	}
	if want := []RerankResult{{Index: 0, Score: 0.6}}; !reflect.DeepEqual(results, want) {
		t.Fatalf("Rerank() = %#v, want %#v", results, want)
	}
}

func TestUpstreamHTTPErrorNeverIncludesResponseBodyOrAPIKey(t *testing.T) {
	const sensitiveBody = "article body and user question must never reach logs"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = fmt.Fprintf(w, `{"error":{"message":%q}}`, sensitiveBody+" key="+protocolTestAPIKey)
	}))
	defer server.Close()

	conf := testDashScopeConfig(server)
	_, err := Embedding(context.Background(), conf, []string{"private question"})
	if err == nil {
		t.Fatal("Embedding() error = nil, want HTTP status error")
	}
	if strings.Contains(err.Error(), sensitiveBody) || strings.Contains(err.Error(), protocolTestAPIKey) {
		t.Fatalf("Embedding() error leaked provider response: %q", err)
	}
	var statusErr *HTTPStatusError
	if !errors.As(err, &statusErr) || statusErr.StatusCode != http.StatusBadRequest {
		t.Fatalf("Embedding() error = %#v, want status 400", err)
	}
	if got := SafeErrorSummary(err); got != "upstream http status 400" {
		t.Fatalf("SafeErrorSummary() = %q", got)
	}
}

func TestSafeErrorSummaryDoesNotForwardArbitraryErrorText(t *testing.T) {
	if got := SafeErrorSummary(errors.New("private question: explain article body")); got != "rag upstream request failed" {
		t.Fatalf("SafeErrorSummary() = %q", got)
	}
	if got := SafeErrorSummary(context.DeadlineExceeded); got != "upstream request timed out" {
		t.Fatalf("SafeErrorSummary(deadline) = %q", got)
	}
	if got := SafeErrorSummary(timeoutError{}); got != "upstream request timed out" {
		t.Fatalf("SafeErrorSummary(timeout) = %q", got)
	}
}

func TestSafeStoredErrorSummaryRejectsLegacyOrInjectedText(t *testing.T) {
	if got := SafeStoredErrorSummary("question=private article body"); got != "rag index operation failed" {
		t.Fatalf("SafeStoredErrorSummary() = %q", got)
	}
	if got := SafeStoredErrorSummary(" upstream http status 429 "); got != "upstream http status 429" {
		t.Fatalf("SafeStoredErrorSummary(http) = %q", got)
	}
	if got := SafeStoredErrorSummary("upstream http status 999"); got != "rag index operation failed" {
		t.Fatalf("SafeStoredErrorSummary(invalid http) = %q", got)
	}
}

func TestStreamChatRequiresDoneTerminator(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"partial\"}}]}\n\n")
	}))
	defer server.Close()

	var deltas []string
	err := StreamChat(context.Background(), testDashScopeConfig(server), []Message{{Role: "user", Content: "test"}}, func(delta string) error {
		deltas = append(deltas, delta)
		return nil
	})
	if err == nil || err.Error() != "chat stream ended before done" {
		t.Fatalf("StreamChat() error = %v, want missing done error", err)
	}
	if !reflect.DeepEqual(deltas, []string{"partial"}) {
		t.Fatalf("deltas = %#v, want partial delta", deltas)
	}
}

func TestStreamChatAcceptsDoneTerminator(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"complete\"}}]}\n\n")
		_, _ = fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer server.Close()

	var deltas []string
	err := StreamChat(context.Background(), testDashScopeConfig(server), []Message{{Role: "user", Content: "test"}}, func(delta string) error {
		deltas = append(deltas, delta)
		return nil
	})
	if err != nil {
		t.Fatalf("StreamChat() error = %v", err)
	}
	if !reflect.DeepEqual(deltas, []string{"complete"}) {
		t.Fatalf("deltas = %#v, want complete delta", deltas)
	}
}

func TestStreamChatRejectsEmbeddedErrorWithoutForwardingBody(t *testing.T) {
	const sensitiveError = "private question and article content"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = fmt.Fprintf(w, "data: {\"error\":{\"message\":%q}}\n\n", sensitiveError)
		_, _ = fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer server.Close()

	err := StreamChat(context.Background(), testDashScopeConfig(server), []Message{{Role: "user", Content: "test"}}, func(string) error { return nil })
	if err == nil || err.Error() != "rag upstream returned stream error" {
		t.Fatalf("StreamChat() error = %v, want fixed embedded error", err)
	}
	if strings.Contains(err.Error(), sensitiveError) {
		t.Fatalf("StreamChat() error leaked provider body: %q", err)
	}
}

type timeoutError struct{}

func (timeoutError) Error() string   { return "private upstream timeout detail" }
func (timeoutError) Timeout() bool   { return true }
func (timeoutError) Temporary() bool { return true }

var _ interface {
	error
	Timeout() bool
} = timeoutError{}

func testDashScopeConfig(server *httptest.Server) Config {
	return Config{
		ChatBaseURL:      server.URL + "/compatible-mode/v1",
		EmbeddingBaseURL: server.URL + "/compatible-mode/v1",
		RerankURL:        server.URL + "/api/v1/services/rerank/text-rerank/text-rerank",
		APIKey:           protocolTestAPIKey,
		ChatModel:        "qwen-plus",
		EmbeddingModel:   "text-embedding-v4",
		EmbeddingDims:    4,
		RerankModel:      "qwen3-vl-rerank",
		httpClient:       server.Client(),
	}
}
