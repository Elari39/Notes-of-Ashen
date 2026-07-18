package model

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestCreateMarkdownArticleRollsBackTaxonomyWhenArticleInsertFails(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	store := NewStore(db)
	insertErr := errors.New("article insert failed")
	mock.ExpectBegin()
	mock.ExpectQuery("FROM categories WHERE name").WillReturnError(sql.ErrNoRows)
	mock.ExpectExec("INSERT INTO categories").WillReturnResult(sqlmock.NewResult(5, 1))
	mock.ExpectExec("INSERT INTO articles").WillReturnError(insertErr)
	mock.ExpectRollback()
	_, err = store.CreateMarkdownArticle(context.Background(), MarkdownArticleImport{
		Article:  ArticleCreate{AuthorID: 1, Title: "A", Slug: "a", Content: "body", Status: ArticleStatusDraft},
		Category: &TaxonomyCreate{Name: "Go", Slug: "go", CreatedBy: 1},
	})
	if !errors.Is(err, insertErr) {
		t.Fatalf("error=%v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
