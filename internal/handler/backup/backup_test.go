package backup

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"notes-of-ashen/internal/authutil"
)

type panicReadCloser struct{}

func (panicReadCloser) Read([]byte) (int, error) { panic("request body must not be read") }
func (panicReadCloser) Close() error             { return nil }

var _ io.ReadCloser = panicReadCloser{}

func TestBackupHandlersRejectNonAdminBeforeParsingBody(t *testing.T) {
	for name, handler := range map[string]http.HandlerFunc{
		"export":  ExportHandler(nil),
		"restore": RestoreHandler(nil),
	} {
		t.Run(name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/backups/"+name, nil)
			req = req.WithContext(authutil.WithUser(req.Context(), 10, authutil.RoleEditor))
			req.Header.Set("Content-Type", "multipart/form-data; boundary=test")
			req.Body = panicReadCloser{}
			recorder := httptest.NewRecorder()

			handler(recorder, req)

			if recorder.Code != http.StatusForbidden {
				t.Fatalf("status = %d, want %d; body = %s", recorder.Code, http.StatusForbidden, recorder.Body.String())
			}
		})
	}
}
