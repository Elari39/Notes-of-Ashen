package aiclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"
)

const (
	APIFormatOpenAI    = "openai"
	APIFormatAnthropic = "anthropic"

	modelProbeMaxTokens = 512
)

// Config 是 AI 客户端实际发起请求所需的运行时配置。
// BaseURL 的公开地址校验由 logic 层强制执行；客户端仅负责协议请求。
type Config struct {
	Enabled                 bool
	APIFormat               string
	BaseURL                 string
	APIKey                  string
	Model                   string
	FirstByteTimeoutSeconds int
	NonStreamTimeoutSeconds int
}

// HTTPStatusError 表示上游 AI 服务返回了非 2xx HTTP 状态。
// Message 已经过 API Key 脱敏，可由上层通过 errors.As 获取 StatusCode。
type HTTPStatusError struct {
	StatusCode int
	Message    string
}

func (e *HTTPStatusError) Error() string {
	if e == nil {
		return "ai request failed"
	}
	if strings.TrimSpace(e.Message) == "" {
		return fmt.Sprintf("ai request failed: status %d", e.StatusCode)
	}
	return fmt.Sprintf("ai request failed: status %d: %s", e.StatusCode, e.Message)
}

type Request struct {
	Action  string
	Title   string
	Content string
}

type Response struct {
	Title              string   `json:"title"`
	Slug               string   `json:"slug"`
	Summary            string   `json:"summary"`
	SEOTitle           string   `json:"seoTitle"`
	SEODescription     string   `json:"seoDescription"`
	SEOKeywords        string   `json:"seoKeywords"`
	CategorySuggestion string   `json:"categorySuggestion"`
	TagSuggestions     []string `json:"tagSuggestions"`
	RevisedContent     string   `json:"revisedContent"`
	Suggestions        []string `json:"suggestions"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type providerError struct {
	Message string `json:"message"`
}

type modelsResponse struct {
	Data []struct {
		ID string `json:"id"`
	} `json:"data"`
	Error *providerError `json:"error,omitempty"`
}

// NormalizeAPIFormat 规范化 AI API 格式；空值为兼容旧配置按 OpenAI 处理。
func NormalizeAPIFormat(format string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "", APIFormatOpenAI:
		return APIFormatOpenAI, nil
	case APIFormatAnthropic:
		return APIFormatAnthropic, nil
	default:
		return "", fmt.Errorf("unsupported ai api format")
	}
}

func Assist(ctx context.Context, conf Config, req Request) (resp *Response, err error) {
	defer func() {
		err = sanitizeError(err, conf.APIKey)
	}()

	content, err := requestCompletion(
		ctx,
		conf,
		systemPrompt(req.Action),
		userPrompt(req),
		0.3,
		maxTokens(req.Action),
	)
	if err != nil {
		return nil, err
	}
	return ParseAssistantJSON(content)
}

// ListModels 获取上游可用模型，并对模型 ID 去空、去重后按字典序排序。
func ListModels(ctx context.Context, conf Config) (models []string, err error) {
	defer func() {
		err = sanitizeError(err, conf.APIKey)
	}()

	format, err := NormalizeAPIFormat(conf.APIFormat)
	if err != nil {
		return nil, err
	}
	endpoint, err := endpointFor(conf.BaseURL, format, endpointModels)
	if err != nil {
		return nil, err
	}
	if format == APIFormatAnthropic {
		endpoint, err = anthropicModelsURL(endpoint)
		if err != nil {
			return nil, err
		}
	}
	raw, err := doJSONRequest(ctx, conf, format, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	var response modelsResponse
	if err := json.Unmarshal(raw, &response); err != nil {
		return nil, fmt.Errorf("decode ai models response: %w", err)
	}
	if response.Error != nil && strings.TrimSpace(response.Error.Message) != "" {
		return nil, fmt.Errorf("ai request failed: %s", response.Error.Message)
	}
	modelIDs := make([]string, 0, len(response.Data))
	for _, item := range response.Data {
		modelIDs = append(modelIDs, item.ID)
	}
	return normalizeModels(modelIDs), nil
}

// TestModel 使用固定的低 token JSON 探针测试模型，并返回完整请求耗时。
// 即使请求失败也会返回已经消耗的时间，便于调用方诊断超时与网络错误。
func TestModel(ctx context.Context, conf Config) (elapsed time.Duration, err error) {
	startedAt := time.Now()
	defer func() {
		elapsed = time.Since(startedAt)
		err = sanitizeError(err, conf.APIKey)
	}()

	content, err := requestCompletion(
		ctx,
		conf,
		"Return only one valid JSON object and no Markdown.",
		`Return exactly {"ok":true}.`,
		0,
		modelProbeMaxTokens,
	)
	if err != nil {
		return 0, err
	}
	if err := validateProbeJSON(content); err != nil {
		return 0, err
	}
	return 0, nil
}

func requestCompletion(ctx context.Context, conf Config, system, user string, temperature float64, maxTokens int) (string, error) {
	format, err := NormalizeAPIFormat(conf.APIFormat)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(conf.Model) == "" {
		return "", fmt.Errorf("ai model is required")
	}
	endpoint, err := endpointFor(conf.BaseURL, format, endpointCompletion)
	if err != nil {
		return "", err
	}

	var payload any
	if format == APIFormatAnthropic {
		payload = newAnthropicRequest(conf.Model, system, user, temperature, maxTokens)
	} else {
		payload = newOpenAIChatRequest(conf.Model, system, user, temperature, maxTokens)
	}

	raw, err := doJSONRequest(ctx, conf, format, http.MethodPost, endpoint, payload)
	if err != nil {
		return "", err
	}
	if format == APIFormatAnthropic {
		return parseAnthropicContent(raw)
	}
	return parseOpenAIContent(raw)
}

func normalizeModels(models []string) []string {
	unique := make(map[string]struct{}, len(models))
	for _, model := range models {
		model = strings.TrimSpace(model)
		if model != "" {
			unique[model] = struct{}{}
		}
	}
	result := make([]string, 0, len(unique))
	for model := range unique {
		result = append(result, model)
	}
	sort.Strings(result)
	return result
}
