package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"notes-of-ashen/internal/response"
)

func TestRequestIDPreservesValidClientValue(t *testing.T) {
	const requestID = "client_trace-123"
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Request-Id", requestID)
	recorder := httptest.NewRecorder()

	RequestID(func(_ http.ResponseWriter, r *http.Request) {
		if got := response.RequestIDFromContext(r.Context()); got != requestID {
			t.Fatalf("RequestIDFromContext() = %q, want %q", got, requestID)
		}
		if got := response.RequestIDSourceFromContext(r.Context()); got != "client" {
			t.Fatalf("RequestIDSourceFromContext() = %q, want client", got)
		}
	})(recorder, req)

	if got := recorder.Header().Get("X-Request-Id"); got != requestID {
		t.Fatalf("response X-Request-Id = %q, want %q", got, requestID)
	}
}

func TestRequestIDReplacesInvalidClientValues(t *testing.T) {
	for _, invalid := range []string{"", "contains space", "中文", strings.Repeat("a", maxRequestIDLength+1)} {
		t.Run("invalid", func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("X-Request-Id", invalid)
			recorder := httptest.NewRecorder()

			RequestID(func(_ http.ResponseWriter, r *http.Request) {
				id := response.RequestIDFromContext(r.Context())
				if !validRequestID(id) {
					t.Fatalf("generated id %q is invalid", id)
				}
				if got := response.RequestIDSourceFromContext(r.Context()); got != "server" {
					t.Fatalf("RequestIDSourceFromContext() = %q, want server", got)
				}
			})(recorder, req)

			if got := recorder.Header().Get("X-Request-Id"); !validRequestID(got) || got == invalid {
				t.Fatalf("response X-Request-Id = %q, want newly generated valid id", got)
			}
		})
	}
}
