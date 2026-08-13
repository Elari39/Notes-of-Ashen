// Package ragclient 封装 RAG 所需的外部向量服务协议。
//
// 本包不读取数据库配置，也不记录正文、问题或 API Key。配置校验、密钥解密和
// 错误映射由 logic/rag 负责；这样模型协议可以独立测试且不会意外泄露敏感内容。
package ragclient

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"notes-of-ashen/internal/aiclient"
)

const (
	maxResponseBytes = 4 << 20
	maxErrorBytes    = 4096
)

// Config 是模型服务运行时配置。APIKey 只在内存中短暂存在，调用方不得记录它。
type Config struct {
	ChatBaseURL      string
	EmbeddingBaseURL string
	RerankURL        string
	APIKey           string
	ChatModel        string
	EmbeddingModel   string
	EmbeddingDims    int
	RerankModel      string
	FirstByteTimeout time.Duration
	RequestTimeout   time.Duration

	// httpClient 仅供本包的本地协议测试注入。生产调用始终使用带 SSRF
	// 防护的 aiclient.NewPublicHTTPClient，外部包无法设置此字段。
	httpClient *http.Client
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type RerankResult struct {
	Index int
	Score float64
}

// HTTPStatusError 仅保留安全的 HTTP 状态信息，不携带上游响应体、请求正文、
// 回答正文或密钥。上游有可能在错误页中回显输入，因此不能把响应体放入 Error。
type HTTPStatusError struct {
	StatusCode int
	Message    string
}

func (e *HTTPStatusError) Error() string {
	if e == nil || e.Message == "" {
		return "rag upstream request failed"
	}
	return "rag upstream request failed: " + e.Message
}

// SafeErrorSummary 将 RAG 出站错误归约为可记录的固定摘要。调用方会把该值写入
// 日志或 MySQL 状态表；绝不能直接记录 err.Error()，因为网络库和第三方服务可能
// 在其中包含 URL 凭据、请求文本或响应回显。
func SafeErrorSummary(err error) string {
	if err == nil {
		return ""
	}
	var statusErr *HTTPStatusError
	if errors.As(err, &statusErr) && statusErr != nil {
		return "upstream http status " + strconv.Itoa(statusErr.StatusCode)
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "upstream request timed out"
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return "upstream request timed out"
	}
	if errors.Is(err, context.Canceled) {
		return "upstream request cancelled"
	}
	return "rag upstream request failed"
}

// SafeStoredErrorSummary 是给持久化索引状态的防御性读取入口。新代码只会写入
// SafeErrorSummary 的固定值，但旧版本、人工修复或数据迁移可能留下不可信文本，
// 因而管理接口也不能直接返回原字段。
func SafeStoredErrorSummary(value string) string {
	value = strings.TrimSpace(value)
	switch value {
	case "":
		return ""
	case "upstream request timed out", "upstream request cancelled", "rag upstream request failed":
		return value
	}
	const prefix = "upstream http status "
	if strings.HasPrefix(value, prefix) {
		statusCode, err := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(value, prefix)))
		if err == nil && statusCode >= http.StatusContinue && statusCode <= 599 {
			return prefix + strconv.Itoa(statusCode)
		}
	}
	return "rag index operation failed"
}

func (c Config) client() *http.Client {
	if c.httpClient != nil {
		return c.httpClient
	}
	return aiclient.NewPublicHTTPClient(c.FirstByteTimeout)
}

func (c Config) timeout(ctx context.Context) (context.Context, context.CancelFunc) {
	if c.RequestTimeout <= 0 {
		return context.WithTimeout(ctx, 10*time.Minute)
	}
	return context.WithTimeout(ctx, c.RequestTimeout)
}

func Embedding(ctx context.Context, conf Config, input []string) ([]float64, error) {
	if len(input) == 0 {
		return nil, errors.New("embedding input is empty")
	}
	endpoint, err := openAIEndpoint(conf.EmbeddingBaseURL, "/embeddings")
	if err != nil {
		return nil, err
	}
	payload := struct {
		Model      string   `json:"model"`
		Input      []string `json:"input"`
		Dimensions int      `json:"dimensions,omitempty"`
	}{
		Model:      strings.TrimSpace(conf.EmbeddingModel),
		Input:      input,
		Dimensions: conf.EmbeddingDims,
	}
	raw, err := conf.jsonRequest(ctx, http.MethodPost, endpoint, payload)
	if err != nil {
		return nil, err
	}
	var response struct {
		Data []struct {
			Embedding []float64 `json:"embedding"`
		} `json:"data"`
		Error json.RawMessage `json:"error"`
	}
	if err := json.Unmarshal(raw, &response); err != nil {
		return nil, fmt.Errorf("decode embedding response: %w", err)
	}
	if len(response.Data) != len(input) {
		return nil, errors.New("embedding response size is invalid")
	}
	if len(response.Data[0].Embedding) == 0 {
		return nil, errors.New("embedding response is empty")
	}
	if conf.EmbeddingDims > 0 && len(response.Data[0].Embedding) != conf.EmbeddingDims {
		return nil, fmt.Errorf("embedding dimensions mismatch: got %d, want %d", len(response.Data[0].Embedding), conf.EmbeddingDims)
	}
	return response.Data[0].Embedding, nil
}

// Embeddings 将多个文本逐条送往服务。DashScope/OpenAI 的批量响应虽然标准化，
// 但部分兼容服务会改变排序；逐条调用保证文段与向量的一一对应和可重试语义。
func Embeddings(ctx context.Context, conf Config, input []string) ([][]float64, error) {
	result := make([][]float64, 0, len(input))
	for _, item := range input {
		vectors, err := Embedding(ctx, conf, []string{item})
		if err != nil {
			return nil, err
		}
		result = append(result, vectors)
	}
	return result, nil
}

func Rerank(ctx context.Context, conf Config, query string, documents []string) ([]RerankResult, error) {
	if strings.TrimSpace(query) == "" || len(documents) == 0 {
		return []RerankResult{}, nil
	}
	// DashScope 原生 Text Rerank API 与 OpenAI 兼容接口不同：检索文本位于
	// input，控制项位于 parameters。不要将 query/documents 平铺在顶层，
	// 否则服务会拒绝请求或忽略重排参数。
	payload := struct {
		Model string `json:"model"`
		Input struct {
			Query     string   `json:"query"`
			Documents []string `json:"documents"`
		} `json:"input"`
		Parameters struct {
			TopN       int  `json:"top_n"`
			ReturnDocs bool `json:"return_documents"`
		} `json:"parameters"`
	}{Model: strings.TrimSpace(conf.RerankModel)}
	payload.Input.Query = query
	payload.Input.Documents = documents
	payload.Parameters.TopN = len(documents)
	payload.Parameters.ReturnDocs = false
	raw, err := conf.jsonRequest(ctx, http.MethodPost, strings.TrimSpace(conf.RerankURL), payload)
	if err != nil {
		return nil, err
	}
	var response struct {
		Output struct {
			Results []rerankResponseItem `json:"results"`
		} `json:"output"`
		// Results 仅用于兼容返回相同结果格式的第三方代理；DashScope 原生响
		// 应使用 output.results。
		Results []rerankResponseItem `json:"results"`
	}
	if err := json.Unmarshal(raw, &response); err != nil {
		return nil, fmt.Errorf("decode rerank response: %w", err)
	}
	items := response.Output.Results
	if len(items) == 0 {
		items = response.Results
	}
	result := make([]RerankResult, 0, len(items))
	for _, item := range items {
		if item.Index < 0 || item.Index >= len(documents) {
			continue
		}
		var score float64
		switch {
		case item.RelevanceScore != nil:
			score = *item.RelevanceScore
		case item.Score != nil:
			score = *item.Score
		default:
			return nil, errors.New("rerank result has no relevance score")
		}
		result = append(result, RerankResult{Index: item.Index, Score: score})
	}
	if len(result) == 0 {
		return nil, errors.New("rerank response is empty")
	}
	return result, nil
}

// rerankResponseItem 使用指针区分分数为 0 和字段缺失。前者是合法的
// DashScope relevance_score，不能被旧代理的 score 字段错误覆盖。
type rerankResponseItem struct {
	Index          int      `json:"index"`
	RelevanceScore *float64 `json:"relevance_score"`
	Score          *float64 `json:"score"`
}

// StreamChat 从 OpenAI 兼容 SSE 流中回调纯文本增量。回调报错会立即中止上游请求。
func StreamChat(ctx context.Context, conf Config, messages []Message, onDelta func(string) error) error {
	endpoint, err := openAIEndpoint(conf.ChatBaseURL, "/chat/completions")
	if err != nil {
		return err
	}
	payload := struct {
		Model       string    `json:"model"`
		Messages    []Message `json:"messages"`
		Stream      bool      `json:"stream"`
		Temperature float64   `json:"temperature"`
		MaxTokens   int       `json:"max_tokens"`
	}{Model: strings.TrimSpace(conf.ChatModel), Messages: messages, Stream: true, Temperature: 0.2, MaxTokens: 1800}
	raw, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode chat request: %w", err)
	}
	requestCtx, cancel := conf.timeout(ctx)
	defer cancel()
	req, err := http.NewRequestWithContext(requestCtx, http.MethodPost, endpoint, bytes.NewReader(raw))
	if err != nil {
		return fmt.Errorf("create chat request: %w", err)
	}
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Content-Type", "application/json")
	if key := strings.TrimSpace(conf.APIKey); key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}
	resp, err := conf.client().Do(req)
	if err != nil {
		return fmt.Errorf("send chat request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return statusError(resp)
	}
	scanner := bufio.NewScanner(io.LimitReader(resp.Body, maxResponseBytes))
	// 一个 SSE data 行理论上可能是一个完整 token，但兼容服务有时会输出较长 reason
	// 字段。放大 Scanner buffer 防止合法行被截断。
	scanner.Buffer(make([]byte, 4096), maxResponseBytes)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "" || data == "[DONE]" {
			if data == "[DONE]" {
				return nil
			}
			continue
		}
		var event struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
			} `json:"choices"`
			Error json.RawMessage `json:"error"`
		}
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			return fmt.Errorf("decode chat stream event: %w", err)
		}
		if len(event.Error) > 0 && string(event.Error) != "null" {
			// 上游 SSE 可能以 200 返回错误事件；错误对象有机会回显问题或
			// 文章内容，因此只返回固定错误，不传播其正文。
			return errors.New("rag upstream returned stream error")
		}
		for _, choice := range event.Choices {
			if choice.Delta.Content != "" {
				if err := onDelta(choice.Delta.Content); err != nil {
					return err
				}
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read chat stream: %w", err)
	}
	// OpenAI 兼容流以 [DONE] 作为完整响应的明确终止标记。若 EOF 提前到达，
	// 不能把局部内容保存成完整回答并向客户端发出 done 事件。
	return errors.New("chat stream ended before done")
}

func (c Config) jsonRequest(ctx context.Context, method, endpoint string, payload any) ([]byte, error) {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		return nil, errors.New("rag endpoint is required")
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("encode rag request: %w", err)
	}
	requestCtx, cancel := c.timeout(ctx)
	defer cancel()
	req, err := http.NewRequestWithContext(requestCtx, method, endpoint, bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("create rag request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	if key := strings.TrimSpace(c.APIKey); key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}
	resp, err := c.client().Do(req)
	if err != nil {
		return nil, fmt.Errorf("send rag request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, statusError(resp)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read rag response: %w", err)
	}
	if len(body) > maxResponseBytes {
		return nil, errors.New("rag response body exceeds limit")
	}
	return body, nil
}

func openAIEndpoint(baseURL, suffix string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", errors.New("rag base url is invalid")
	}
	basePath := strings.TrimRight(parsed.Path, "/")
	for _, endpoint := range []string{"/chat/completions", "/embeddings"} {
		if strings.HasSuffix(basePath, endpoint) {
			basePath = strings.TrimSuffix(basePath, endpoint)
			break
		}
	}
	parsed.Path = strings.TrimRight(basePath, "/") + suffix
	parsed.RawPath = ""
	return parsed.String(), nil
}

func statusError(resp *http.Response) error {
	// 有限读取以便连接可复用，但不解析、更不保留或传播响应体。模型服务可能
	// 在错误信息中回显用户问题、文章片段甚至 Authorization 内容。
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, maxErrorBytes+1))
	return &HTTPStatusError{
		StatusCode: resp.StatusCode,
		Message:    "upstream http status " + strconv.Itoa(resp.StatusCode),
	}
}
