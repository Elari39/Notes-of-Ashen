package aiclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"notes-of-ashen/internal/config"
)

// httpConnPool 复用 TCP/TLS 连接池（P4-11）：避免每次 Assist 调用都新建
// http.Client/Transport 重新握手、旧 Transport 空闲连接未关闭导致的 fd 泄漏。
// ResponseHeaderTimeout 设为首字节超时默认值，单次请求总超时由 ctx 控制。
var (
	httpConnPoolOnce sync.Once
	httpConnPool     *http.Client
)

func sharedHTTPClient(headerTimeout time.Duration) *http.Client {
	httpConnPoolOnce.Do(func() {
		httpConnPool = &http.Client{
			Transport: &http.Transport{
				Proxy: http.ProxyFromEnvironment,
				DialContext: (&net.Dialer{
					Timeout:   30 * time.Second,
					KeepAlive: 30 * time.Second,
				}).DialContext,
				TLSHandshakeTimeout:   30 * time.Second,
				ResponseHeaderTimeout: headerTimeout,
				ExpectContinueTimeout: 1 * time.Second,
			},
		}
	})
	return httpConnPool
}

type Request struct {
	Action  string
	Title   string
	Content string
}

type Response struct {
	Summary        string   `json:"summary"`
	SEODescription string   `json:"seoDescription"`
	SEOKeywords    string   `json:"seoKeywords"`
	RevisedContent string   `json:"revisedContent"`
	Suggestions    []string `json:"suggestions"`
}

type chatRequest struct {
	Model          string         `json:"model"`
	Messages       []chatMessage  `json:"messages"`
	Temperature    float64        `json:"temperature,omitempty"`
	MaxTokens      int            `json:"max_tokens,omitempty"`
	ResponseFormat responseFormat `json:"response_format,omitempty"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type responseFormat struct {
	Type string `json:"type"`
}

type chatResponse struct {
	Choices []struct {
		Message chatMessage `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func Assist(ctx context.Context, conf config.AIConf, req Request) (*Response, error) {
	timeout := nonStreamTimeout(conf)
	headerTimeout := firstByteTimeout(conf)
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	body, err := json.Marshal(chatRequest{
		Model:          strings.TrimSpace(conf.Model),
		Temperature:    assistTemperature(conf),
		MaxTokens:      assistMaxTokens(conf, req.Action),
		ResponseFormat: responseFormat{Type: "json_object"},
		Messages: []chatMessage{
			{Role: "system", Content: systemPrompt(req.Action)},
			{Role: "user", Content: userPrompt(req)},
		},
	})
	if err != nil {
		return nil, err
	}

	endpoint := chatCompletionsEndpoint(conf.BaseURL)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+strings.TrimSpace(conf.APIKey))

	httpResp, err := sharedHTTPClient(headerTimeout).Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer httpResp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(httpResp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		return nil, fmt.Errorf("ai request failed: status %d: %s", httpResp.StatusCode, strings.TrimSpace(string(raw)))
	}

	var resp chatResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, err
	}
	if resp.Error != nil && resp.Error.Message != "" {
		return nil, fmt.Errorf("ai request failed: %s", resp.Error.Message)
	}
	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("ai response has no choices")
	}
	content := resp.Choices[0].Message.Content
	parsed, err := ParseAssistantJSON(content)
	if err != nil {
		return nil, err
	}
	return parsed, nil
}

func chatCompletionsEndpoint(baseURL string) string {
	endpoint := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if strings.HasSuffix(endpoint, "/chat/completions") {
		return endpoint
	}
	return endpoint + "/chat/completions"
}

func ParseAssistantJSON(content string) (*Response, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		return nil, fmt.Errorf("ai response is empty")
	}
	start := strings.Index(content, "{")
	end := strings.LastIndex(content, "}")
	if start < 0 || end < start {
		return nil, fmt.Errorf("ai response is not json")
	}
	var resp Response
	if err := json.Unmarshal([]byte(content[start:end+1]), &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func systemPrompt(action string) string {
	switch action {
	case "metadata":
		return `你是博客文章 SEO 编辑助手。必须只输出 json，不要输出 Markdown。
JSON 字段必须为 summary、seoDescription、seoKeywords。
summary 约 100 个中文字符，提炼文章核心观点，不要写成标题；seoDescription 不超过 180 字；seoKeywords 为逗号分隔关键词。
示例 JSON：{"summary":"本文围绕主题提炼核心内容，保留关键背景、问题与结论，方便读者快速判断是否继续阅读。","seoDescription":"文章内容摘要。","seoKeywords":"关键词一,关键词二"}`
	case "proofread":
		return `你是中文博客文章校对助手。必须只输出 json，不要输出 Markdown。
JSON 字段必须为 revisedContent、suggestions。
保留原意和 Markdown 结构，只修正错别字、病句、标点和明显语法问题。
示例 JSON：{"revisedContent":"修订后的 Markdown 正文","suggestions":["修改说明"]}`
	case "polish":
		return `你是中文博客文章润色助手。必须只输出 json，不要输出 Markdown。
JSON 字段必须为 revisedContent、suggestions。
保留原意和 Markdown 结构，让表达更清晰自然，不要扩写事实。
示例 JSON：{"revisedContent":"润色后的 Markdown 正文","suggestions":["修改说明"]}`
	case "expand":
		return `你是中文博客文章伴写助手。必须只输出 json，不要输出 Markdown。
JSON 字段必须为 revisedContent、suggestions。
在不虚构事实的前提下扩写用户给出的段落，让论述更完整、衔接更自然，并保留 Markdown 结构。
示例 JSON：{"revisedContent":"扩写后的 Markdown 段落","suggestions":["扩写说明"]}`
	case "shorten":
		return `你是中文博客文章压缩助手。必须只输出 json，不要输出 Markdown。
JSON 字段必须为 revisedContent、suggestions。
保留核心信息和语气，删去冗余表达，让段落更短更清晰，并保留 Markdown 结构。
示例 JSON：{"revisedContent":"缩写后的 Markdown 段落","suggestions":["缩写说明"]}`
	case "translate":
		return `你是技术博客翻译助手。必须只输出 json，不要输出 Markdown。
JSON 字段必须为 revisedContent、suggestions。
将用户给出的段落翻译为自然英文，保留 Markdown 结构、代码、链接和专有名词；如果原文主要是英文，则翻译为中文。
示例 JSON：{"revisedContent":"Translated Markdown paragraph","suggestions":["翻译说明"]}`
	default:
		return `你是博客文章编辑助手。必须只输出 json，不要输出 Markdown。`
	}
}

func userPrompt(req Request) string {
	var builder strings.Builder
	if strings.TrimSpace(req.Title) != "" {
		builder.WriteString("标题：")
		builder.WriteString(strings.TrimSpace(req.Title))
		builder.WriteString("\n\n")
	}
	builder.WriteString("正文：\n")
	builder.WriteString(req.Content)
	return builder.String()
}

func maxTokens(action string) int {
	switch action {
	case "metadata":
		return 800
	case "proofread", "polish":
		return 12000
	case "expand", "shorten", "translate":
		return 4000
	default:
		return 4000
	}
}

// assistTemperature 取配置覆盖的温度，<=0 回退默认 0.3。
func assistTemperature(conf config.AIConf) float64 {
	if conf.Temperature > 0 {
		return conf.Temperature
	}
	return 0.3
}

// assistMaxTokens 取配置覆盖的最大 token 数，<=0 时按 action 回退默认值。
func assistMaxTokens(conf config.AIConf, action string) int {
	if conf.MaxTokens > 0 {
		return conf.MaxTokens
	}
	return maxTokens(action)
}

func firstByteTimeout(conf config.AIConf) time.Duration {
	seconds := conf.FirstByteTimeoutSeconds
	if seconds <= 0 {
		seconds = 60
	}
	return time.Duration(seconds) * time.Second
}

func nonStreamTimeout(conf config.AIConf) time.Duration {
	seconds := conf.NonStreamTimeoutSeconds
	if seconds <= 0 {
		seconds = conf.TimeoutSeconds
	}
	if seconds <= 0 {
		seconds = 600
	}
	return time.Duration(seconds) * time.Second
}
