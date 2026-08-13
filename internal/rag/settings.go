package rag

import (
	"errors"
	"net/url"
	"strings"
	"time"

	"notes-of-ashen/internal/ragclient"
	"notes-of-ashen/internal/validator"
	"notes-of-ashen/model"
)

const (
	defaultChatBaseURL      = "https://dashscope.aliyuncs.com/compatible-mode/v1"
	defaultEmbeddingBaseURL = "https://dashscope.aliyuncs.com/compatible-mode/v1"
	defaultRerankURL        = "https://dashscope.aliyuncs.com/api/v1/services/rerank/text-rerank/text-rerank"
	defaultChatModel        = "qwen-plus"
	defaultEmbeddingModel   = "text-embedding-v4"
	// DashScope text-embedding 系列（v1/v2/v3/v4）都支持 1024 维输出，
	// 而 4096 不在 text-embedding-v4 的合法维度集合内（合法值为
	// 64/128/256/512/768/1024/1536/2048/3072），会造成每次 embedding
	// 请求被上游 400 拒绝、索引重建必然失败。
	defaultEmbeddingDims = 1024
	defaultRerankModel   = "qwen3-vl-rerank"
)

// historyRetentionDays 必须是管理界面提供的明确档位。0 表示永久保留，其他值
// 分别覆盖产品定义的 7、30、60、90、180、365 天，避免任意大数造成无意的长期
// 私有会话留存。
var allowedHistoryRetentionDays = map[int]struct{}{
	0: {}, 7: {}, 30: {}, 60: {}, 90: {}, 180: {}, 365: {},
}

func IsValidHistoryRetentionDays(days int) bool {
	_, ok := allowedHistoryRetentionDays[days]
	return ok
}

// EffectiveProviderConfig 读取数据库已加密的独立 RAG key；严禁回退文章 AI
// 助手的密钥，避免权限/账单边界被静默混用。
func EffectiveProviderConfig(settings model.RAGSettings, authSecret string) (ragclient.Config, error) {
	if !settings.Enabled {
		return ragclient.Config{}, errors.New("rag is disabled")
	}
	apiKey, err := decryptAPIKey(settings.APIKeyCipher, authSecret)
	if err != nil {
		return ragclient.Config{}, err
	}
	if strings.TrimSpace(apiKey) == "" || strings.TrimSpace(settings.ChatBaseURL) == "" || strings.TrimSpace(settings.EmbeddingBaseURL) == "" || strings.TrimSpace(settings.RerankURL) == "" || strings.TrimSpace(settings.ChatModel) == "" || strings.TrimSpace(settings.EmbeddingModel) == "" || strings.TrimSpace(settings.RerankModel) == "" || settings.EmbeddingDimensions < 1 {
		return ragclient.Config{}, errors.New("rag is not configured")
	}
	for _, item := range []struct{ value, field string }{
		{settings.ChatBaseURL, "chatBaseUrl"}, {settings.EmbeddingBaseURL, "embeddingBaseUrl"}, {settings.RerankURL, "rerankUrl"},
	} {
		if err := validateProviderURL(item.value, item.field); err != nil {
			return ragclient.Config{}, err
		}
	}
	return ragclient.Config{
		ChatBaseURL: strings.TrimRight(strings.TrimSpace(settings.ChatBaseURL), "/"), EmbeddingBaseURL: strings.TrimRight(strings.TrimSpace(settings.EmbeddingBaseURL), "/"), RerankURL: strings.TrimSpace(settings.RerankURL), APIKey: apiKey, ChatModel: strings.TrimSpace(settings.ChatModel), EmbeddingModel: strings.TrimSpace(settings.EmbeddingModel), EmbeddingDims: settings.EmbeddingDimensions, RerankModel: strings.TrimSpace(settings.RerankModel), FirstByteTimeout: 60 * time.Second, RequestTimeout: 10 * time.Minute,
	}, nil
}

func validateSettings(settings model.RAGSettings, requireConfigured bool) error {
	for _, item := range []struct{ value, field string }{
		{settings.ChatBaseURL, "chatBaseUrl"}, {settings.EmbeddingBaseURL, "embeddingBaseUrl"}, {settings.RerankURL, "rerankUrl"},
	} {
		if err := validateProviderURL(item.value, item.field); err != nil {
			return err
		}
	}
	for _, item := range []struct{ value, field string }{
		{settings.ChatModel, "chatModel"}, {settings.EmbeddingModel, "embeddingModel"}, {settings.RerankModel, "rerankModel"},
	} {
		if err := validator.Length(strings.TrimSpace(item.value), item.field, 0, 160); err != nil {
			return err
		}
	}
	if settings.EmbeddingDimensions < 1 || settings.EmbeddingDimensions > 32768 {
		return errors.New("embeddingDimensions is invalid")
	}
	if !IsValidHistoryRetentionDays(settings.HistoryRetentionDays) {
		return errors.New("historyRetentionDays is invalid")
	}
	if requireConfigured {
		for _, value := range []string{settings.ChatBaseURL, settings.EmbeddingBaseURL, settings.RerankURL, settings.ChatModel, settings.EmbeddingModel, settings.RerankModel} {
			if strings.TrimSpace(value) == "" {
				return errors.New("rag provider fields are required when enabled")
			}
		}
	}
	return nil
}

// validateProviderURL 是 RAG 上游专用的 URL 校验。通用 HTTP URL 校验允许
// userinfo、query 和 fragment，以兼容头像等普通链接；模型服务 URL 则不能携带
// 它们，避免管理员误把额外凭据保存、返回或随请求发送给上游。
func validateProviderURL(value, field string) error {
	value = strings.TrimSpace(value)
	if err := validator.OptionalHTTPURL(value, field); err != nil {
		return err
	}
	if value == "" {
		return nil
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.User != nil || parsed.RawQuery != "" || parsed.ForceQuery || parsed.Fragment != "" || strings.Contains(value, "#") {
		return errors.New(field + " must not contain credentials, query, or fragment")
	}
	return nil
}

func defaults(settings model.RAGSettings) model.RAGSettings {
	if settings.ChatBaseURL == "" {
		settings.ChatBaseURL = defaultChatBaseURL
	}
	if settings.EmbeddingBaseURL == "" {
		settings.EmbeddingBaseURL = defaultEmbeddingBaseURL
	}
	if settings.RerankURL == "" {
		settings.RerankURL = defaultRerankURL
	}
	if settings.ChatModel == "" {
		settings.ChatModel = defaultChatModel
	}
	if settings.EmbeddingModel == "" {
		settings.EmbeddingModel = defaultEmbeddingModel
	}
	if settings.EmbeddingDimensions <= 0 {
		settings.EmbeddingDimensions = defaultEmbeddingDims
	}
	if settings.RerankModel == "" {
		settings.RerankModel = defaultRerankModel
	}
	if !IsValidHistoryRetentionDays(settings.HistoryRetentionDays) {
		settings.HistoryRetentionDays = 90
	}
	return settings
}

// DefaultSettings 补足迁移前旧数据库缺少的键，供管理业务层调用。
func DefaultSettings(settings model.RAGSettings) model.RAGSettings { return defaults(settings) }

func ValidateSettings(settings model.RAGSettings, requireConfigured bool) error {
	return validateSettings(settings, requireConfigured)
}

func EncryptAPIKey(value, authSecret string) (string, error) { return encryptAPIKey(value, authSecret) }
func APIKeyStatus(value, authSecret string) (bool, bool)     { return apiKeyStatus(value, authSecret) }

// isEmbeddingChanged 控制 collection 重建；聊天和 rerank 模型变更仅即时生效。
func isEmbeddingChanged(current, next model.RAGSettings) bool {
	return strings.TrimRight(strings.TrimSpace(current.EmbeddingBaseURL), "/") != strings.TrimRight(strings.TrimSpace(next.EmbeddingBaseURL), "/") || strings.TrimSpace(current.EmbeddingModel) != strings.TrimSpace(next.EmbeddingModel) || current.EmbeddingDimensions != next.EmbeddingDimensions
}

func EmbeddingChanged(current, next model.RAGSettings) bool { return isEmbeddingChanged(current, next) }
