package migration

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-sql-driver/mysql"
)

func TestDiscoverSortsAndChecksumsMigrations(t *testing.T) {
	source := fstest.MapFS{
		"000002_add_index.sql":      {Data: []byte("CREATE INDEX idx_example ON example (id);\n")},
		"000001_create_example.sql": {Data: []byte("CREATE TABLE example (id BIGINT PRIMARY KEY);\n")},
		"README.md":                 {Data: []byte("not a migration")},
	}

	migrations, err := Discover(source)
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}
	if len(migrations) != 2 {
		t.Fatalf("Discover() returned %d migrations, want 2", len(migrations))
	}
	if migrations[0].Version != 1 || migrations[0].Name != "000001_create_example.sql" {
		t.Fatalf("first migration = %#v, want version 1 create_example", migrations[0])
	}
	if migrations[1].Version != 2 || migrations[1].Name != "000002_add_index.sql" {
		t.Fatalf("second migration = %#v, want version 2 add_index", migrations[1])
	}
	if len(migrations[0].Checksum) != 64 || migrations[0].Checksum == migrations[1].Checksum {
		t.Fatalf("unexpected checksums: %q, %q", migrations[0].Checksum, migrations[1].Checksum)
	}
}

func TestLatestVersionReturnsHighestEmbeddedMigration(t *testing.T) {
	version, err := LatestVersion(singleMigrationFS())
	if err != nil {
		t.Fatalf("LatestVersion() error = %v", err)
	}
	if version != 1 {
		t.Fatalf("LatestVersion() = %d, want 1", version)
	}
}

func TestCurrentVersionReadsDatabaseMaximum(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock: %v", err)
	}
	defer db.Close()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT COALESCE(MAX(version), 0) FROM schema_migrations`)).
		WillReturnRows(sqlmock.NewRows([]string{"version"}).AddRow(uint64(24)))

	version, err := CurrentVersion(context.Background(), db)
	if err != nil {
		t.Fatalf("CurrentVersion() error = %v", err)
	}
	if version != 24 {
		t.Fatalf("CurrentVersion() = %d, want 24", version)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}

func TestValidateRollbackCompatibility(t *testing.T) {
	if err := ValidateRollbackCompatibility(24, 24, false); err != nil {
		t.Fatalf("equal versions rejected: %v", err)
	}
	if err := ValidateRollbackCompatibility(23, 24, false); err != nil {
		t.Fatalf("target ahead of database rejected: %v", err)
	}
	err := ValidateRollbackCompatibility(24, 23, false)
	if !errors.Is(err, ErrRollbackSchemaIncompatible) {
		t.Fatalf("incompatible versions error = %v, want ErrRollbackSchemaIncompatible", err)
	}
	if err := ValidateRollbackCompatibility(24, 23, true); err != nil {
		t.Fatalf("explicit override rejected: %v", err)
	}
}

func TestDiscoverRejectsNumberGapsAndInvalidNames(t *testing.T) {
	tests := []struct {
		name   string
		source fstest.MapFS
		want   string
	}{
		{
			name: "gap",
			source: fstest.MapFS{
				"000001_create.sql": {Data: []byte("SELECT 1;")},
				"000003_skip.sql":   {Data: []byte("SELECT 1;")},
			},
			want: "contiguous",
		},
		{
			name: "invalid name",
			source: fstest.MapFS{
				"initial.sql": {Data: []byte("SELECT 1;")},
			},
			want: "invalid migration filename",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Discover(tt.source)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("Discover() error = %v, want containing %q", err, tt.want)
			}
		})
	}
}

func TestCheckReportsMissingVersionAndChecksumDrift(t *testing.T) {
	source := fstest.MapFS{
		"000001_create_example.sql": {Data: []byte("CREATE TABLE example (id BIGINT PRIMARY KEY);\n")},
		"000002_add_index.sql":      {Data: []byte("CREATE INDEX idx_example ON example (id);\n")},
	}
	migrations, err := Discover(source)
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock: %v", err)
	}
	defer db.Close()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT version, name, checksum FROM schema_migrations ORDER BY version`)).
		WillReturnRows(sqlmock.NewRows([]string{"version", "name", "checksum"}).
			AddRow(migrations[0].Version, migrations[0].Name, "changed-checksum"))

	err = Check(context.Background(), db, source)
	var stateErr *StateError
	if !errors.As(err, &stateErr) {
		t.Fatalf("Check() error = %v, want *StateError", err)
	}
	if got, want := stateErr.Missing, []uint64{2}; len(got) != len(want) || got[0] != want[0] {
		t.Fatalf("missing versions = %v, want %v", got, want)
	}
	if len(stateErr.ChecksumDrifts) != 1 || stateErr.ChecksumDrifts[0].Version != 1 {
		t.Fatalf("checksum drifts = %#v, want version 1", stateErr.ChecksumDrifts)
	}
	if !strings.Contains(err.Error(), "000002") || !strings.Contains(err.Error(), "checksum drift") {
		t.Fatalf("Check() error = %q, want missing version and checksum drift", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}

func TestCheckMetadataMissingDoesNotCreateTables(t *testing.T) {
	source := singleMigrationFS()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock: %v", err)
	}
	defer db.Close()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT version, name, checksum FROM schema_migrations ORDER BY version`)).
		WillReturnError(&mysql.MySQLError{Number: 1146, Message: "Table schema_migrations doesn't exist"})

	err = Check(context.Background(), db, source)
	var stateErr *StateError
	if !errors.As(err, &stateErr) || !stateErr.MetadataMissing {
		t.Fatalf("Check() error = %v, want metadata-missing StateError", err)
	}
	if !strings.Contains(err.Error(), "migrate service") {
		t.Fatalf("Check() error = %q, want actionable migration message", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("Check() issued an unexpected write: %v", err)
	}
}

func TestRunExecutesWholeScriptAndRecordsSuccess(t *testing.T) {
	source := singleMigrationFS()
	migrations, err := Discover(source)
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}
	item := migrations[0]

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock: %v", err)
	}
	defer db.Close()
	expectLockAcquired(mock)
	mock.ExpectExec(regexp.QuoteMeta(createMigrationsTable)).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(regexp.QuoteMeta(createMigrationRunsTable)).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT version, name, checksum FROM schema_migrations ORDER BY version`)).
		WillReturnRows(sqlmock.NewRows([]string{"version", "name", "checksum"}))
	mock.ExpectExec(regexp.QuoteMeta(item.sql)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO schema_migrations (version, name, checksum, execution_ms, applied_at) VALUES (?, ?, ?, ?, ?)`)).
		WithArgs(item.Version, item.Name, item.Checksum, sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO schema_migration_runs (version, name, checksum, status, started_at, finished_at, execution_ms, error_message) VALUES (?, ?, ?, ?, ?, ?, ?, NULL)`)).
		WithArgs(item.Version, item.Name, item.Checksum, "success", sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	expectLockReleased(mock)

	if err := Run(context.Background(), db, source); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}

func TestRunDoesNotMarkFailedMigrationAsApplied(t *testing.T) {
	source := singleMigrationFS()
	migrations, err := Discover(source)
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}
	item := migrations[0]

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock: %v", err)
	}
	defer db.Close()
	expectLockAcquired(mock)
	mock.ExpectExec(regexp.QuoteMeta(createMigrationsTable)).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(regexp.QuoteMeta(createMigrationRunsTable)).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT version, name, checksum FROM schema_migrations ORDER BY version`)).
		WillReturnRows(sqlmock.NewRows([]string{"version", "name", "checksum"}))
	executeErr := errors.New("syntax error near example")
	mock.ExpectExec(regexp.QuoteMeta(item.sql)).WillReturnError(executeErr)
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO schema_migration_runs (version, name, checksum, status, started_at, finished_at, execution_ms, error_message) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`)).
		WithArgs(item.Version, item.Name, item.Checksum, "failed", sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), executeErr.Error()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	expectLockReleased(mock)

	err = Run(context.Background(), db, source)
	if !errors.Is(err, executeErr) {
		t.Fatalf("Run() error = %v, want execution error", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}

func TestRunStopsWhenMigrationLockIsHeld(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock: %v", err)
	}
	defer db.Close()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT GET_LOCK(?, ?)`)).
		WithArgs(LockName, int(lockWaitTimeout.Seconds())).
		WillReturnRows(sqlmock.NewRows([]string{"GET_LOCK"}).AddRow(0))

	err = Run(context.Background(), db, singleMigrationFS())
	if !errors.Is(err, ErrLockNotAcquired) {
		t.Fatalf("Run() error = %v, want ErrLockNotAcquired", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}

func singleMigrationFS() fstest.MapFS {
	return fstest.MapFS{
		"000001_create_example.sql": {Data: []byte("CREATE TABLE example (id BIGINT PRIMARY KEY);\nCREATE INDEX idx_example ON example (id);\n")},
	}
}

func expectLockAcquired(mock sqlmock.Sqlmock) {
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT GET_LOCK(?, ?)`)).
		WithArgs(LockName, int(lockWaitTimeout.Seconds())).
		WillReturnRows(sqlmock.NewRows([]string{"GET_LOCK"}).AddRow(1))
}

func expectLockReleased(mock sqlmock.Sqlmock) {
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT RELEASE_LOCK(?)`)).
		WithArgs(LockName).
		WillReturnRows(sqlmock.NewRows([]string{"RELEASE_LOCK"}).AddRow(1))
}
