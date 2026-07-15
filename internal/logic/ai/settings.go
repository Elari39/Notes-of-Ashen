package ai

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"

	"notes-of-ashen/internal/aiclient"
	"notes-of-ashen/internal/authutil"
	apperrors "notes-of-ashen/internal/errors"
	"notes-of-ashen/internal/svc"
	"notes-of-ashen/internal/types"
	"notes-of-ashen/internal/validator"
	"notes-of-ashen/model"

	"github.com/zeromicro/go-zero/core/logx"
)

const (
	minAITimeoutSeconds  = 1
	maxAITimeoutSeconds  = 1800
	secretCipherV2Prefix = "v2:"
	secretCipherV3Prefix = "v3:"
	maxAIAPIKeyLength    = 4096
)

var errAIAPIKeyNeedsUpdate = errors.New("ai api key needs update")

func Settings(ctx context.Context, svcCtx *svc.ServiceContext) (*types.AISettingsResp, error) {
	if err := authutil.RequireAdmin(ctx); err != nil {
		return nil, err
	}
	settings, err := svcCtx.Store.AISettings(ctx)
	if err != nil {
		return nil, err
	}
	return aiSettingsResp(*settings, svcCtx.Config.Auth.AccessSecret), nil
}

func UpdateSettings(ctx context.Context, svcCtx *svc.ServiceContext, req types.UpdateAISettingsReq) (*types.AISettingsResp, error) {
	if err := authutil.RequireAdmin(ctx); err != nil {
		return nil, err
	}
	current, err := svcCtx.Store.AISettings(ctx)
	if err != nil {
		return nil, err
	}

	apiFormat := current.APIFormat
	if strings.TrimSpace(req.APIFormat) != "" {
		apiFormat, err = normalizeAIAPIFormat(req.APIFormat)
		if err != nil {
			return nil, err
		}
	}
	baseURL, err := normalizeAndValidateAIBaseURL(req.BaseURL, false)
	if err != nil {
		return nil, err
	}
	modelName := strings.TrimSpace(req.Model)
	if err := validator.Length(modelName, "model", 0, 120); err != nil {
		return nil, err
	}
	firstByteTimeout, nonStreamTimeout, err := validateAITimeouts(
		req.FirstByteTimeoutSeconds,
		req.NonStreamTimeoutSeconds,
		svcCtx.Config.Timeout,
	)
	if err != nil {
		return nil, err
	}

	apiKey := strings.TrimSpace(req.APIKey)
	if apiKey != "" && req.ClearAPIKey {
		return nil, apperrors.BadRequest("apiKey and clearApiKey cannot be used together")
	}
	if err := validator.Length(apiKey, "apiKey", 0, maxAIAPIKeyLength); err != nil {
		return nil, err
	}
	endpointChanged := !sameAIEndpoint(*current, apiFormat, baseURL)
	if endpointChanged && strings.TrimSpace(current.APIKeyCipher) != "" && apiKey == "" && !req.ClearAPIKey {
		return nil, apperrors.BadRequest("api key must be replaced or cleared when ai endpoint changes")
	}

	apiKeyCipher, err := apiKeyCipherForUpdate(current.APIKeyCipher, req, svcCtx.Config.Auth.AccessSecret)
	if err != nil {
		return nil, err
	}
	next := model.AISettings{
		Enabled:                 req.Enabled,
		APIFormat:               apiFormat,
		BaseURL:                 baseURL,
		APIKeyCipher:            apiKeyCipher,
		Model:                   modelName,
		FirstByteTimeoutSeconds: firstByteTimeout,
		NonStreamTimeoutSeconds: nonStreamTimeout,
	}
	if req.Enabled {
		if err := validateEnabledAISettings(next, svcCtx.Config.Auth.AccessSecret); err != nil {
			return nil, err
		}
	}

	if err := svcCtx.Store.UpdateAISettings(ctx, next); err != nil {
		return nil, err
	}
	return aiSettingsResp(next, svcCtx.Config.Auth.AccessSecret), nil
}

func Models(ctx context.Context, svcCtx *svc.ServiceContext, req types.AIConnectionReq) (*types.AIModelsResp, error) {
	if err := authutil.RequireAdmin(ctx); err != nil {
		return nil, err
	}
	conf, err := draftConfig(ctx, svcCtx, draftConfigInput{
		APIFormat:               req.APIFormat,
		BaseURL:                 req.BaseURL,
		APIKey:                  req.APIKey,
		FirstByteTimeoutSeconds: req.FirstByteTimeoutSeconds,
		NonStreamTimeoutSeconds: req.NonStreamTimeoutSeconds,
	})
	if err != nil {
		return nil, err
	}
	models, err := aiclient.ListModels(ctx, conf)
	if err != nil {
		return nil, MapProviderError(ctx, "list models", err)
	}
	return &types.AIModelsResp{Models: models}, nil
}

func TestModel(ctx context.Context, svcCtx *svc.ServiceContext, req types.AIModelTestReq) (*types.AIModelTestResp, error) {
	if err := authutil.RequireAdmin(ctx); err != nil {
		return nil, err
	}
	modelName := strings.TrimSpace(req.Model)
	if err := validator.Length(modelName, "model", 1, 120); err != nil {
		return nil, err
	}
	conf, err := draftConfig(ctx, svcCtx, draftConfigInput{
		APIFormat:               req.APIFormat,
		BaseURL:                 req.BaseURL,
		APIKey:                  req.APIKey,
		Model:                   modelName,
		FirstByteTimeoutSeconds: req.FirstByteTimeoutSeconds,
		NonStreamTimeoutSeconds: req.NonStreamTimeoutSeconds,
	})
	if err != nil {
		return nil, err
	}
	latency, err := aiclient.TestModel(ctx, conf)
	if err != nil {
		return nil, MapProviderError(ctx, "test model", err)
	}
	latencyMs := latency.Milliseconds()
	if latencyMs < 1 {
		latencyMs = 1
	}
	return &types.AIModelTestResp{Model: modelName, LatencyMs: latencyMs}, nil
}

// EffectiveConfig returns the database-backed runtime configuration. AI business
// settings no longer fall back to YAML or environment variables.
func EffectiveConfig(ctx context.Context, svcCtx *svc.ServiceContext) (aiclient.Config, bool, error) {
	settings, err := svcCtx.Store.AISettings(ctx)
	if err != nil {
		return normalizeAIConf(aiclient.Config{}), false, err
	}
	conf := configFromSettings(*settings)
	configured := strings.TrimSpace(settings.APIKeyCipher) != ""
	if !settings.Enabled || !configured {
		return conf, configured, nil
	}
	apiKey, err := decryptAIAPIKey(settings.APIKeyCipher, svcCtx.Config.Auth.AccessSecret)
	if err != nil {
		logx.WithContext(ctx).Errorf("decrypt ai api key failed: %v", err)
		return conf, true, apperrors.BadRequest("ai api key needs update")
	}
	conf.APIKey = apiKey
	return conf, true, nil
}

type draftConfigInput struct {
	APIFormat               string
	BaseURL                 string
	APIKey                  string
	Model                   string
	FirstByteTimeoutSeconds int
	NonStreamTimeoutSeconds int
}

func draftConfig(ctx context.Context, svcCtx *svc.ServiceContext, input draftConfigInput) (aiclient.Config, error) {
	apiFormat, err := normalizeAIAPIFormat(input.APIFormat)
	if err != nil {
		return aiclient.Config{}, err
	}
	baseURL, err := normalizeAndValidateAIBaseURL(input.BaseURL, true)
	if err != nil {
		return aiclient.Config{}, err
	}
	firstByteTimeout, nonStreamTimeout, err := validateAITimeouts(
		input.FirstByteTimeoutSeconds,
		input.NonStreamTimeoutSeconds,
		svcCtx.Config.Timeout,
	)
	if err != nil {
		return aiclient.Config{}, err
	}
	apiKey := strings.TrimSpace(input.APIKey)
	if err := validator.Length(apiKey, "apiKey", 0, maxAIAPIKeyLength); err != nil {
		return aiclient.Config{}, err
	}
	if apiKey == "" {
		stored, err := svcCtx.Store.AISettings(ctx)
		if err != nil {
			return aiclient.Config{}, err
		}
		if !sameAIEndpoint(*stored, apiFormat, baseURL) {
			return aiclient.Config{}, apperrors.BadRequest("api key is required for unsaved ai endpoint")
		}
		if strings.TrimSpace(stored.APIKeyCipher) == "" {
			return aiclient.Config{}, apperrors.BadRequest("api key is required")
		}
		apiKey, err = decryptAIAPIKey(stored.APIKeyCipher, svcCtx.Config.Auth.AccessSecret)
		if err != nil {
			return aiclient.Config{}, apperrors.BadRequest("ai api key needs update")
		}
	}
	return normalizeAIConf(aiclient.Config{
		Enabled:                 true,
		APIFormat:               apiFormat,
		BaseURL:                 baseURL,
		APIKey:                  apiKey,
		Model:                   strings.TrimSpace(input.Model),
		FirstByteTimeoutSeconds: firstByteTimeout,
		NonStreamTimeoutSeconds: nonStreamTimeout,
	}), nil
}

func configFromSettings(settings model.AISettings) aiclient.Config {
	return normalizeAIConf(aiclient.Config{
		Enabled:                 settings.Enabled,
		APIFormat:               model.NormalizeAIAPIFormat(settings.APIFormat),
		BaseURL:                 normalizeAIBaseURL(settings.BaseURL),
		Model:                   strings.TrimSpace(settings.Model),
		FirstByteTimeoutSeconds: settings.FirstByteTimeoutSeconds,
		NonStreamTimeoutSeconds: settings.NonStreamTimeoutSeconds,
	})
}

func normalizeAIConf(conf aiclient.Config) aiclient.Config {
	conf.APIFormat = model.NormalizeAIAPIFormat(conf.APIFormat)
	conf.BaseURL = normalizeAIBaseURL(conf.BaseURL)
	conf.Model = strings.TrimSpace(conf.Model)
	if conf.FirstByteTimeoutSeconds <= 0 {
		conf.FirstByteTimeoutSeconds = model.DefaultAIFirstByteWait
	}
	if conf.NonStreamTimeoutSeconds <= 0 {
		conf.NonStreamTimeoutSeconds = model.DefaultAINonStreamWait
	}
	return conf
}

func normalizeAIAPIFormat(value string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", model.AIAPIFormatOpenAI:
		return model.AIAPIFormatOpenAI, nil
	case model.AIAPIFormatAnthropic:
		return model.AIAPIFormatAnthropic, nil
	default:
		return "", apperrors.BadRequest("apiFormat is invalid")
	}
}

func normalizeAIBaseURL(value string) string {
	return strings.TrimRight(strings.TrimSpace(value), "/")
}

func sameAIEndpoint(settings model.AISettings, apiFormat, baseURL string) bool {
	return model.NormalizeAIAPIFormat(settings.APIFormat) == apiFormat &&
		normalizeAIBaseURL(settings.BaseURL) == normalizeAIBaseURL(baseURL)
}

func normalizeAndValidateAIBaseURL(value string, required bool) (string, error) {
	baseURL := normalizeAIBaseURL(value)
	minLength := 0
	if required {
		minLength = 1
	}
	if err := validator.Length(baseURL, "baseUrl", minLength, 255); err != nil {
		return "", err
	}
	if err := validator.OptionalHTTPURL(baseURL, "baseUrl"); err != nil {
		return "", err
	}
	return baseURL, nil
}

func validateAITimeouts(firstByte, nonStream int, globalTimeoutMS int64) (int, int, error) {
	var err error
	firstByte, err = normalizeAITimeout(firstByte, model.DefaultAIFirstByteWait, "firstByteTimeoutSeconds")
	if err != nil {
		return 0, 0, err
	}
	nonStream, err = normalizeAITimeout(nonStream, model.DefaultAINonStreamWait, "nonStreamTimeoutSeconds")
	if err != nil {
		return 0, 0, err
	}
	if nonStream < firstByte {
		return 0, 0, apperrors.BadRequest("nonStreamTimeoutSeconds must be greater than or equal to firstByteTimeoutSeconds")
	}
	if globalTimeoutMS > 0 && int64(nonStream)*1000 > globalTimeoutMS {
		return 0, 0, apperrors.BadRequest("nonStreamTimeoutSeconds must not exceed the server request timeout")
	}
	return firstByte, nonStream, nil
}

func normalizeAITimeout(value int, fallback int, field string) (int, error) {
	if value <= 0 {
		value = fallback
	}
	if value < minAITimeoutSeconds || value > maxAITimeoutSeconds {
		return 0, apperrors.BadRequest(field + " is invalid")
	}
	return value, nil
}

func validateEnabledAISettings(settings model.AISettings, authSecret string) error {
	if strings.TrimSpace(settings.BaseURL) == "" {
		return apperrors.BadRequest("baseUrl is required when AI is enabled")
	}
	if strings.TrimSpace(settings.Model) == "" {
		return apperrors.BadRequest("model is required when AI is enabled")
	}
	if strings.TrimSpace(settings.APIKeyCipher) == "" {
		return apperrors.BadRequest("apiKey is required when AI is enabled")
	}
	if _, err := decryptAIAPIKey(settings.APIKeyCipher, authSecret); err != nil {
		return apperrors.BadRequest("ai api key needs update")
	}
	return nil
}

func aiSettingsResp(settings model.AISettings, authSecret string) *types.AISettingsResp {
	conf := configFromSettings(settings)
	configured, needsUpdate := apiKeyStatus(settings.APIKeyCipher, authSecret)
	return &types.AISettingsResp{
		Enabled:                 conf.Enabled,
		APIFormat:               conf.APIFormat,
		BaseURL:                 conf.BaseURL,
		Model:                   conf.Model,
		APIKeyConfigured:        configured,
		APIKeyNeedsUpdate:       needsUpdate,
		FirstByteTimeoutSeconds: conf.FirstByteTimeoutSeconds,
		NonStreamTimeoutSeconds: conf.NonStreamTimeoutSeconds,
	}
}

func apiKeyStatus(encoded, authSecret string) (bool, bool) {
	encoded = strings.TrimSpace(encoded)
	if encoded == "" {
		return false, false
	}
	if strings.HasPrefix(encoded, secretCipherV2Prefix) {
		return true, true
	}
	_, err := decryptAIAPIKey(encoded, authSecret)
	return true, err != nil
}

func apiKeyCipherForUpdate(currentCipher string, req types.UpdateAISettingsReq, authSecret string) (string, error) {
	apiKey := strings.TrimSpace(req.APIKey)
	if apiKey != "" && req.ClearAPIKey {
		return "", apperrors.BadRequest("apiKey and clearApiKey cannot be used together")
	}
	if req.ClearAPIKey {
		return "", nil
	}
	if apiKey != "" {
		return encryptSecret(apiKey, authSecret)
	}
	currentCipher = strings.TrimSpace(currentCipher)
	if currentCipher != "" && !strings.Contains(currentCipher, ":") {
		plainText, err := decryptAIAPIKey(currentCipher, authSecret)
		if err == nil {
			return encryptSecret(plainText, authSecret)
		}
	}
	return currentCipher, nil
}

func encryptSecret(plainText string, authSecret string) (string, error) {
	authSecret = strings.TrimSpace(authSecret)
	if authSecret == "" {
		return "", apperrors.BadRequest("auth access secret is not configured")
	}
	encoded, err := encryptSecretPayload(plainText, authSecret)
	if err != nil {
		return "", err
	}
	return secretCipherV3Prefix + encoded, nil
}

func decryptSecret(encoded string, authSecret string) (string, error) {
	return decryptAIAPIKey(encoded, authSecret)
}

func encryptLegacySecret(plainText string, authSecret string) (string, error) {
	return encryptSecretPayload(plainText, authSecret)
}

func decryptAIAPIKey(encoded, authSecret string) (string, error) {
	encoded = strings.TrimSpace(encoded)
	if encoded == "" {
		return "", errAIAPIKeyNeedsUpdate
	}
	if strings.HasPrefix(encoded, secretCipherV2Prefix) {
		return "", errAIAPIKeyNeedsUpdate
	}
	if strings.HasPrefix(encoded, secretCipherV3Prefix) {
		encoded = strings.TrimPrefix(encoded, secretCipherV3Prefix)
	}
	if strings.TrimSpace(authSecret) == "" {
		return "", errAIAPIKeyNeedsUpdate
	}
	plainText, err := decryptSecretPayload(encoded, authSecret)
	if err != nil {
		return "", fmt.Errorf("%w: %v", errAIAPIKeyNeedsUpdate, err)
	}
	return plainText, nil
}

func encryptSecretPayload(plainText string, secret string) (string, error) {
	key := aiEncryptionKey(secret)
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	cipherText := gcm.Seal(nil, nonce, []byte(plainText), nil)
	payload := append(nonce, cipherText...)
	return base64.StdEncoding.EncodeToString(payload), nil
}

func decryptSecretPayload(encoded string, secret string) (string, error) {
	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", err
	}
	key := aiEncryptionKey(secret)
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(raw) < gcm.NonceSize() {
		return "", fmt.Errorf("ciphertext is too short")
	}
	nonce := raw[:gcm.NonceSize()]
	cipherText := raw[gcm.NonceSize():]
	plainText, err := gcm.Open(nil, nonce, cipherText, nil)
	if err != nil {
		return "", err
	}
	return string(plainText), nil
}

func aiEncryptionKey(secret string) [32]byte {
	return sha256.Sum256([]byte("notes-of-ashen:ai-settings:" + secret))
}

// MapProviderError 将上游 AI 错误映射为稳定的本站业务错误，供 AI 业务调用复用。
func MapProviderError(ctx context.Context, operation string, err error) error {
	logx.WithContext(ctx).Errorf("ai provider %s failed: %v", operation, err)
	var netErr net.Error
	if errors.Is(err, context.DeadlineExceeded) || (errors.As(err, &netErr) && netErr.Timeout()) {
		return apperrors.GatewayTimeout("ai provider request timed out")
	}
	return apperrors.BadGateway("ai provider request failed")
}
