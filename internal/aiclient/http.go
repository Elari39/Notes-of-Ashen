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

	"notes-of-ashen/internal/validator"
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
	dialer := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}
	httpConnPool = &http.Client{
		Transport: &http.Transport{
			// AI 出站请求不继承环境代理。通用 HTTP 代理会在代理端再次解析目标域名，
			// 使本进程无法保证连接落到已校验的公网 IP。
			Proxy:                 nil,
			DialContext:           newPublicDialContext(net.DefaultResolver.LookupIP, dialer.DialContext),
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

// NewPublicHTTPClient 返回用于管理员可配置 AI 上游的受限 HTTP 客户端。
// 它禁止环境代理、在每次拨号时重新做公网 DNS/地址校验并禁止重定向，供 RAG
// 等同样需要连接第三方模型服务的模块复用，避免各处自行实现不一致的 SSRF 防护。
// 调用方仍必须对保存阶段的 URL 使用 validator.OptionalHTTPURL 做语法校验。
func NewPublicHTTPClient(headerTimeout time.Duration) *http.Client {
	if headerTimeout <= 0 {
		headerTimeout = defaultFirstByteTimeout
	}
	return sharedHTTPClient(headerTimeout)
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

	client := conf.httpClient
	if client == nil {
		client = sharedHTTPClient(firstByteTimeout(conf))
	}
	httpResp, err := client.Do(httpReq)
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

type lookupIPFunc func(context.Context, string, string) ([]net.IP, error)
type dialContextFunc func(context.Context, string, string) (net.Conn, error)

// newPublicDialContext 在每次新建 TCP 连接前重新解析目标，并直接连接已校验的
// 公网 IP。解析结果只要包含一个受限地址就整体拒绝，避免攻击者借助公私混合记录
// 或 DNS rebinding 让后续连接落到内网。若 443 端口的域名仅解析到
// 198.18.0.0/15，则视为透明代理的 Fake-IP，保留原域名交给系统拨号；显式填写
// 该网段的 IP 或使用其他端口仍会拒绝。
func newPublicDialContext(lookup lookupIPFunc, dial dialContextFunc) dialContextFunc {
	return func(ctx context.Context, network, address string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, fmt.Errorf("parse ai endpoint address: %w", err)
		}

		var ips []net.IP
		if literal := net.ParseIP(host); literal != nil {
			if validator.IsBlockedHostIP(literal) {
				return nil, fmt.Errorf("ai endpoint resolved to a blocked address")
			}
			ips = []net.IP{literal}
		} else {
			ips, err = lookup(ctx, "ip", host)
			if err != nil {
				return nil, fmt.Errorf("resolve ai endpoint: %w", err)
			}
		}
		if len(ips) == 0 {
			return nil, fmt.Errorf("resolve ai endpoint: no addresses")
		}
		publicIPs := make([]net.IP, 0, len(ips))
		fakeIPCount := 0
		for _, ip := range ips {
			if validator.IsProxyFakeIP(ip) {
				fakeIPCount++
				continue
			}
			if validator.IsBlockedHostIP(ip) {
				return nil, fmt.Errorf("ai endpoint resolved to a blocked address")
			}
			publicIPs = append(publicIPs, ip)
		}
		if len(publicIPs) == 0 && fakeIPCount > 0 {
			if port != "443" {
				return nil, fmt.Errorf("ai endpoint resolved to a blocked address")
			}
			conn, dialErr := dial(ctx, network, address)
			if dialErr != nil {
				return nil, fmt.Errorf("dial ai endpoint through fake-ip proxy: %w", dialErr)
			}
			return conn, nil
		}
		ips = publicIPs

		var lastErr error
		for _, ip := range ips {
			conn, dialErr := dial(ctx, network, net.JoinHostPort(ip.String(), port))
			if dialErr == nil {
				return conn, nil
			}
			lastErr = dialErr
		}
		return nil, fmt.Errorf("dial ai endpoint: %w", lastErr)
	}
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
