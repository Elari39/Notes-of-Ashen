package site

import (
	"context"
	"net/http"
	"regexp"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"

	embeddedmigrations "notes-of-ashen/deploy/mysql/migrations"
	"notes-of-ashen/internal/migration"
	"notes-of-ashen/internal/svc"
	"notes-of-ashen/model"
)

const runtimeSchemaColumnsQueryPrefixForTest = `SELECT table_name, column_name FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name IN (`
const schemaMigrationStateQueryForTest = `SELECT version, name, checksum FROM schema_migrations ORDER BY version`

func TestHealthReportsMissingSchemaAsReadinessFailure(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	if err != nil {
		t.Fatalf("create sqlmock: %v", err)
	}
	defer db.Close()

	mock.ExpectPing()
	mock.ExpectQuery(regexp.QuoteMeta(schemaMigrationStateQueryForTest)).
		WillReturnRows(appliedEmbeddedMigrationRows(t))
	mock.ExpectQuery(regexp.QuoteMeta(runtimeSchemaColumnsQueryPrefixForTest)).
		WillReturnRows(sqlmock.NewRows([]string{"table_name", "column_name"}))

	report := Health(context.Background(), &svc.ServiceContext{Store: model.NewStore(db)})
	schema := report.Checks["schema"]
	if schema.Status != "down" {
		t.Fatalf("schema status = %q, want down", schema.Status)
	}
	if schema.Error != schemaMigrationRequiredMessage {
		t.Fatalf("schema error = %q, want %q", schema.Error, schemaMigrationRequiredMessage)
	}
	if report.Status != "degraded" {
		t.Fatalf("report status = %q, want degraded", report.Status)
	}
	if got := HTTPStatus(report); got != http.StatusServiceUnavailable {
		t.Fatalf("HTTPStatus(report) = %d, want %d", got, http.StatusServiceUnavailable)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}

func TestHealthReportsMissingMigrationVersionAsReadinessFailure(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	if err != nil {
		t.Fatalf("create sqlmock: %v", err)
	}
	defer db.Close()

	mock.ExpectPing()
	mock.ExpectQuery(regexp.QuoteMeta(schemaMigrationStateQueryForTest)).
		WillReturnRows(sqlmock.NewRows([]string{"version", "name", "checksum"}))

	report := Health(context.Background(), &svc.ServiceContext{Store: model.NewStore(db)})
	schema := report.Checks["schema"]
	if schema.Status != "down" {
		t.Fatalf("schema status = %q, want down", schema.Status)
	}
	if !strings.Contains(schema.Error, "missing migration versions") || !strings.Contains(schema.Error, "000001") {
		t.Fatalf("schema error = %q, want concrete missing migration version", schema.Error)
	}
	if report.Status != "degraded" {
		t.Fatalf("report status = %q, want degraded", report.Status)
	}
	if got := HTTPStatus(report); got != http.StatusServiceUnavailable {
		t.Fatalf("HTTPStatus(report) = %d, want %d", got, http.StatusServiceUnavailable)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}

func appliedEmbeddedMigrationRows(t *testing.T) *sqlmock.Rows {
	t.Helper()
	items, err := migration.Discover(embeddedmigrations.FS)
	if err != nil {
		t.Fatalf("discover embedded migrations: %v", err)
	}
	rows := sqlmock.NewRows([]string{"version", "name", "checksum"})
	for _, item := range items {
		rows.AddRow(item.Version, item.Name, item.Checksum)
	}
	return rows
}
