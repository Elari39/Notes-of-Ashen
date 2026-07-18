package article

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAIAssistHandlerRejectsOversizedRequestBody(t *testing.T) {
	body := []byte(`{"action":"metadata","content":"` + strings.Repeat("x", int(maxAIAssistRequestBodySize)) + `"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/articles/ai/assist", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	AIAssistHandler(nil).ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}
