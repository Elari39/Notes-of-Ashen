package backup

import (
	"context"
	stderrors "errors"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"

	apperrors "notes-of-ashen/internal/errors"
	"notes-of-ashen/internal/svc"
	"notes-of-ashen/model"
)

func TestEnsureSchemaReady(t *testing.T) {
	tests := []struct {
		name        string
		configure   func(sqlmock.Sqlmock)
		wantMessage string
	}{
		{
			name: "migration is required",
			configure: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name IN (?, ?, ?)`)).
					WithArgs("media_assets", "traffic_content_daily_stats", "traffic_content_daily_visitors").
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))
			},
			wantMessage: backupSchemaMigrationRequiredMessage,
		},
		{
			name: "schema check is unavailable",
			configure: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name IN (?, ?, ?)`)).
					WithArgs("media_assets", "traffic_content_daily_stats", "traffic_content_daily_visitors").
					WillReturnError(stderrors.New("database query failed"))
			},
			wantMessage: backupSchemaCheckUnavailableMessage,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("create sqlmock: %v", err)
			}
			defer db.Close()
			tt.configure(mock)

			err = EnsureSchemaReady(context.Background(), &svc.ServiceContext{Store: model.NewStore(db)})
			var codeErr *apperrors.CodeError
			if !stderrors.As(err, &codeErr) {
				t.Fatalf("EnsureSchemaReady() error = %v, want CodeError", err)
			}
			if codeErr.Code != 50300 || codeErr.Message != tt.wantMessage {
				t.Fatalf("EnsureSchemaReady() error = %#v, want controlled service unavailable error", codeErr)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatalf("unmet SQL expectations: %v", err)
			}
		})
	}
}
