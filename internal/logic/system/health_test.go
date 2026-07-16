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
	tests := []struct {
		name        string
		count       int
		wantErr     error
		wantMessage string
	}{
		{
			name:    "schema is ready",
			count:   3,
			wantErr: nil,
		},
		{
			name:        "schema migration is required",
			count:       2,
			wantErr:     errBackupSchemaMigrationRequired,
			wantMessage: backupSchemaMigrationHealthMessage,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("create sqlmock: %v", err)
			}
			defer db.Close()
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name IN (?, ?, ?)`)).
				WithArgs("media_assets", "traffic_content_daily_stats", "traffic_content_daily_visitors").
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(tt.count))

			err = backupSchemaHealthProbe(context.Background(), &svc.ServiceContext{Store: model.NewStore(db)})
			if tt.wantErr == nil {
				if err != nil {
					t.Fatalf("backupSchemaHealthProbe() error = %v", err)
				}
			} else if !errors.Is(err, tt.wantErr) {
				t.Fatalf("backupSchemaHealthProbe() error = %v, want %v", err, tt.wantErr)
			}
			if tt.wantMessage != "" && dependencyCheckFailureMessage(err) != tt.wantMessage {
				t.Fatalf("dependencyCheckFailureMessage() = %q, want %q", dependencyCheckFailureMessage(err), tt.wantMessage)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatalf("unmet SQL expectations: %v", err)
			}
		})
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
