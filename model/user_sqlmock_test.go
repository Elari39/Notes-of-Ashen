package model

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func newUserSQLMockStore(t *testing.T) (*Store, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return &Store{db: db}, mock
}

func TestWithUserRegistrationLockCommitsBeforeRelease(t *testing.T) {
	store, mock := newUserSQLMockStore(t)
	mock.ExpectQuery(regexp.QuoteMeta(userRegistrationLockAcquireSQL)).
		WithArgs(userRegistrationLockName).
		WillReturnRows(sqlmock.NewRows([]string{"acquired"}).AddRow(1))
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM users")).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectCommit()
	mock.ExpectQuery(regexp.QuoteMeta(userRegistrationLockReleaseSQL)).
		WithArgs(userRegistrationLockName).
		WillReturnRows(sqlmock.NewRows([]string{"released"}).AddRow(1))

	err := store.WithUserRegistrationLock(context.Background(), func(ctx context.Context, tx *UserRegistrationTx) error {
		count, err := tx.CountUsers(ctx)
		if err != nil {
			return err
		}
		if count != 0 {
			t.Fatalf("user count = %d, want 0", count)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("WithUserRegistrationLock() error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestWithUserRegistrationLockRollsBackAndReleasesOnCallbackError(t *testing.T) {
	store, mock := newUserSQLMockStore(t)
	callbackErr := errors.New("registration failed")
	mock.ExpectQuery(regexp.QuoteMeta(userRegistrationLockAcquireSQL)).
		WithArgs(userRegistrationLockName).
		WillReturnRows(sqlmock.NewRows([]string{"acquired"}).AddRow(1))
	mock.ExpectBegin()
	mock.ExpectRollback()
	mock.ExpectQuery(regexp.QuoteMeta(userRegistrationLockReleaseSQL)).
		WithArgs(userRegistrationLockName).
		WillReturnRows(sqlmock.NewRows([]string{"released"}).AddRow(1))

	err := store.WithUserRegistrationLock(context.Background(), func(context.Context, *UserRegistrationTx) error {
		return callbackErr
	})
	if !errors.Is(err, callbackErr) {
		t.Fatalf("WithUserRegistrationLock() error = %v, want %v", err, callbackErr)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestWithUserRegistrationLockDoesNotBeginWhenLockNotAcquired(t *testing.T) {
	store, mock := newUserSQLMockStore(t)
	mock.ExpectQuery(regexp.QuoteMeta(userRegistrationLockAcquireSQL)).
		WithArgs(userRegistrationLockName).
		WillReturnRows(sqlmock.NewRows([]string{"acquired"}).AddRow(0))

	err := store.WithUserRegistrationLock(context.Background(), func(context.Context, *UserRegistrationTx) error {
		t.Fatal("callback must not run when GET_LOCK is not acquired")
		return nil
	})
	if !errors.Is(err, errRegistrationLockNotAcquired) {
		t.Fatalf("WithUserRegistrationLock() error = %v, want %v", err, errRegistrationLockNotAcquired)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestUpdateUserStatusSafelyRollsBackForLastActiveAdmin(t *testing.T) {
	store, mock := newUserSQLMockStore(t)
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id FROM users WHERE role = 'admin' AND status = 'active' ORDER BY id FOR UPDATE")).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, role, status FROM users WHERE id = ? FOR UPDATE")).
		WithArgs(uint64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "role", "status"}).AddRow(1, "admin", "active"))
	mock.ExpectRollback()

	err := store.UpdateUserStatusSafely(context.Background(), 1, 2, "disabled")
	if !errors.Is(err, ErrLastActiveAdmin) {
		t.Fatalf("UpdateUserStatusSafely() error = %v, want %v", err, ErrLastActiveAdmin)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestUpdateUserStatusSafelyCommitsNormalUpdate(t *testing.T) {
	store, mock := newUserSQLMockStore(t)
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id FROM users WHERE role = 'admin' AND status = 'active' ORDER BY id FOR UPDATE")).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1).AddRow(2))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, role, status FROM users WHERE id = ? FOR UPDATE")).
		WithArgs(uint64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "role", "status"}).AddRow(1, "admin", "active"))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE users SET status = ?, token_version = token_version + 1, updated_at = NOW() WHERE id = ?")).
		WithArgs("disabled", uint64(1)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE refresh_tokens SET revoked_at = NOW() WHERE user_id = ? AND revoked_at IS NULL")).
		WithArgs(uint64(1)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	if err := store.UpdateUserStatusSafely(context.Background(), 1, 2, "disabled"); err != nil {
		t.Fatalf("UpdateUserStatusSafely() error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestUserTokenVersion(t *testing.T) {
	databaseErr := errors.New("database unavailable")
	tests := []struct {
		name    string
		prepare func(sqlmock.Sqlmock)
		want    uint64
		wantErr error
	}{
		{
			name: "success",
			prepare: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta("SELECT token_version FROM users WHERE id = ?")).
					WithArgs(uint64(7)).
					WillReturnRows(sqlmock.NewRows([]string{"token_version"}).AddRow(uint64(5)))
			},
			want: 5,
		},
		{
			name: "not found",
			prepare: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta("SELECT token_version FROM users WHERE id = ?")).
					WithArgs(uint64(7)).
					WillReturnError(sql.ErrNoRows)
			},
			wantErr: ErrNotFound,
		},
		{
			name: "database error",
			prepare: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta("SELECT token_version FROM users WHERE id = ?")).
					WithArgs(uint64(7)).
					WillReturnError(databaseErr)
			},
			wantErr: databaseErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store, mock := newUserSQLMockStore(t)
			tt.prepare(mock)

			version, err := store.UserTokenVersion(context.Background(), 7)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("UserTokenVersion() error = %v, want %v", err, tt.wantErr)
				}
			} else if err != nil || version != tt.want {
				t.Fatalf("UserTokenVersion() = (%d, %v), want (%d, nil)", version, err, tt.want)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}
