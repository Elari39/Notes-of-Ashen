package model

import (
	"context"
	"errors"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestSchemaReady(t *testing.T) {
	tests := []struct {
		name       string
		omitTable  string
		omitColumn string
		want       bool
	}{
		{name: "all runtime fields exist", want: true},
		{name: "a required table is missing", omitTable: "media_assets", want: false},
		{name: "a required migrated field is missing", omitTable: "articles", omitColumn: "like_count", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("create sqlmock: %v", err)
			}
			defer db.Close()

			mock.ExpectQuery(regexp.QuoteMeta(runtimeSchemaColumnsQuery())).
				WillReturnRows(runtimeSchemaRows(tt.omitTable, tt.omitColumn))

			ready, err := NewStore(db).SchemaReady(context.Background())
			if err != nil {
				t.Fatalf("SchemaReady() error = %v", err)
			}
			if ready != tt.want {
				t.Fatalf("SchemaReady() = %v, want %v", ready, tt.want)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatalf("unmet SQL expectations: %v", err)
			}
		})
	}
}

func TestSchemaReadyPropagatesMetadataQueryError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock: %v", err)
	}
	defer db.Close()

	wantErr := errors.New("metadata query failed")
	mock.ExpectQuery(regexp.QuoteMeta(runtimeSchemaColumnsQuery())).WillReturnError(wantErr)

	ready, err := NewStore(db).SchemaReady(context.Background())
	if !errors.Is(err, wantErr) {
		t.Fatalf("SchemaReady() error = %v, want %v", err, wantErr)
	}
	if ready {
		t.Fatal("SchemaReady() = true, want false")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}

func TestBackupSchemaReadyUsesRuntimeManifest(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery(regexp.QuoteMeta(runtimeSchemaColumnsQuery())).
		WillReturnRows(runtimeSchemaRows("articles", "scheduled_at"))

	ready, err := NewStore(db).BackupSchemaReady(context.Background())
	if err != nil {
		t.Fatalf("BackupSchemaReady() error = %v", err)
	}
	if ready {
		t.Fatal("BackupSchemaReady() = true, want false when a runtime field is missing")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}

func TestSchemaReadyWithoutDatabase(t *testing.T) {
	ready, err := (*Store)(nil).SchemaReady(context.Background())
	if err == nil {
		t.Fatal("SchemaReady() error = nil, want error")
	}
	if ready {
		t.Fatal("SchemaReady() = true, want false")
	}
}

func runtimeSchemaRows(omitTable, omitColumn string) *sqlmock.Rows {
	rows := sqlmock.NewRows([]string{"table_name", "column_name"})
	for _, requirement := range runtimeSchemaManifest {
		if requirement.Name == omitTable && omitColumn == "" {
			continue
		}
		for _, column := range requirement.Columns {
			if requirement.Name == omitTable && column == omitColumn {
				continue
			}
			rows.AddRow(requirement.Name, column)
		}
	}
	return rows
}
