package system

import (
	"context"
	"errors"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"

	"notes-of-ashen/internal/svc"
	"notes-of-ashen/model"
)

func TestBackupSchemaHealthProbe(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock: %v", err)
	}
	defer db.Close()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT table_name, column_name FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name IN (`)).
		WillReturnRows(sqlmock.NewRows([]string{"table_name", "column_name"}))

	err = backupSchemaHealthProbe(context.Background(), &svc.ServiceContext{Store: model.NewStore(db)})
	if !errors.Is(err, errBackupSchemaMigrationRequired) {
		t.Fatalf("backupSchemaHealthProbe() error = %v, want %v", err, errBackupSchemaMigrationRequired)
	}
	if got := dependencyCheckFailureMessage(err); got != backupSchemaMigrationHealthMessage {
		t.Fatalf("dependencyCheckFailureMessage() = %q, want %q", got, backupSchemaMigrationHealthMessage)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}

func TestBackupSchemaMigrationHealthMessageMentionsRAG(t *testing.T) {
	if got := dependencyCheckFailureMessage(errBackupSchemaMigrationRequired); got != "数据库结构未升级，请执行媒体、内容分析与 RAG 迁移后重试" {
		t.Fatalf("dependencyCheckFailureMessage() = %q, want RAG migration guidance", got)
	}
}

func TestBackupSchemaHealthProbeWithoutStore(t *testing.T) {
	err := backupSchemaHealthProbe(context.Background(), nil)
	if !errors.Is(err, errBackupSchemaCheckUnavailable) {
		t.Fatalf("backupSchemaHealthProbe() error = %v, want unavailable error", err)
	}
	if got := dependencyCheckFailureMessage(err); got != "探测失败" {
		t.Fatalf("dependencyCheckFailureMessage() = %q, want safe fallback", got)
	}
}
