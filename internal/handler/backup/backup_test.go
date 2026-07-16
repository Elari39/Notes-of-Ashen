package backup

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"

	"notes-of-ashen/internal/authutil"
	"notes-of-ashen/internal/svc"
	"notes-of-ashen/model"
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

func TestBackupHandlersRejectMissingSchemaBeforeParsingBody(t *testing.T) {
	for name, factory := range map[string]func(*svc.ServiceContext) http.HandlerFunc{
		"export":  ExportHandler,
		"restore": RestoreHandler,
	} {
		t.Run(name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("create sqlmock: %v", err)
			}
			defer db.Close()
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name IN (?, ?, ?)`)).
				WithArgs("media_assets", "traffic_content_daily_stats", "traffic_content_daily_visitors").
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))

			req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/backups/"+name, nil)
			req = req.WithContext(authutil.WithUser(req.Context(), 1, authutil.RoleAdmin))
			req.Header.Set("Content-Type", "multipart/form-data; boundary=test")
			req.Body = panicReadCloser{}
			recorder := httptest.NewRecorder()

			factory(&svc.ServiceContext{Store: model.NewStore(db)})(recorder, req)

			if recorder.Code != http.StatusServiceUnavailable {
				t.Fatalf("status = %d, want %d; body = %s", recorder.Code, http.StatusServiceUnavailable, recorder.Body.String())
			}
			var body struct {
				Code    int    `json:"code"`
				Message string `json:"message"`
			}
			if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if body.Code != 50300 || body.Message != "database schema migration is required" {
				t.Fatalf("response = %#v, want controlled schema migration error", body)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatalf("unmet SQL expectations: %v", err)
			}
		})
	}
}
