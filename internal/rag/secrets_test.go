package rag

import (
	"strings"
	"testing"
)

func TestRAGAPIKeyUsesIndependentV3Cipher(t *testing.T) {
	const secret = "test-auth-secret-long-enough"
	const value = "rag-test-key"
	cipher, err := EncryptAPIKey(value, secret)
	if err != nil {
		t.Fatalf("EncryptAPIKey() error = %v", err)
	}
	if !strings.HasPrefix(cipher, "v3:") {
		t.Fatalf("cipher = %q, want v3 prefix", cipher)
	}
	configured, needsUpdate := APIKeyStatus(cipher, secret)
	if !configured || needsUpdate {
		t.Fatalf("APIKeyStatus(valid) = (%t, %t), want (true, false)", configured, needsUpdate)
	}
	configured, needsUpdate = APIKeyStatus(cipher, "different-auth-secret")
	if !configured || !needsUpdate {
		t.Fatalf("APIKeyStatus(rotated secret) = (%t, %t), want (true, true)", configured, needsUpdate)
	}
}

func TestRAGAPIKeyRejectsLegacyOrEmptyCipher(t *testing.T) {
	for _, value := range []string{"", "v2:legacy", "plaintext"} {
		configured, needsUpdate := APIKeyStatus(value, "test-auth-secret-long-enough")
		if value == "" {
			if configured || needsUpdate {
				t.Fatalf("APIKeyStatus(empty) = (%t, %t), want (false, false)", configured, needsUpdate)
			}
			continue
		}
		if !configured || !needsUpdate {
			t.Fatalf("APIKeyStatus(%q) = (%t, %t), want (true, true)", value, configured, needsUpdate)
		}
	}
}
