package ai

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"strings"

	"notes-of-ashen/internal/authutil"
	"notes-of-ashen/internal/config"
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
)

func Settings(ctx context.Context, svcCtx *svc.ServiceContext) (*types.AISettingsResp, error) {
	if err := authutil.RequireAdmin(ctx); err != nil {
		return nil, err
	}
	conf, configured, err := EffectiveConfig(ctx, svcCtx)
	if err != nil {
		return nil, err
	}
	return aiSettingsResp(conf, configured), nil
}

func UpdateSettings(ctx context.Context, svcCtx *svc.ServiceContext, req types.UpdateAISettingsReq) (*types.AISettingsResp, error) {
	if err := authutil.RequireAdmin(ctx); err != nil {
		return nil, err
	}
	current, err := svcCtx.Store.AISettings(ctx)
	if err != nil {
		return nil, err
	}

	baseURL := strings.TrimRight(strings.TrimSpace(req.BaseURL), "/")
	if err := validator.Length(baseURL, "baseUrl", 0, 255); err != nil {
		return nil, err
	}
	if err := validator.OptionalHTTPURL(baseURL, "baseUrl"); err != nil {
		return nil, err
	}
	modelName := strings.TrimSpace(req.Model)
	if err := validator.Length(modelName, "model", 0, 120); err != nil {
		return nil, err
	}
	firstByteTimeout, err := normalizeAITimeout(req.FirstByteTimeoutSeconds, model.DefaultAIFirstByteWait, "firstByteTimeoutSeconds")
	if err != nil {
		return nil, err
	}
	streamTimeout, err := normalizeAITimeout(req.StreamTimeoutSeconds, model.DefaultAIStreamWait, "streamTimeoutSeconds")
	if err != nil {
		return nil, err
	}
	nonStreamTimeout, err := normalizeAITimeout(req.NonStreamTimeoutSeconds, model.DefaultAINonStreamWait, "nonStreamTimeoutSeconds")
	if err != nil {
		return nil, err
	}
	if streamTimeout < firstByteTimeout {
		return nil, apperrors.BadRequest("streamTimeoutSeconds must be greater than or equal to firstByteTimeoutSeconds")
	}
	if nonStreamTimeout < firstByteTimeout {
		return nil, apperrors.BadRequest("nonStreamTimeoutSeconds must be greater than or equal to firstByteTimeoutSeconds")
	}

	apiKeyCipher, err := apiKeyCipherForUpdate(current.APIKeyCipher, req, svcCtx.Config)
	if err != nil {
		return nil, err
	}

	if err := svcCtx.Store.UpdateAISettings(ctx, model.AISettings{
		Enabled:                 req.Enabled,
		BaseURL:                 baseURL,
		APIKeyCipher:            apiKeyCipher,
		Model:                   modelName,
		FirstByteTimeoutSeconds: firstByteTimeout,
		StreamTimeoutSeconds:    streamTimeout,
		NonStreamTimeoutSeconds: nonStreamTimeout,
	}); err != nil {
		return nil, err
	}
	conf, configured, err := EffectiveConfig(ctx, svcCtx)
	if err != nil {
		return nil, err
	}
	return aiSettingsResp(conf, configured), nil
}

func EffectiveConfig(ctx context.Context, svcCtx *svc.ServiceContext) (config.AIConf, bool, error) {
	conf := normalizeAIConf(svcCtx.Config.AI)
	settings, err := svcCtx.Store.AISettings(ctx)
	if err != nil {
		return conf, hasAISecret(conf), err
	}

	if !settings.Configured {
		return conf, hasAISecret(conf), nil
	}

	conf.Enabled = settings.Enabled
	conf.BaseURL = strings.TrimSpace(settings.BaseURL)
	conf.Model = strings.TrimSpace(settings.Model)
	conf.FirstByteTimeoutSeconds = settings.FirstByteTimeoutSeconds
	conf.StreamTimeoutSeconds = settings.StreamTimeoutSeconds
	conf.NonStreamTimeoutSeconds = settings.NonStreamTimeoutSeconds
	if strings.TrimSpace(settings.APIKeyCipher) != "" {
		apiKey, err := decryptAIAPIKey(settings.APIKeyCipher, svcCtx.Config.AI.KeyEncryptionSecret, svcCtx.Config.Auth.AccessSecret)
		if err != nil {
			// 旧密文（无 v2: 前缀）在缺失 KeyEncryptionSecret 时不再回退到 Auth.AccessSecret 解密，
			// 避免 AccessSecret 轮换后旧密文不可读却被静默忽略。提示需迁移。
			logx.WithContext(ctx).Errorf("decrypt ai api key failed (cipher may need migration): %v", err)
			return conf, true, fmt.Errorf("decrypt ai api key: %w", err)
		}
		conf.APIKey = apiKey
	} else {
		conf.APIKey = ""
	}
	return normalizeAIConf(conf), strings.TrimSpace(settings.APIKeyCipher) != "", nil
}

func normalizeAIConf(conf config.AIConf) config.AIConf {
	if conf.FirstByteTimeoutSeconds <= 0 {
		conf.FirstByteTimeoutSeconds = model.DefaultAIFirstByteWait
	}
	if conf.StreamTimeoutSeconds <= 0 {
		conf.StreamTimeoutSeconds = model.DefaultAIStreamWait
	}
	if conf.NonStreamTimeoutSeconds <= 0 {
		if conf.TimeoutSeconds > 0 {
			conf.NonStreamTimeoutSeconds = conf.TimeoutSeconds
		} else {
			conf.NonStreamTimeoutSeconds = model.DefaultAINonStreamWait
		}
	}
	return conf
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

func aiSettingsResp(conf config.AIConf, apiKeyConfigured bool) *types.AISettingsResp {
	conf = normalizeAIConf(conf)
	return &types.AISettingsResp{
		Enabled:                 conf.Enabled,
		BaseURL:                 strings.TrimSpace(conf.BaseURL),
		Model:                   strings.TrimSpace(conf.Model),
		APIKeyConfigured:        apiKeyConfigured || strings.TrimSpace(conf.APIKey) != "",
		FirstByteTimeoutSeconds: conf.FirstByteTimeoutSeconds,
		StreamTimeoutSeconds:    conf.StreamTimeoutSeconds,
		NonStreamTimeoutSeconds: conf.NonStreamTimeoutSeconds,
	}
}

func hasAISecret(conf config.AIConf) bool {
	return strings.TrimSpace(conf.APIKey) != ""
}

func apiKeyCipherForUpdate(currentCipher string, req types.UpdateAISettingsReq, conf config.Config) (string, error) {
	apiKeyCipher := currentCipher
	if req.ClearAPIKey {
		return "", nil
	}
	apiKey := strings.TrimSpace(req.APIKey)
	if apiKey != "" {
		return encryptSecret(apiKey, conf.AI.KeyEncryptionSecret)
	}
	if shouldMigrateAPIKeyCipher(apiKeyCipher, conf.AI.KeyEncryptionSecret) {
		plainText, err := decryptSecret(apiKeyCipher, conf.Auth.AccessSecret)
		if err != nil {
			return "", fmt.Errorf("decrypt legacy ai api key: %w", err)
		}
		return encryptSecret(plainText, conf.AI.KeyEncryptionSecret)
	}
	return apiKeyCipher, nil
}

func encryptSecret(plainText string, secret string) (string, error) {
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return "", apperrors.BadRequest("ai key encryption secret is not configured")
	}
	encoded, err := encryptSecretPayload(plainText, secret)
	if err != nil {
		return "", err
	}
	return secretCipherV2Prefix + encoded, nil
}

func decryptSecret(encoded string, secret string) (string, error) {
	encoded = strings.TrimSpace(encoded)
	if !strings.HasPrefix(encoded, secretCipherV2Prefix) {
		return decryptSecretPayload(encoded, secret)
	}
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return "", apperrors.BadRequest("ai key encryption secret is not configured")
	}
	return decryptSecretPayload(strings.TrimPrefix(encoded, secretCipherV2Prefix), secret)
}

func encryptLegacySecret(plainText string, secret string) (string, error) {
	return encryptSecretPayload(plainText, secret)
}

func decryptAIAPIKey(encoded, keyEncryptionSecret, legacySecret string) (string, error) {
	encoded = strings.TrimSpace(encoded)
	if strings.HasPrefix(encoded, secretCipherV2Prefix) {
		return decryptSecret(encoded, keyEncryptionSecret)
	}
	// 旧密文（无 v2: 前缀）。安全策略：仅在配置了 KeyEncryptionSecret 时兼容回退解密，
	// 避免轮换 APP_AUTH_ACCESS_SECRET 后旧密文被静默用 legacy 密钥解密失败仍可读取的歧义；
	// 缺失 KeyEncryptionSecret 则直接 fail-closed，要求管理员重新填写 AI Key 完成迁移。
	if strings.TrimSpace(keyEncryptionSecret) == "" {
		return "", fmt.Errorf("legacy ai api key ciphertext requires KeyEncryptionSecret to decrypt")
	}
	// 优先尝试用 KeyEncryptionSecret 解密（兼容已迁移但前缀未更新的情形），失败再回退 legacySecret。
	if plain, err := decryptSecretPayload(encoded, keyEncryptionSecret); err == nil {
		return plain, nil
	}
	return decryptSecretPayload(encoded, legacySecret)
}

func shouldMigrateAPIKeyCipher(cipherText, keyEncryptionSecret string) bool {
	return strings.TrimSpace(cipherText) != "" &&
		!strings.HasPrefix(strings.TrimSpace(cipherText), secretCipherV2Prefix) &&
		strings.TrimSpace(keyEncryptionSecret) != ""
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
