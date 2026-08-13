package model

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestInsertBackupRestoreMarkerUsesRestoreTransaction(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	defer db.Close()

	mock.ExpectBegin()
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatalf("BeginTx() error = %v", err)
	}
	now := time.Date(2026, time.July, 17, 0, 0, 0, 0, time.UTC)
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO site_settings(setting_key,setting_value,created_at,updated_at) VALUES(?,?,?,?)`)).
		WithArgs(BackupRestoreMarkerKey, "restore-transaction", now, now).
		WillReturnResult(sqlmock.NewResult(1, 1))
	if err := insertBackupRestoreMarker(context.Background(), tx, "restore-transaction", now); err != nil {
		t.Fatalf("insertBackupRestoreMarker() error = %v", err)
	}
	mock.ExpectCommit()
	if err := tx.Commit(); err != nil {
		t.Fatalf("Commit() error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestBackupRestoreMarkerLookupAndClear(t *testing.T) {
	t.Run("missing marker", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT setting_value FROM site_settings WHERE setting_key = ? LIMIT 1`)).
			WithArgs(BackupRestoreMarkerKey).
			WillReturnRows(sqlmock.NewRows([]string{"setting_value"}))
		marker, err := NewStore(db).BackupRestoreMarker(context.Background())
		if err != nil {
			t.Fatalf("BackupRestoreMarker() error = %v", err)
		}
		if marker != "" {
			t.Fatalf("BackupRestoreMarker() = %q, want empty", marker)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("matching marker is cleared", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()
		mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM site_settings WHERE setting_key = ? AND setting_value = ?`)).
			WithArgs(BackupRestoreMarkerKey, "restore-transaction").
			WillReturnResult(sqlmock.NewResult(0, 1))
		if err := NewStore(db).ClearBackupRestoreMarker(context.Background(), "restore-transaction"); err != nil {
			t.Fatalf("ClearBackupRestoreMarker() error = %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("marker mismatch is rejected", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()
		mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM site_settings WHERE setting_key = ? AND setting_value = ?`)).
			WithArgs(BackupRestoreMarkerKey, "restore-transaction").
			WillReturnResult(sqlmock.NewResult(0, 0))
		if err := NewStore(db).ClearBackupRestoreMarker(context.Background(), "restore-transaction"); err == nil {
			t.Fatal("ClearBackupRestoreMarker() error = nil, want marker mismatch")
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatal(err)
		}
	})
}

func TestBackupSettingsExcludesSecretsAndRestoreMarker(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mock.ExpectBegin()
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatalf("BeginTx() error = %v", err)
	}
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT setting_key, setting_value FROM site_settings WHERE setting_key NOT IN (?, ?, ?) ORDER BY setting_key`)).
		WithArgs(AIAPIKeyCipherKey, RAGAPIKeyCipherKey, BackupRestoreMarkerKey).
		WillReturnRows(sqlmock.NewRows([]string{"setting_key", "setting_value"}).AddRow("site_title", "Notes"))
	settings, err := backupSettings(context.Background(), tx)
	if err != nil {
		t.Fatalf("backupSettings() error = %v", err)
	}
	if len(settings) != 1 || settings[0].Key != "site_title" {
		t.Fatalf("backupSettings() = %#v, want exported normal setting", settings)
	}
	mock.ExpectCommit()
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestRestoredSettingsDisablesRAGAndDropsRAGKey(t *testing.T) {
	settings := restoredSettings([]BackupSetting{
		{Key: RAGEnabledKey, Value: "true"},
		{Key: RAGAPIKeyCipherKey, Value: "v3:secret"},
		{Key: RAGChatPageEnabledKey, Value: "true"},
		{Key: RAGChatNavHiddenKey, Value: "false"},
		{Key: RAGChatAccessLevelKey, Value: "editor"},
		{Key: "site_title", Value: "Notes"},
	})
	if settings[RAGEnabledKey] != "false" || settings[RAGAPIKeyCipherKey] != "" {
		t.Fatalf("restored RAG engine/key settings = %#v, want disabled with empty key", settings)
	}
	if settings[RAGChatPageEnabledKey] != "false" || settings[RAGChatNavHiddenKey] != "true" {
		t.Fatalf("restored RAG page settings = %#v, want closed and hidden", settings)
	}
	if settings[RAGChatAccessLevelKey] != "editor" {
		t.Fatalf("restored RAG access level = %q, want editor", settings[RAGChatAccessLevelKey])
	}
	if settings["site_title"] != "Notes" {
		t.Fatalf("restored ordinary setting = %q, want Notes", settings["site_title"])
	}
}

func TestBackupRAGChatHistoryExcludesExpiredSessions(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	mock.ExpectBegin()
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatalf("BeginTx() error = %v", err)
	}
	now := time.Now().UTC()
	expires := now.Add(time.Hour)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, user_id, title, source_epoch, expires_at, created_at, updated_at FROM rag_chat_sessions WHERE user_id IS NOT NULL AND (expires_at IS NULL OR expires_at > NOW(6)) ORDER BY created_at, id`)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "title", "source_epoch", "expires_at", "created_at", "updated_at"}).
			AddRow("active-session", 1, "active", 1, expires, now, now))
	sessions, err := backupRAGChatSessions(context.Background(), tx)
	if err != nil {
		t.Fatalf("backupRAGChatSessions() error = %v", err)
	}
	if len(sessions) != 1 || sessions[0].ID != "active-session" {
		t.Fatalf("backupRAGChatSessions() = %#v, want only active session", sessions)
	}

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT m.id, m.session_id, m.role, m.content, m.sources, m.hidden_at, m.created_at FROM rag_chat_messages m JOIN rag_chat_sessions s ON s.id = m.session_id WHERE s.user_id IS NOT NULL AND (s.expires_at IS NULL OR s.expires_at > NOW(6)) ORDER BY m.id`)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "session_id", "role", "content", "sources", "hidden_at", "created_at"}).
			AddRow(1, "active-session", "user", "question", nil, nil, now))
	messages, err := backupRAGChatMessages(context.Background(), tx)
	if err != nil {
		t.Fatalf("backupRAGChatMessages() error = %v", err)
	}
	if len(messages) != 1 || messages[0].SessionID != "active-session" {
		t.Fatalf("backupRAGChatMessages() = %#v, want only active session messages", messages)
	}

	mock.ExpectCommit()
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
