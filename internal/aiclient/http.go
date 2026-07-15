package aiclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const (
	defaultFirstByteTimeout = 60 * time.Second
	defaultNonStreamTimeout = 600 * time.Second
	maxResponseBodyBytes    = 1 << 20
	maxErrorMessageBytes    = 4096
)

var errResponseBodyTooLarge = errors.New("ai response body exceeds limit")

// httpConnPool 复用 TCP/TLS 连接池，ResponseHeaderTimeout 负责首字节超时，
// 单次请求总超时由 ctx 控制。首字节超时变化时重建 Client 并关闭旧空闲连接。
var (
	httpConnPoolMu sync.Mutex
	httpConnPool   *http.Client
	httpConnPoolTo time.Duration
)

func sharedHTTPClient(headerTimeout time.Duration) *http.Client {
	httpConnPoolMu.Lock()
	defer httpConnPoolMu.Unlock()
	if httpConnPool != nil && httpConnPoolTo == headerTimeout {
		return httpConnPool
	}
	if httpConnPool != nil {
		httpConnPool.CloseIdleConnections()
	}
	httpConnPool = &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			MaxIdleConns:          20,
			MaxIdleConnsPerHost:   10,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   30 * time.Second,
			ResponseHeaderTimeout: headerTimeout,
			ExpectContinueTimeout: 1 * time.Second,
		},
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	httpConnPoolTo = headerTimeout
	return httpConnPool
}

func doJSONRequest(ctx context.Context, conf Config, format, method, endpoint string, payload any) ([]byte, error) {
	var body io.Reader
	if payload != nil {
		raw, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("encode ai request: %w", err)
		}
		body = bytes.NewReader(raw)
	}

	requestCtx, cancel := context.WithTimeout(ctx, nonStreamTimeout(conf))
	defer cancel()
	httpReq, err := http.NewRequestWithContext(requestCtx, method, endpoint, body)
	if err != nil {
		return nil, fmt.Errorf("create ai request: %w", err)
	}
	httpReq.Header.Set("Accept", "application/json")
	if payload != nil {
		httpReq.Header.Set("Content-Type", "application/json")
	}
	apiKey := strings.TrimSpace(conf.APIKey)
	if format == APIFormatAnthropic {
		if apiKey != "" {
			httpReq.Header.Set("x-api-key", apiKey)
		}
		httpReq.Header.Set("anthropic-version", "2023-06-01")
	} else if apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	}

	httpResp, err := sharedHTTPClient(firstByteTimeout(conf)).Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send ai request: %w", err)
	}
	defer httpResp.Body.Close()

	raw, readErr := readResponseBody(httpResp.Body)
	if httpResp.StatusCode < http.StatusOK || httpResp.StatusCode >= http.StatusMultipleChoices {
		message := http.StatusText(httpResp.StatusCode)
		switch {
		case errors.Is(readErr, errResponseBodyTooLarge):
			message = errResponseBodyTooLarge.Error()
		case readErr != nil:
			message = "failed to read ai error response"
		case len(raw) > 0:
			message = upstreamErrorMessage(raw, conf.APIKey)
		}
		return nil, newHTTPStatusError(httpResp.StatusCode, message, conf.APIKey)
	}
	if readErr != nil {
		return nil, readErr
	}
	return raw, nil
}

func readResponseBody(body io.Reader) ([]byte, error) {
	raw, err := io.ReadAll(io.LimitReader(body, maxResponseBodyBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read ai response: %w", err)
	}
	if len(raw) > maxResponseBodyBytes {
		return nil, errResponseBodyTooLarge
	}
	return raw, nil
}

func upstreamErrorMessage(raw []byte, apiKey string) string {
	var envelope struct {
		Error   json.RawMessage `json:"error"`
		Message string          `json:"message"`
	}
	if json.Unmarshal(raw, &envelope) == nil {
		if len(envelope.Error) > 0 {
			var object providerError
			if json.Unmarshal(envelope.Error, &object) == nil && strings.TrimSpace(object.Message) != "" {
				return truncateErrorMessage(redactSecret(object.Message, apiKey))
			}
			var message string
			if json.Unmarshal(envelope.Error, &message) == nil && strings.TrimSpace(message) != "" {
				return truncateErrorMessage(redactSecret(message, apiKey))
			}
		}
		if strings.TrimSpace(envelope.Message) != "" {
			return truncateErrorMessage(redactSecret(envelope.Message, apiKey))
		}
	}
	return truncateErrorMessage(redactSecret(string(raw), apiKey))
}

func truncateErrorMessage(message string) string {
	message = strings.TrimSpace(message)
	if len(message) <= maxErrorMessageBytes {
		return message
	}
	return message[:maxErrorMessageBytes] + "..."
}

type endpointKind int

const (
	endpointCompletion endpointKind = iota
	endpointModels
)

func endpointFor(baseURL, format string, kind endpointKind) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil {
		return "", fmt.Errorf("parse ai base url: %w", err)
	}

	desiredPath := "/models"
	if kind == endpointCompletion {
		if format == APIFormatAnthropic {
			desiredPath = "/messages"
		} else {
			desiredPath = "/chat/completions"
		}
	}
	basePath := strings.TrimRight(parsed.Path, "/")
	hadKnownEndpoint := false
	for _, knownPath := range []string{"/chat/completions", "/messages", "/models"} {
		if strings.HasSuffix(basePath, knownPath) {
			basePath = strings.TrimSuffix(basePath, knownPath)
			hadKnownEndpoint = true
			break
		}
	}
	if format == APIFormatAnthropic && !hadKnownEndpoint && !strings.HasSuffix(basePath, "/v1") {
		basePath = strings.TrimRight(basePath, "/") + "/v1"
	}
	parsed.Path = strings.TrimRight(basePath, "/") + desiredPath
	parsed.RawPath = ""
	return parsed.String(), nil
}

func chatCompletionsEndpoint(baseURL string) string {
	endpoint, err := endpointFor(baseURL, APIFormatOpenAI, endpointCompletion)
	if err != nil {
		return strings.TrimRight(strings.TrimSpace(baseURL), "/") + "/chat/completions"
	}
	return endpoint
}

func firstByteTimeout(conf Config) time.Duration {
	seconds := conf.FirstByteTimeoutSeconds
	if seconds <= 0 {
		return defaultFirstByteTimeout
	}
	return time.Duration(seconds) * time.Second
}

func nonStreamTimeout(conf Config) time.Duration {
	seconds := conf.NonStreamTimeoutSeconds
	if seconds <= 0 {
		return defaultNonStreamTimeout
	}
	return time.Duration(seconds) * time.Second
}

func newHTTPStatusError(statusCode int, message, apiKey string) error {
	return &HTTPStatusError{
		StatusCode: statusCode,
		Message:    redactSecret(message, apiKey),
	}
}

type redactedError struct {
	cause   error
	message string
}

func (e *redactedError) Error() string {
	return e.message
}

func (e *redactedError) Unwrap() error {
	return e.cause
}

func sanitizeError(err error, apiKey string) error {
	if err == nil {
		return nil
	}
	message := redactSecret(err.Error(), apiKey)
	if message == err.Error() {
		return err
	}
	return &redactedError{cause: err, message: message}
}

func redactSecret(message, apiKey string) string {
	secrets := []string{apiKey, strings.TrimSpace(apiKey)}
	for _, secret := range secrets {
		if secret == "" {
			continue
		}
		message = strings.ReplaceAll(message, secret, "[REDACTED]")
		if escaped := url.QueryEscape(secret); escaped != secret {
			message = strings.ReplaceAll(message, escaped, "[REDACTED]")
		}
		if escaped := url.PathEscape(secret); escaped != secret {
			message = strings.ReplaceAll(message, escaped, "[REDACTED]")
		}
	}
	return message
}
