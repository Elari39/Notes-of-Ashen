package ai

import (
	"strings"
	"testing"
	"time"

	"notes-of-ashen/internal/config"
	"notes-of-ashen/internal/types"
	"notes-of-ashen/model"
)

func TestEncryptSecretRoundTrip(t *testing.T) {
	cipherText, err := encryptSecret("sk-test", "access-secret")
	if err != nil {
		t.Fatalf("encryptSecret() error = %v", err)
	}
	if cipherText == "" || cipherText == "sk-test" {
		t.Fatalf("cipherText = %q, want encrypted value", cipherText)
	}
	if !strings.HasPrefix(cipherText, secretCipherV2Prefix) {
		t.Fatalf("cipherText = %q, want v2 prefix", cipherText)
	}
	plainText, err := decryptSecret(cipherText, "access-secret")
	if err != nil {
		t.Fatalf("decryptSecret() error = %v", err)
	}
	if plainText != "sk-test" {
		t.Fatalf("plainText = %q, want sk-test", plainText)
	}
}

func TestDecryptSecretRejectsWrongSecret(t *testing.T) {
	cipherText, err := encryptSecret("sk-test", "access-secret")
	if err != nil {
		t.Fatalf("encryptSecret() error = %v", err)
	}
	if _, err := decryptSecret(cipherText, "other-secret"); err == nil {
		t.Fatal("decryptSecret() error = nil, want error")
	}
}

func TestDecryptSecretSupportsLegacyCipher(t *testing.T) {
	cipherText, err := encryptLegacySecret("sk-test", "access-secret")
	if err != nil {
		t.Fatalf("encryptLegacySecret() error = %v", err)
	}
	if strings.HasPrefix(cipherText, secretCipherV2Prefix) {
		t.Fatalf("legacy cipherText = %q, should not have v2 prefix", cipherText)
	}
	plainText, err := decryptSecret(cipherText, "access-secret")
	if err != nil {
		t.Fatalf("decryptSecret() error = %v", err)
	}
	if plainText != "sk-test" {
		t.Fatalf("plainText = %q, want sk-test", plainText)
	}
}

func TestEncryptSecretRequiresConfiguredSecret(t *testing.T) {
	if _, err := encryptSecret("sk-test", ""); err == nil {
		t.Fatal("encryptSecret() error = nil, want error")
	}
}

func TestAPIKeyCipherForUpdateMigratesLegacyCipher(t *testing.T) {
	legacyCipher, err := encryptLegacySecret("sk-test", "access-secret")
	if err != nil {
		t.Fatalf("encryptLegacySecret() error = %v", err)
	}

	cipherText, err := apiKeyCipherForUpdate(legacyCipher, types.UpdateAISettingsReq{}, config.Config{
		Auth: config.AuthConf{AccessSecret: "access-secret"},
		AI:   config.AIConf{KeyEncryptionSecret: "ai-key-secret"},
	})
	if err != nil {
		t.Fatalf("apiKeyCipherForUpdate() error = %v", err)
	}
	if cipherText == legacyCipher || !strings.HasPrefix(cipherText, secretCipherV2Prefix) {
		t.Fatalf("cipherText = %q, want migrated v2 cipher", cipherText)
	}
	plainText, err := decryptAIAPIKey(cipherText, "ai-key-secret", "access-secret")
	if err != nil {
		t.Fatalf("decryptAIAPIKey() error = %v", err)
	}
	if plainText != "sk-test" {
		t.Fatalf("plainText = %q, want sk-test", plainText)
	}
}

func TestAPIKeyCipherForUpdateRejectsNewKeyWithoutEncryptionSecret(t *testing.T) {
	_, err := apiKeyCipherForUpdate("", types.UpdateAISettingsReq{APIKey: "sk-test"}, config.Config{})
	if err == nil {
		t.Fatal("apiKeyCipherForUpdate() error = nil, want error")
	}
}

func TestAPIKeyCipherForUpdateClearSkipsEncryptionSecret(t *testing.T) {
	cipherText, err := apiKeyCipherForUpdate("legacy-cipher", types.UpdateAISettingsReq{ClearAPIKey: true}, config.Config{})
	if err != nil {
		t.Fatalf("apiKeyCipherForUpdate() error = %v", err)
	}
	if cipherText != "" {
		t.Fatalf("cipherText = %q, want empty", cipherText)
	}
}

func TestNormalizeAIConfDefaults(t *testing.T) {
	conf := normalizeAIConf(config.AIConf{})
	if conf.FirstByteTimeoutSeconds != model.DefaultAIFirstByteWait {
		t.Fatalf("FirstByteTimeoutSeconds = %d", conf.FirstByteTimeoutSeconds)
	}
	if conf.StreamTimeoutSeconds != model.DefaultAIStreamWait {
		t.Fatalf("StreamTimeoutSeconds = %d", conf.StreamTimeoutSeconds)
	}
	if conf.NonStreamTimeoutSeconds != model.DefaultAINonStreamWait {
		t.Fatalf("NonStreamTimeoutSeconds = %d", conf.NonStreamTimeoutSeconds)
	}
}

func TestNormalizeAITimeoutRejectsOutOfRange(t *testing.T) {
	if _, err := normalizeAITimeout(int((31 * time.Minute).Seconds()), 600, "timeout"); err == nil {
		t.Fatal("normalizeAITimeout() error = nil, want error")
	}
}
