package rag

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strings"
)

const ragSecretCipherV3Prefix = "v3:"

// RAG API Key 使用与文章 AI 助手相同的 AES-GCM v3 密文格式，但采用独立用途
// 派生域。这样即使同一 APP_AUTH_ACCESS_SECRET 被轮换或某个配置字段泄露，也不能
// 横向解密另一类设置；同时它不依赖 logic/ai，避免低层 Worker 出现依赖环。
func encryptAPIKey(value, authSecret string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil
	}
	if strings.TrimSpace(authSecret) == "" {
		return "", errors.New("auth access secret is not configured")
	}
	key := ragEncryptionKey(authSecret)
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
	payload := append(nonce, gcm.Seal(nil, nonce, []byte(value), nil)...)
	return ragSecretCipherV3Prefix + base64.StdEncoding.EncodeToString(payload), nil
}

func decryptAPIKey(value, authSecret string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", errors.New("rag api key is required")
	}
	if !strings.HasPrefix(value, ragSecretCipherV3Prefix) || strings.TrimSpace(authSecret) == "" {
		return "", errors.New("rag api key needs update")
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(value, ragSecretCipherV3Prefix))
	if err != nil {
		return "", fmt.Errorf("rag api key needs update: %w", err)
	}
	key := ragEncryptionKey(authSecret)
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(raw) < gcm.NonceSize() {
		return "", errors.New("rag api key needs update")
	}
	plain, err := gcm.Open(nil, raw[:gcm.NonceSize()], raw[gcm.NonceSize():], nil)
	if err != nil {
		return "", fmt.Errorf("rag api key needs update: %w", err)
	}
	return string(plain), nil
}

func apiKeyStatus(value, authSecret string) (configured, needsUpdate bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return false, false
	}
	_, err := decryptAPIKey(value, authSecret)
	return true, err != nil
}

func ragEncryptionKey(secret string) [32]byte {
	return sha256.Sum256([]byte("notes-of-ashen:rag-settings:" + secret))
}
