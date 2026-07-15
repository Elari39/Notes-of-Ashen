package user

import (
	"context"
	"testing"
	"time"

	"notes-of-ashen/internal/authutil"
	"notes-of-ashen/internal/svc"
	"notes-of-ashen/internal/types"
	"notes-of-ashen/model"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestUpdateMePreservesOmittedProfileFieldsAndAllowsExplicitClear(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	now := time.Now()
	userRows := func(avatarURL, nickname string) *sqlmock.Rows {
		return sqlmock.NewRows([]string{
			"id", "account", "password_hash", "email", "avatar_url", "nickname", "role", "status", "created_at", "updated_at",
		}).AddRow(uint64(7), "writer", "hash", "writer@example.com", avatarURL, nickname, "user", "active", now, now)
	}
	mock.ExpectQuery("SELECT id, account, password_hash, email, avatar_url, nickname, role, status, created_at, updated_at").
		WithArgs(uint64(7)).
		WillReturnRows(userRows("https://example.com/avatar.png", "Writer"))
	empty := ""
	mock.ExpectExec("UPDATE users SET email = \\?, avatar_url = \\?, nickname = \\? WHERE id = \\?").
		WithArgs("writer@example.com", "", "Writer", uint64(7)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery("SELECT id, account, password_hash, email, avatar_url, nickname, role, status, created_at, updated_at").
		WithArgs(uint64(7)).
		WillReturnRows(userRows("", "Writer"))

	ctx := authutil.WithUser(context.Background(), 7, authutil.RoleUser)
	resp, err := UpdateMe(ctx, &svc.ServiceContext{Store: model.NewStore(db)}, types.UpdateMeReq{AvatarURL: &empty})
	if err != nil {
		t.Fatalf("UpdateMe() error = %v", err)
	}
	if resp.AvatarURL != "" || resp.Nickname != "Writer" || resp.Email != "writer@example.com" {
		t.Fatalf("unexpected response: %#v", resp)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
