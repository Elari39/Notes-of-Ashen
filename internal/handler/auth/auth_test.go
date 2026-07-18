package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"notes-of-ashen/internal/types"
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

func TestResolveRefreshTokenPrefersBodyAndFallsBackToCookie(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
	req.AddCookie(&http.Cookie{Name: "noa_refresh_token", Value: "cookie-token"})

	if got := resolveRefreshToken(req, types.RefreshReq{}).RefreshToken; got != "cookie-token" {
		t.Fatalf("cookie fallback = %q, want cookie-token", got)
	}
	if got := resolveRefreshToken(req, types.RefreshReq{RefreshToken: "body-token"}).RefreshToken; got != "body-token" {
		t.Fatalf("body token = %q, want body-token", got)
	}
	if got := resolveRefreshToken(httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil), types.RefreshReq{}).RefreshToken; got != "" {
		t.Fatalf("missing token = %q, want empty", got)
	}
}
