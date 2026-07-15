package ai

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"notes-of-ashen/internal/aiclient"
	"notes-of-ashen/internal/authutil"
	"notes-of-ashen/internal/config"
	apperrors "notes-of-ashen/internal/errors"
	"notes-of-ashen/internal/svc"
	"notes-of-ashen/internal/types"
	"notes-of-ashen/model"
)

func TestEncryptSecretV3RoundTrip(t *testing.T) {
	cipherText, err := encryptSecret("sk-test", "access-secret")
	if err != nil {
		t.Fatalf("encryptSecret() error = %v", err)
	}
	if !strings.HasPrefix(cipherText, secretCipherV3Prefix) || strings.Contains(cipherText, "sk-test") {
		t.Fatalf("cipherText = %q, want encrypted v3 value", cipherText)
	}
	plainText, err := decryptSecret(cipherText, "access-secret")
	if err != nil {
		t.Fatalf("decryptSecret() error = %v", err)
	}
	if plainText != "sk-test" {
		t.Fatalf("plainText = %q, want sk-test", plainText)
	}
}

func TestAPIKeyStatusMarksWrongOrV2SecretForUpdate(t *testing.T) {
	cipherText, err := encryptSecret("sk-test", "access-secret")
	if err != nil {
		t.Fatal(err)
	}
	if configured, needsUpdate := apiKeyStatus(cipherText, "other-secret"); !configured || !needsUpdate {
		t.Fatalf("apiKeyStatus(wrong secret) = %v, %v", configured, needsUpdate)
	}
	if configured, needsUpdate := apiKeyStatus(secretCipherV2Prefix+"payload", "access-secret"); !configured || !needsUpdate {
		t.Fatalf("apiKeyStatus(v2) = %v, %v", configured, needsUpdate)
	}
	if _, err := decryptAIAPIKey(secretCipherV2Prefix+"payload", "access-secret"); !errors.Is(err, errAIAPIKeyNeedsUpdate) {
		t.Fatalf("decryptAIAPIKey(v2) error = %v", err)
	}
}

func TestAPIKeyCipherForUpdateMigratesLegacyCipher(t *testing.T) {
	legacyCipher, err := encryptLegacySecret("sk-test", "access-secret")
	if err != nil {
		t.Fatalf("encryptLegacySecret() error = %v", err)
	}
	cipherText, err := apiKeyCipherForUpdate(legacyCipher, types.UpdateAISettingsReq{}, "access-secret")
	if err != nil {
		t.Fatalf("apiKeyCipherForUpdate() error = %v", err)
	}
	if cipherText == legacyCipher || !strings.HasPrefix(cipherText, secretCipherV3Prefix) {
		t.Fatalf("cipherText = %q, want migrated v3 cipher", cipherText)
	}
	plainText, err := decryptAIAPIKey(cipherText, "access-secret")
	if err != nil || plainText != "sk-test" {
		t.Fatalf("decryptAIAPIKey() = %q, %v", plainText, err)
	}
}

func TestAPIKeyCipherForUpdateValidation(t *testing.T) {
	if _, err := apiKeyCipherForUpdate("", types.UpdateAISettingsReq{APIKey: "sk-test"}, ""); err == nil {
		t.Fatal("new key without auth secret should fail")
	}
	if _, err := apiKeyCipherForUpdate("old", types.UpdateAISettingsReq{APIKey: "new", ClearAPIKey: true}, "access-secret"); err == nil {
		t.Fatal("apiKey and clearApiKey together should fail")
	}
	cipherText, err := apiKeyCipherForUpdate("old", types.UpdateAISettingsReq{ClearAPIKey: true}, "")
	if err != nil || cipherText != "" {
		t.Fatalf("clear key = %q, %v", cipherText, err)
	}
}

func TestNormalizeAIConfDefaults(t *testing.T) {
	conf := normalizeAIConf(aiclient.Config{})
	if conf.APIFormat != model.AIAPIFormatOpenAI {
		t.Fatalf("APIFormat = %q", conf.APIFormat)
	}
	if conf.FirstByteTimeoutSeconds != model.DefaultAIFirstByteWait {
		t.Fatalf("FirstByteTimeoutSeconds = %d", conf.FirstByteTimeoutSeconds)
	}
	if conf.NonStreamTimeoutSeconds != model.DefaultAINonStreamWait {
		t.Fatalf("NonStreamTimeoutSeconds = %d", conf.NonStreamTimeoutSeconds)
	}
}

func TestNormalizeAIAPIFormat(t *testing.T) {
	for input, want := range map[string]string{
		"":          model.AIAPIFormatOpenAI,
		" OpenAI ":  model.AIAPIFormatOpenAI,
		"ANTHROPIC": model.AIAPIFormatAnthropic,
	} {
		got, err := normalizeAIAPIFormat(input)
		if err != nil || got != want {
			t.Fatalf("normalizeAIAPIFormat(%q) = %q, %v; want %q", input, got, err, want)
		}
	}
	if _, err := normalizeAIAPIFormat("unknown"); err == nil {
		t.Fatal("invalid api format should fail")
	}
}

func TestValidateAITimeouts(t *testing.T) {
	if _, _, err := validateAITimeouts(-1, 600, 610000); err == nil {
		t.Fatal("negative first byte timeout should fail")
	}
	if _, _, err := validateAITimeouts(int((31 * time.Minute).Seconds()), 600, 610000); err == nil {
		t.Fatal("out-of-range first byte timeout should fail")
	}
	if _, _, err := validateAITimeouts(60, 30, 610000); err == nil {
		t.Fatal("total timeout below first-byte timeout should fail")
	}
	if _, _, err := validateAITimeouts(60, 700, 610000); err == nil {
		t.Fatal("timeout above server request timeout should fail")
	}
	firstByte, total, err := validateAITimeouts(0, 0, 610000)
	if err != nil || firstByte != 60 || total != 600 {
		t.Fatalf("default timeouts = %d, %d, %v", firstByte, total, err)
	}
}

func TestSameAIEndpointNormalizesTrailingSlash(t *testing.T) {
	stored := model.AISettings{APIFormat: model.AIAPIFormatOpenAI, BaseURL: "https://api.example.com/v1/"}
	if !sameAIEndpoint(stored, model.AIAPIFormatOpenAI, "https://api.example.com/v1") {
		t.Fatal("equivalent endpoint should allow reusing the saved key")
	}
	if sameAIEndpoint(stored, model.AIAPIFormatAnthropic, "https://api.example.com/v1") {
		t.Fatal("different format must not reuse the saved key")
	}
	if sameAIEndpoint(stored, model.AIAPIFormatOpenAI, "https://other.example.com/v1") {
		t.Fatal("different base URL must not reuse the saved key")
	}
}

func TestValidateEnabledAISettingsRequiresUsableKey(t *testing.T) {
	settings := model.AISettings{
		Enabled:                 true,
		APIFormat:               model.AIAPIFormatOpenAI,
		BaseURL:                 "https://api.example.com/v1",
		Model:                   "model",
		APIKeyCipher:            secretCipherV2Prefix + "payload",
		FirstByteTimeoutSeconds: 60,
		NonStreamTimeoutSeconds: 600,
	}
	if err := validateEnabledAISettings(settings, "access-secret"); err == nil {
		t.Fatal("v2 key should require replacement before enabling")
	}
}

func TestAIConnectionOperationsRequireAdmin(t *testing.T) {
	editorCtx := authutil.WithUser(context.Background(), 1, authutil.RoleEditor)
	if _, err := Models(editorCtx, &svc.ServiceContext{}, types.AIConnectionReq{}); err == nil {
		t.Fatal("Models() should reject editor")
	}
	if _, err := TestModel(editorCtx, &svc.ServiceContext{}, types.AIModelTestReq{}); err == nil {
		t.Fatal("TestModel() should reject editor")
	}
}

func TestMapAIProviderErrorDoesNotForwardProviderAuthStatus(t *testing.T) {
	err := MapProviderError(context.Background(), "test", &aiclient.HTTPStatusError{StatusCode: http.StatusUnauthorized})
	var codeErr *apperrors.CodeError
	if !errors.As(err, &codeErr) || codeErr.StatusCode != http.StatusBadGateway {
		t.Fatalf("provider 401 mapped to %#v", err)
	}
	err = MapProviderError(context.Background(), "test", context.DeadlineExceeded)
	if !errors.As(err, &codeErr) || codeErr.StatusCode != http.StatusGatewayTimeout {
		t.Fatalf("timeout mapped to %#v", err)
	}
}

func TestDraftConfigReusesSavedKeyOnlyForSameEndpoint(t *testing.T) {
	cipherText, err := encryptSecret("saved-key", "access-secret")
	if err != nil {
		t.Fatal(err)
	}
	svcCtx, mock, closeDB := newAISettingsMockContext(t)
	defer closeDB()
	mock.ExpectQuery("SELECT setting_key, setting_value FROM site_settings").
		WillReturnRows(aiSettingsRows(cipherText, "https://api.example.com/v1", model.AIAPIFormatOpenAI))

	conf, err := draftConfig(context.Background(), svcCtx, draftConfigInput{
		APIFormat: model.AIAPIFormatOpenAI,
		BaseURL:   "https://api.example.com/v1/",
	})
	if err != nil {
		t.Fatalf("draftConfig() error = %v", err)
	}
	if conf.APIKey != "saved-key" {
		t.Fatalf("APIKey = %q", conf.APIKey)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestDraftConfigRejectsSavedKeyForDifferentEndpoint(t *testing.T) {
	cipherText, err := encryptSecret("saved-key", "access-secret")
	if err != nil {
		t.Fatal(err)
	}
	svcCtx, mock, closeDB := newAISettingsMockContext(t)
	defer closeDB()
	mock.ExpectQuery("SELECT setting_key, setting_value FROM site_settings").
		WillReturnRows(aiSettingsRows(cipherText, "https://api.example.com/v1", model.AIAPIFormatOpenAI))

	_, err = draftConfig(context.Background(), svcCtx, draftConfigInput{
		APIFormat: model.AIAPIFormatAnthropic,
		BaseURL:   "https://api.example.com/v1",
	})
	if err == nil || !strings.Contains(err.Error(), "api key is required") {
		t.Fatalf("draftConfig() error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestDraftConfigUsesTemporaryKeyWithoutReadingSettings(t *testing.T) {
	svcCtx, mock, closeDB := newAISettingsMockContext(t)
	defer closeDB()
	conf, err := draftConfig(context.Background(), svcCtx, draftConfigInput{
		APIFormat: model.AIAPIFormatAnthropic,
		BaseURL:   "https://api.anthropic.com/v1",
		APIKey:    "temporary-key",
		Model:     "claude-model",
	})
	if err != nil {
		t.Fatalf("draftConfig() error = %v", err)
	}
	if conf.APIKey != "temporary-key" || conf.Model != "claude-model" {
		t.Fatalf("unexpected config: %#v", conf)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestUpdateSettingsPreservesOmittedConnectionFields(t *testing.T) {
	svcCtx, mock, closeDB := newAISettingsMockContext(t)
	defer closeDB()
	mock.ExpectQuery("SELECT setting_key, setting_value FROM site_settings").
		WillReturnRows(aiSettingsRows("cipher-text", "https://api.example.com/v1", model.AIAPIFormatOpenAI))
	mock.ExpectExec("INSERT INTO site_settings").
		WithArgs(
			model.AIEnabledKey, "false",
			model.AIAPIFormatKey, model.AIAPIFormatOpenAI,
			model.AIBaseURLKey, "https://api.example.com/v1",
			model.AIAPIKeyCipherKey, "cipher-text",
			model.AIModelKey, "model",
			model.AIFirstByteTimeoutKey, "60",
			model.AINonStreamTimeoutKey, "600",
		).
		WillReturnResult(sqlmock.NewResult(0, 7))

	ctx := authutil.WithUser(context.Background(), 1, authutil.RoleAdmin)
	resp, err := UpdateSettings(ctx, svcCtx, types.UpdateAISettingsReq{Enabled: false})
	if err != nil {
		t.Fatalf("UpdateSettings() error = %v", err)
	}
	if resp.BaseURL != "https://api.example.com/v1" || resp.Model != "model" || resp.FirstByteTimeoutSeconds != 60 || resp.NonStreamTimeoutSeconds != 600 {
		t.Fatalf("omitted fields were not preserved: %#v", resp)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestUpdateSettingsCanClearDisabledEndpointAndResetTimeouts(t *testing.T) {
	svcCtx, mock, closeDB := newAISettingsMockContext(t)
	defer closeDB()
	mock.ExpectQuery("SELECT setting_key, setting_value FROM site_settings").
		WillReturnRows(aiSettingsRows("cipher-text", "https://api.example.com/v1", model.AIAPIFormatOpenAI))
	mock.ExpectExec("INSERT INTO site_settings").
		WithArgs(
			model.AIEnabledKey, "false",
			model.AIAPIFormatKey, model.AIAPIFormatOpenAI,
			model.AIBaseURLKey, "",
			model.AIAPIKeyCipherKey, "cipher-text",
			model.AIModelKey, "",
			model.AIFirstByteTimeoutKey, "60",
			model.AINonStreamTimeoutKey, "600",
		).
		WillReturnResult(sqlmock.NewResult(0, 7))

	empty := ""
	zero := 0
	ctx := authutil.WithUser(context.Background(), 1, authutil.RoleAdmin)
	resp, err := UpdateSettings(ctx, svcCtx, types.UpdateAISettingsReq{
		Enabled:                 false,
		BaseURL:                 &empty,
		Model:                   &empty,
		FirstByteTimeoutSeconds: &zero,
		NonStreamTimeoutSeconds: &zero,
	})
	if err != nil {
		t.Fatalf("UpdateSettings() error = %v", err)
	}
	if resp.BaseURL != "" || resp.Model != "" || resp.FirstByteTimeoutSeconds != 60 || resp.NonStreamTimeoutSeconds != 600 {
		t.Fatalf("explicit clear/default fields were not applied: %#v", resp)
	}
	if !resp.APIKeyConfigured {
		t.Fatal("clearing a disabled endpoint should preserve the existing API key")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func newAISettingsMockContext(t *testing.T) (*svc.ServiceContext, sqlmock.Sqlmock, func()) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	conf := config.Config{Auth: config.AuthConf{AccessSecret: "access-secret"}}
	conf.Timeout = 610000
	return &svc.ServiceContext{Config: conf, Store: model.NewStore(db)}, mock, func() { _ = db.Close() }
}

func aiSettingsRows(cipherText, baseURL, apiFormat string) *sqlmock.Rows {
	return sqlmock.NewRows([]string{"setting_key", "setting_value"}).
		AddRow(model.AIEnabledKey, "false").
		AddRow(model.AIAPIFormatKey, apiFormat).
		AddRow(model.AIBaseURLKey, baseURL).
		AddRow(model.AIAPIKeyCipherKey, cipherText).
		AddRow(model.AIModelKey, "model").
		AddRow(model.AIFirstByteTimeoutKey, "60").
		AddRow(model.AINonStreamTimeoutKey, "600")
}
