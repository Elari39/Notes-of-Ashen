package model

import (
	"context"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestDeleteArticleRemovesVersionsBeforeArticle(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock: %v", err)
	}
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM article_tags WHERE article_id = ?")).
		WithArgs(uint64(42)).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM article_likes WHERE article_id = ?")).
		WithArgs(uint64(42)).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM article_versions WHERE article_id = ?")).
		WithArgs(uint64(42)).
		WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM articles WHERE id = ?")).
		WithArgs(uint64(42)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	if err := NewStore(db).DeleteArticle(context.Background(), 42); err != nil {
		t.Fatalf("DeleteArticle() error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}

func TestMediaURLReferencedDetectsVersionOfExistingArticle(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock: %v", err)
	}
	defer db.Close()

	const mediaURL = "/uploads/cover.png"
	like := "%" + mediaURL + "%"
	mock.ExpectQuery(regexp.QuoteMeta("SELECT EXISTS(SELECT 1 FROM articles WHERE cover_url = ? OR content LIKE ?)")).
		WithArgs(mediaURL, like).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(0))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT EXISTS(SELECT 1 FROM article_versions INNER JOIN articles ON articles.id = article_versions.article_id WHERE article_versions.cover_url = ? OR article_versions.content LIKE ?)")).
		WithArgs(mediaURL, like).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(1))

	referenced, err := NewStore(db).MediaURLReferenced(context.Background(), mediaURL)
	if err != nil {
		t.Fatalf("MediaURLReferenced() error = %v", err)
	}
	if !referenced {
		t.Fatal("MediaURLReferenced() = false, want true for a version of an existing article")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}

func TestMediaURLReferencedIgnoresOrphanedArticleVersion(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock: %v", err)
	}
	defer db.Close()

	const mediaURL = "/uploads/orphan.png"
	like := "%" + mediaURL + "%"
	mock.ExpectQuery(regexp.QuoteMeta("SELECT EXISTS(SELECT 1 FROM articles WHERE cover_url = ? OR content LIKE ?)")).
		WithArgs(mediaURL, like).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(0))
	// 该查询通过 INNER JOIN 排除 article_id 已不存在的孤儿历史版本。
	mock.ExpectQuery(regexp.QuoteMeta("SELECT EXISTS(SELECT 1 FROM article_versions INNER JOIN articles ON articles.id = article_versions.article_id WHERE article_versions.cover_url = ? OR article_versions.content LIKE ?)")).
		WithArgs(mediaURL, like).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(0))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT EXISTS(SELECT 1 FROM projects WHERE cover_url = ? OR content_markdown LIKE ?)")).
		WithArgs(mediaURL, like).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(0))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT EXISTS(SELECT 1 FROM users WHERE avatar_url = ?)")).
		WithArgs(mediaURL).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(0))

	referenced, err := NewStore(db).MediaURLReferenced(context.Background(), mediaURL)
	if err != nil {
		t.Fatalf("MediaURLReferenced() error = %v", err)
	}
	if referenced {
		t.Fatal("MediaURLReferenced() = true, want false for an orphaned article version")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}
