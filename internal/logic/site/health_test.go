package site

import (
	"context"
	"net/http"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"

	"notes-of-ashen/internal/svc"
	"notes-of-ashen/model"
)

const runtimeSchemaColumnsQueryPrefixForTest = `SELECT table_name, column_name FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name IN (`

func TestHealthReportsMissingSchemaAsReadinessFailure(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	if err != nil {
		t.Fatalf("create sqlmock: %v", err)
	}
	defer db.Close()

	mock.ExpectPing()
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
