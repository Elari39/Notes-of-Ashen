package model

import (
	"context"
	"errors"
	"testing"
)

var errRowsAffected = errors.New("rows affected failed")

type resultStub struct {
	affected int64
	err      error
}

func (r resultStub) LastInsertId() (int64, error) {
	return 0, nil
}

func (r resultStub) RowsAffected() (int64, error) {
	return r.affected, r.err
}

func TestRequireAffectedRejectsZeroRows(t *testing.T) {
	if err := requireAffected(resultStub{affected: 1}); err != nil {
		t.Fatalf("requireAffected() error = %v, want nil", err)
	}
	if err := requireAffected(resultStub{}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("requireAffected() error = %v, want ErrNotFound", err)
	}
}

func TestRequireUpdateAffectedAllowsChangedRows(t *testing.T) {
	called := false
	err := requireUpdateAffected(context.Background(), resultStub{affected: 1}, func(context.Context) error {
		called = true
		return nil
	})
	if err != nil {
		t.Fatalf("requireUpdateAffected() error = %v, want nil", err)
	}
	if called {
		t.Fatal("existence check should not be called when rows changed")
	}
}

func TestRequireUpdateAffectedAllowsExistingUnchangedRow(t *testing.T) {
	err := requireUpdateAffected(context.Background(), resultStub{}, func(context.Context) error {
		return nil
	})
	if err != nil {
		t.Fatalf("requireUpdateAffected() error = %v, want nil", err)
	}
}

func TestRequireUpdateAffectedRejectsMissingRow(t *testing.T) {
	err := requireUpdateAffected(context.Background(), resultStub{}, func(context.Context) error {
		return ErrNotFound
	})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("requireUpdateAffected() error = %v, want ErrNotFound", err)
	}
}

func TestRequireUpdateAffectedReturnsRowsAffectedError(t *testing.T) {
	err := requireUpdateAffected(context.Background(), resultStub{err: errRowsAffected}, func(context.Context) error {
		return nil
	})
	if !errors.Is(err, errRowsAffected) {
		t.Fatalf("requireUpdateAffected() error = %v, want rows affected error", err)
	}
}
