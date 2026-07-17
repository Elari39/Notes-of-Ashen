package search

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"notes-of-ashen/internal/authutil"
	"notes-of-ashen/internal/config"
	apperrors "notes-of-ashen/internal/errors"
	searchclient "notes-of-ashen/internal/search"
	"notes-of-ashen/internal/svc"
	"notes-of-ashen/model"
)

func TestReindexPermissionAndDisabledSearch(t *testing.T) {
	tests := []struct {
		name        string
		role        string
		wantDenied  bool
		wantEnabled bool
	}{
		{name: "user is denied", role: authutil.RoleUser, wantDenied: true},
		{name: "editor accepts disabled search", role: authutil.RoleEditor},
		{name: "admin accepts disabled search", role: authutil.RoleAdmin},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := authutil.WithUser(context.Background(), 1, tt.role)
			resp, err := Reindex(ctx, &svc.ServiceContext{})
			if tt.wantDenied {
				assertForbidden(t, err)
				return
			}
			if err != nil {
				t.Fatalf("Reindex() error = %v", err)
			}
			if resp.Enabled != tt.wantEnabled || resp.Indexed != 0 {
				t.Fatalf("Reindex() = %#v, want disabled response", resp)
			}
		})
	}
}

func TestSuggestionsRejectsInvalidQuery(t *testing.T) {
	tests := []string{
		"x",
		strings.Repeat("界", 81),
	}

	for _, query := range tests {
		t.Run("length "+strconv.Itoa(len([]rune(query))), func(t *testing.T) {
			_, err := Suggestions(context.Background(), &svc.ServiceContext{}, query, 8)
			var codeErr *apperrors.CodeError
			if !errors.As(err, &codeErr) || codeErr.StatusCode != http.StatusBadRequest {
				t.Fatalf("Suggestions() error = %T %[1]v, want bad request", err)
			}
		})
	}
}

func TestSuggestionsNormalizesLimit(t *testing.T) {
	tests := []struct {
		name             string
		limit            int
		wantCategorySize int
		wantTagSize      int
	}{
		{name: "defaults non-positive limit", limit: 0, wantCategorySize: 3, wantTagSize: 8},
		{name: "caps oversized limit", limit: 99, wantCategorySize: 3, wantTagSize: 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock := newSQLMock(t)
			defer db.Close()

			expectArticleList(mock, "go", 4, emptyArticleRows())
			mock.ExpectQuery("SELECT c\\.id").
				WithArgs("%go%", "go%", tt.wantCategorySize).
				WillReturnRows(sqlmock.NewRows([]string{"id"}))
			mock.ExpectQuery("SELECT t\\.id").
				WithArgs("%go%", "go%", tt.wantTagSize).
				WillReturnRows(sqlmock.NewRows([]string{"id"}))

			resp, err := Suggestions(context.Background(), &svc.ServiceContext{Store: model.NewStore(db)}, " go ", tt.limit)
			if err != nil {
				t.Fatalf("Suggestions() error = %v", err)
			}
			if len(resp.Items) != 0 {
				t.Fatalf("Suggestions() items = %#v, want empty", resp.Items)
			}
			assertSQLMock(t, mock)
		})
	}
}

func TestSuggestionArticlesFallsBackToMySQL(t *testing.T) {
	tests := []struct {
		name     string
		status   int
		response string
	}{
		{name: "meilisearch failure", status: http.StatusInternalServerError, response: `{"message":"unavailable"}`},
		{name: "meilisearch empty hits", status: http.StatusOK, response: `{"hits":[],"estimatedTotalHits":0}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.response))
			}))
			defer server.Close()

			db, mock := newSQLMock(t)
			defer db.Close()
			expectArticleList(mock, "go", 2, onePublishedArticleRows())

			client := searchclient.NewClient(config.SearchConf{
				Enabled:          true,
				MeilisearchHost:  server.URL,
				MeilisearchIndex: "articles",
			})
			items, err := suggestionArticles(context.Background(), &svc.ServiceContext{
				Store:  model.NewStore(db),
				Search: client,
			}, "go", 2)
			if err != nil {
				t.Fatalf("suggestionArticles() error = %v", err)
			}
			if len(items) != 1 || items[0].ID != 7 || items[0].Title != "Go article" {
				t.Fatalf("suggestionArticles() = %#v, want MySQL fallback article", items)
			}
			assertSQLMock(t, mock)
		})
	}
}

func assertForbidden(t *testing.T, err error) {
	t.Helper()
	var codeErr *apperrors.CodeError
	if !errors.As(err, &codeErr) {
		t.Fatalf("error = %T %[1]v, want CodeError", err)
	}
	if codeErr.Code != 40300 || codeErr.StatusCode != http.StatusForbidden {
		t.Fatalf("error = code %d status %d, want code 40300 status %d", codeErr.Code, codeErr.StatusCode, http.StatusForbidden)
	}
}

func newSQLMock(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	return db, mock
}

func assertSQLMock(t *testing.T, mock sqlmock.Sqlmock) {
	t.Helper()
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sqlmock expectations: %v", err)
	}
}

func expectArticleList(mock sqlmock.Sqlmock, query string, limit int, rows *sqlmock.Rows) {
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM articles").
		WithArgs(query).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM articles").
		WithArgs("%" + query + "%").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery("SELECT id, author_id").
		WithArgs("%"+query+"%", limit, 0).
		WillReturnRows(rows)
}

func emptyArticleRows() *sqlmock.Rows {
	return sqlmock.NewRows(articleColumns)
}

func onePublishedArticleRows() *sqlmock.Rows {
	now := time.Date(2026, time.July, 17, 12, 0, 0, 0, time.UTC)
	return sqlmock.NewRows(articleColumns).AddRow(
		uint64(7), uint64(1), nil, "Go article", "go-article", nil, "content", nil,
		model.ArticleStatusPublished, uint64(0), uint64(0), nil, now, 0, 0, nil, nil, nil, now, now,
	)
}

var articleColumns = []string{
	"id", "author_id", "category_id", "title", "slug", "summary", "content", "cover_url", "status", "view_count",
	"like_count", "scheduled_at", "published_at", "is_pinned", "display_priority", "seo_title", "seo_description", "seo_keywords",
	"created_at", "updated_at",
}
