package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLoginHandlerMalformedJSONReturnsBadRequest(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader("{"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	LoginHandler(nil).ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}

	var body struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response body: %v", err)
	}
	if body.Code != 40000 {
		t.Fatalf("code = %d, want 40000; message = %q", body.Code, body.Message)
	}
}
