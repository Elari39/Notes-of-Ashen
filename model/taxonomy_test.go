package model

import (
	"context"
	"database/sql"
	"reflect"
	"testing"
)

type execCall struct {
	query string
	args  []interface{}
}

type fakeExecContexter struct {
	calls    []execCall
	affected []int64
}

func (f *fakeExecContexter) ExecContext(_ context.Context, query string, args ...interface{}) (sql.Result, error) {
	f.calls = append(f.calls, execCall{query: query, args: args})
	affected := int64(1)
	if index := len(f.calls) - 1; index < len(f.affected) {
		affected = f.affected[index]
	}
	return resultStub{affected: affected}, nil
}

func TestDeleteCategoryTxClearsArticleCategory(t *testing.T) {
	execer := &fakeExecContexter{}

	if err := deleteCategoryTx(context.Background(), execer, 12); err != nil {
		t.Fatalf("deleteCategoryTx() error = %v", err)
	}

	want := []execCall{
		{query: "UPDATE articles SET category_id = NULL WHERE category_id = ?", args: []interface{}{uint64(12)}},
		{query: "DELETE FROM categories WHERE id = ?", args: []interface{}{uint64(12)}},
	}
	if !reflect.DeepEqual(execer.calls, want) {
		t.Fatalf("calls = %#v, want %#v", execer.calls, want)
	}
}

func TestDeleteTagTxClearsArticleAndProjectRelations(t *testing.T) {
	execer := &fakeExecContexter{}

	if err := deleteTagTx(context.Background(), execer, 34); err != nil {
		t.Fatalf("deleteTagTx() error = %v", err)
	}

	want := []execCall{
		{query: "DELETE FROM article_tags WHERE tag_id = ?", args: []interface{}{uint64(34)}},
		{query: "DELETE FROM project_tags WHERE tag_id = ?", args: []interface{}{uint64(34)}},
		{query: "DELETE FROM tags WHERE id = ?", args: []interface{}{uint64(34)}},
	}
	if !reflect.DeepEqual(execer.calls, want) {
		t.Fatalf("calls = %#v, want %#v", execer.calls, want)
	}
}
