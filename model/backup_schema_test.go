package model

import (
	"context"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestBackupSchemaReady(t *testing.T) {
	tests := []struct {
		name  string
		count int
		want  bool
	}{
		{name: "all required tables exist", count: len(backupSchemaRequiredTables), want: true},
		{name: "a required table is missing", count: len(backupSchemaRequiredTables) - 1, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("create sqlmock: %v", err)
			}
			defer db.Close()

			mock.ExpectQuery(regexp.QuoteMeta(backupSchemaReadyQuery)).
				WithArgs(backupSchemaRequiredTables[0], backupSchemaRequiredTables[1], backupSchemaRequiredTables[2]).
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(tt.count))

			ready, err := NewStore(db).BackupSchemaReady(context.Background())
			if err != nil {
				t.Fatalf("BackupSchemaReady() error = %v", err)
			}
			if ready != tt.want {
				t.Fatalf("BackupSchemaReady() = %v, want %v", ready, tt.want)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatalf("unmet SQL expectations: %v", err)
			}
		})
	}
}

func TestBackupSchemaReadyWithoutDatabase(t *testing.T) {
	ready, err := (*Store)(nil).BackupSchemaReady(context.Background())
	if err == nil {
		t.Fatal("BackupSchemaReady() error = nil, want error")
	}
	if ready {
		t.Fatal("BackupSchemaReady() = true, want false")
	}
}
