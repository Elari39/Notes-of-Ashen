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

func TestBackupSettingsExcludesRestoreMarker(t *testing.T) {
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
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT setting_key, setting_value FROM site_settings WHERE setting_key NOT IN (?, ?) ORDER BY setting_key`)).
		WithArgs("ai_api_key_cipher", BackupRestoreMarkerKey).
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
