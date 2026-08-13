package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"

	"notes-of-ashen/model"
)

func TestRAGChatPageMiddlewareReturnsNotFoundBeforeDownstream(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	defer db.Close()
	mock.ExpectQuery("SELECT setting_key, setting_value FROM site_settings WHERE setting_key IN").
		WillReturnRows(sqlmock.NewRows([]string{"setting_key", "setting_value"}).AddRow(model.RAGChatPageEnabledKey, "false"))

	called := false
	handler := NewRAGChatPageMiddleware(model.NewStore(db)).Handle(func(http.ResponseWriter, *http.Request) {
		called = true
	})
	recorder := httptest.NewRecorder()
	handler(recorder, httptest.NewRequest(http.MethodPost, "/api/v1/rag/chat/stream", nil))

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNotFound)
	}
	if called {
		t.Fatal("downstream handler was called while RAG chat page is disabled")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestRAGChatPageMiddlewarePassesEnabledPage(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	defer db.Close()
	mock.ExpectQuery("SELECT setting_key, setting_value FROM site_settings WHERE setting_key IN").
		WillReturnRows(sqlmock.NewRows([]string{"setting_key", "setting_value"}).AddRow(model.RAGChatPageEnabledKey, "true"))

	handler := NewRAGChatPageMiddleware(model.NewStore(db)).Handle(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	recorder := httptest.NewRecorder()
	handler(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/rag/sessions", nil))

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNoContent)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
