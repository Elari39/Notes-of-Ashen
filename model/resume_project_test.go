package model

import (
	"context"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestReplaceProjectItemsUsesEachInsertID(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM project_tags")).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM projects")).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("INSERT INTO projects").WillReturnResult(sqlmock.NewResult(10, 1))
	mock.ExpectExec("INSERT INTO projects").WillReturnResult(sqlmock.NewResult(20, 1))
	mock.ExpectExec("INSERT INTO project_tags").
		WithArgs(uint64(10), uint64(1), uint64(20), uint64(2)).
		WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectCommit()
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	items := []ProjectItem{{Title: "A", TagIDs: []uint64{1}}, {Title: "B", TagIDs: []uint64{2}}}
	if err := replaceProjectItemsTx(context.Background(), tx, items); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
