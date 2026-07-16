package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"notes-of-ashen/internal/security"
)

func TestMaintenanceBlocksConcurrentRestoreBeforeBodyParsing(t *testing.T) {
	if !security.TryStartRestore() {
		t.Fatal("could not start test restore state")
	}
	defer security.EndRestore()

	middleware := NewMaintenanceMiddleware(nil)
	called := false
	handler := middleware.Handle(func(http.ResponseWriter, *http.Request) {
		called = true
	})
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/backups/restore", nil)

	handler(recorder, req)

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusServiceUnavailable)
	}
	if called {
		t.Fatal("restore handler was called while maintenance was active")
	}
}
