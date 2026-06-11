package ai

import (
	"testing"
	"time"

	"notes-of-ashen/internal/config"
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
