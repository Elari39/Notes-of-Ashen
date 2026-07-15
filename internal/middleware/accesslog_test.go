package middleware

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	basehandler "notes-of-ashen/internal/httphelper"

	"github.com/zeromicro/go-zero/core/logx"
)

func TestAccessLogDoesNotLeakRequestSecrets(t *testing.T) {
	var logs bytes.Buffer
	previousWriter := logx.Reset()
	logx.SetWriter(logx.NewWriter(&logs))
	t.Cleanup(func() {
		logx.SetWriter(previousWriter)
	})

	const (
		querySecret  = "query-secret"
		headerSecret = "header-secret"
		cookieSecret = "cookie-secret"
		bodySecret   = "body-secret"
		requestID    = "safe-request-id"
	)
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/admin/ai/test?apiKey="+querySecret,
		strings.NewReader(`{"apiKey":"`+bodySecret+`"}`),
	)
	req.RemoteAddr = "172.18.0.5:54321"
	req.Header.Set("Authorization", "Bearer "+headerSecret)
	req.Header.Set("Cookie", "refresh_token="+cookieSecret)
	req.Header.Set("X-Forwarded-For", "198.51.100.20, 172.18.0.5")
	req.Header.Set("X-Request-Id", requestID)

	handler := RequestID(NewAccessLogMiddleware(basehandler.ForwardedOptions{
		TrustedProxyCIDRs: "172.18.0.0/16",
	}).Handle(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))

	recorder := httptest.NewRecorder()
	handler(recorder, req)
	output := logs.String()

	for _, secret := range []string{querySecret, headerSecret, cookieSecret, bodySecret} {
		if strings.Contains(output, secret) {
			t.Fatalf("access log leaked secret %q: %s", secret, output)
		}
	}
	for _, expected := range []string{
		"POST",
		"/api/v1/admin/ai/test",
		"502",
		"198.51.100.20",
		requestID,
		"duration=",
	} {
		if !strings.Contains(output, expected) {
			t.Fatalf("access log missing %q: %s", expected, output)
		}
	}
}

func TestAccessLogPreservesFlusher(t *testing.T) {
	handler := NewAccessLogMiddleware().Handle(func(w http.ResponseWriter, _ *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Fatal("wrapped response writer does not implement http.Flusher")
		}
		flusher.Flush()
	})

	recorder := httptest.NewRecorder()
	handler(recorder, httptest.NewRequest(http.MethodGet, "/events", nil))
	if !recorder.Flushed {
		t.Fatal("wrapped response writer did not forward Flush")
	}
}
