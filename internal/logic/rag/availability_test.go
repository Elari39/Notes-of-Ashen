package rag

import (
	"context"
	"encoding/json"
	stdErrors "errors"
	"net/http"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	apperrors "notes-of-ashen/internal/errors"
	ragcore "notes-of-ashen/internal/rag"
	"notes-of-ashen/internal/svc"
	"notes-of-ashen/model"
)

func TestMapUpstreamErrorMapsTimeoutToGatewayTimeout(t *testing.T) {
	err := mapUpstreamError(context.DeadlineExceeded)
	var codeErr *apperrors.CodeError
	if !stdErrors.As(err, &codeErr) || codeErr.StatusCode != http.StatusGatewayTimeout {
		t.Fatalf("mapUpstreamError(deadline) = %#v, want 504", err)
	}

	err = mapUpstreamError(stdErrors.New("private upstream message"))
	if !stdErrors.As(err, &codeErr) || codeErr.StatusCode != http.StatusBadGateway || codeErr.Message != "rag provider request failed" {
		t.Fatalf("mapUpstreamError(other) = %#v, want safe 502", err)
	}
}

func TestMapVectorIndexErrorFailsClosedAsServiceUnavailable(t *testing.T) {
	err := mapVectorIndexError(stdErrors.New("qdrant refused request"))
	var codeErr *apperrors.CodeError
	if !stdErrors.As(err, &codeErr) || codeErr.StatusCode != http.StatusServiceUnavailable || codeErr.Message != "rag vector index is unavailable" {
		t.Fatalf("mapVectorIndexError() = %#v, want safe 503", err)
	}
}

func TestMessagesRespDoesNotExposePersistedHiddenAssistantContent(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	defer db.Close()

	hiddenAt := time.Date(2026, time.August, 13, 0, 0, 0, 0, time.UTC)
	items := []model.RAGChatMessage{{
		ID:        7,
		SessionID: "session-1",
		Role:      "assistant",
		Content:   "已下线文章中的旧答案",
		Sources:   []byte(`[{"articleId":42,"snippet":"旧片段"}]`),
		HiddenAt:  &hiddenAt,
		CreatedAt: hiddenAt,
	}}

	got, err := messagesResp(context.Background(), &svc.ServiceContext{Store: model.NewStore(db)}, items)
	if err != nil {
		t.Fatalf("messagesResp() error = %v", err)
	}
	if len(got) != 1 || got[0].HiddenAt == nil {
		t.Fatalf("messagesResp() = %#v, want one hidden message", got)
	}
	if got[0].Content != "" || len(got[0].Sources) != 0 {
		t.Fatalf("hidden assistant response leaks content or sources: %#v", got[0])
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestSafePromptHistoryOmitsAssistantAnswerWhoseSourceIsNoLongerPublic(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	defer db.Close()

	sources, err := json.Marshal([]Source{{ArticleID: 42, SourceHash: "hash", Title: "已下线文章"}})
	if err != nil {
		t.Fatal(err)
	}
	items := []model.RAGChatMessage{
		{ID: 1, SessionID: "session-1", Role: "user", Content: "旧问题"},
		{ID: 2, SessionID: "session-1", Role: "assistant", Content: "不应再传给上游的旧回答", Sources: sources},
		{ID: 3, SessionID: "session-1", Role: "user", Content: "后续问题"},
	}

	// 文章仍存在但已转为草稿；异步 outbox 尚未写 HiddenAt 时，也必须立即从
	// 下一轮模型上下文中排除相关 assistant 内容。
	articleRows := sqlmock.NewRows([]string{
		"id", "author_id", "category_id", "title", "slug", "summary", "content", "cover_url", "status", "view_count", "like_count", "scheduled_at", "published_at", "is_pinned", "display_priority", "seo_title", "seo_description", "seo_keywords", "created_at", "updated_at",
	}).AddRow(42, 1, 0, "已下线文章", "offline", "", "内容", "", model.ArticleStatusDraft, 0, 0, nil, nil, 0, 0, "", "", "", time.Now(), time.Now())
	mock.ExpectQuery("SELECT id, author_id, category_id, title, slug, summary, content, cover_url, status, view_count, like_count, scheduled_at, published_at, is_pinned, display_priority, seo_title, seo_description, seo_keywords, created_at, updated_at FROM articles WHERE id IN").
		WithArgs(uint64(42)).WillReturnRows(articleRows)

	history, err := safePromptHistory(context.Background(), &svc.ServiceContext{Store: model.NewStore(db)}, items)
	if err != nil {
		t.Fatalf("safePromptHistory() error = %v", err)
	}
	if len(history) != 2 || history[0].ID != 1 || history[1].ID != 3 {
		t.Fatalf("safePromptHistory() = %#v, want only user messages", history)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestSafePromptHistoryKeepsAssistantAnswerWhenSourcesRemainPublic(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	defer db.Close()

	article := model.Article{ID: 42, Title: "公开文章", Summary: "", Content: "内容", Status: model.ArticleStatusPublished}
	sources, err := json.Marshal([]Source{{ArticleID: article.ID, SourceHash: ragcore.ContentHash(article.Title, article.Summary, article.Content)}})
	if err != nil {
		t.Fatal(err)
	}
	items := []model.RAGChatMessage{{ID: 2, SessionID: "session-1", Role: "assistant", Content: "可继续引用的回答", Sources: sources}}
	articleRows := sqlmock.NewRows([]string{
		"id", "author_id", "category_id", "title", "slug", "summary", "content", "cover_url", "status", "view_count", "like_count", "scheduled_at", "published_at", "is_pinned", "display_priority", "seo_title", "seo_description", "seo_keywords", "created_at", "updated_at",
	}).AddRow(article.ID, 1, 0, article.Title, "public", article.Summary, article.Content, "", article.Status, 0, 0, nil, nil, 0, 0, "", "", "", time.Now(), time.Now())
	mock.ExpectQuery("SELECT id, author_id, category_id, title, slug, summary, content, cover_url, status, view_count, like_count, scheduled_at, published_at, is_pinned, display_priority, seo_title, seo_description, seo_keywords, created_at, updated_at FROM articles WHERE id IN").
		WithArgs(article.ID).WillReturnRows(articleRows)

	history, err := safePromptHistory(context.Background(), &svc.ServiceContext{Store: model.NewStore(db)}, items)
	if err != nil {
		t.Fatalf("safePromptHistory() error = %v", err)
	}
	if len(history) != 1 || history[0].Content != "可继续引用的回答" {
		t.Fatalf("safePromptHistory() = %#v, want assistant answer retained", history)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestSafePromptHistoryOmitsAssistantAnswerWhenSourceContentChanged(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	defer db.Close()

	article := model.Article{ID: 42, Title: "公开文章", Summary: "", Content: "更新后的内容", Status: model.ArticleStatusPublished}
	sources, err := json.Marshal([]Source{{ArticleID: article.ID, SourceHash: "old-content-hash"}})
	if err != nil {
		t.Fatal(err)
	}
	items := []model.RAGChatMessage{{ID: 2, SessionID: "session-1", Role: "assistant", Content: "基于旧版本的回答", Sources: sources}}
	articleRows := sqlmock.NewRows([]string{
		"id", "author_id", "category_id", "title", "slug", "summary", "content", "cover_url", "status", "view_count", "like_count", "scheduled_at", "published_at", "is_pinned", "display_priority", "seo_title", "seo_description", "seo_keywords", "created_at", "updated_at",
	}).AddRow(article.ID, 1, 0, article.Title, "public", article.Summary, article.Content, "", article.Status, 0, 0, nil, nil, 0, 0, "", "", "", time.Now(), time.Now())
	mock.ExpectQuery("SELECT id, author_id, category_id, title, slug, summary, content, cover_url, status, view_count, like_count, scheduled_at, published_at, is_pinned, display_priority, seo_title, seo_description, seo_keywords, created_at, updated_at FROM articles WHERE id IN").
		WithArgs(article.ID).WillReturnRows(articleRows)

	history, err := safePromptHistory(context.Background(), &svc.ServiceContext{Store: model.NewStore(db)}, items)
	if err != nil {
		t.Fatalf("safePromptHistory() error = %v", err)
	}
	if len(history) != 0 {
		t.Fatalf("safePromptHistory() = %#v, want stale assistant answer omitted", history)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestSafePromptHistoryKeepsNoEvidenceAnswerWithoutSources(t *testing.T) {
	items := []model.RAGChatMessage{
		{ID: 1, SessionID: "session-1", Role: "user", Content: "没有资料的问题"},
		{ID: 2, SessionID: "session-1", Role: "assistant", Content: "现有文章中没有足够依据。", Sources: []byte("[]")},
	}

	history, err := safePromptHistory(context.Background(), &svc.ServiceContext{}, items)
	if err != nil {
		t.Fatalf("safePromptHistory() error = %v", err)
	}
	if len(history) != 2 || history[1].Content != "现有文章中没有足够依据。" {
		t.Fatalf("safePromptHistory() = %#v, want no-evidence answer retained", history)
	}
}

// 会话历史是 /ask 页面的一部分。页面关闭时，即使请求未携带登录信息，也必须先
// 返回 404；否则列表/详情/删除接口会泄露已启用过问答功能这一事实。
func TestSessionEndpointsReturnNotFoundWhenChatPageIsDisabled(t *testing.T) {
	tests := []struct {
		name string
		call func(context.Context, *svc.ServiceContext) error
	}{
		{
			name: "list",
			call: func(ctx context.Context, svcCtx *svc.ServiceContext) error {
				_, err := ListSessions(ctx, svcCtx)
				return err
			},
		},
		{
			name: "detail",
			call: func(ctx context.Context, svcCtx *svc.ServiceContext) error {
				_, err := GetSession(ctx, svcCtx, "session-id")
				return err
			},
		},
		{
			name: "delete",
			call: func(ctx context.Context, svcCtx *svc.ServiceContext) error {
				return DeleteSession(ctx, svcCtx, "session-id")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("sqlmock.New() error = %v", err)
			}
			defer db.Close()

			mock.ExpectQuery("SELECT setting_key, setting_value FROM site_settings WHERE setting_key IN").
				WillReturnRows(sqlmock.NewRows([]string{"setting_key", "setting_value"}).AddRow(model.RAGChatPageEnabledKey, "false"))

			err = tt.call(context.Background(), &svc.ServiceContext{Store: model.NewStore(db)})
			assertRAGChatNotFound(t, err)
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatalf("sqlmock expectations: %v", err)
			}
		})
	}
}

// 404 仅代表页面关闭；页面已开启而引擎尚未配置时，会话接口应与流式接口一样
// fail-closed 返回 503，不能把运维状态伪装成路由不存在。
func TestSessionEndpointsReturnServiceUnavailableWhenRAGIsDisabled(t *testing.T) {
	tests := []struct {
		name string
		call func(context.Context, *svc.ServiceContext) error
	}{
		{
			name: "list",
			call: func(ctx context.Context, svcCtx *svc.ServiceContext) error {
				_, err := ListSessions(ctx, svcCtx)
				return err
			},
		},
		{
			name: "detail",
			call: func(ctx context.Context, svcCtx *svc.ServiceContext) error {
				_, err := GetSession(ctx, svcCtx, "session-id")
				return err
			},
		},
		{
			name: "delete",
			call: func(ctx context.Context, svcCtx *svc.ServiceContext) error {
				return DeleteSession(ctx, svcCtx, "session-id")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("sqlmock.New() error = %v", err)
			}
			defer db.Close()

			mock.ExpectQuery("SELECT setting_key, setting_value FROM site_settings WHERE setting_key IN").
				WillReturnRows(sqlmock.NewRows([]string{"setting_key", "setting_value"}).AddRow(model.RAGChatPageEnabledKey, "true"))
			mock.ExpectQuery("SELECT setting_key, setting_value FROM site_settings WHERE setting_key IN").
				WillReturnRows(sqlmock.NewRows([]string{"setting_key", "setting_value"}).AddRow(model.RAGEnabledKey, "false"))

			err = tt.call(context.Background(), &svc.ServiceContext{Store: model.NewStore(db)})
			assertRAGChatServiceUnavailable(t, err)
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatalf("sqlmock expectations: %v", err)
			}
		})
	}
}

func assertRAGChatNotFound(t *testing.T, err error) {
	t.Helper()
	var codeErr *apperrors.CodeError
	if !stdErrors.As(err, &codeErr) || codeErr.StatusCode != http.StatusNotFound {
		t.Fatalf("error = %#v, want RAG chat 404", err)
	}
}

func assertRAGChatServiceUnavailable(t *testing.T, err error) {
	t.Helper()
	var codeErr *apperrors.CodeError
	if !stdErrors.As(err, &codeErr) || codeErr.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("error = %#v, want RAG chat 503", err)
	}
}
