package aiclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"notes-of-ashen/internal/config"
)

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
	timeout := time.Duration(conf.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	body, err := json.Marshal(chatRequest{
		Model:          strings.TrimSpace(conf.Model),
		Temperature:    0.3,
		MaxTokens:      maxTokens(req.Action),
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

	client := &http.Client{Timeout: timeout}
	httpResp, err := client.Do(httpReq)
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
	default:
		return 4000
	}
}
