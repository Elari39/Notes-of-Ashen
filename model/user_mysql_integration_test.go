package model

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-sql-driver/mysql"
	"notes-of-ashen/deploy/mysql/migrations"
	"notes-of-ashen/internal/migration"
)

func testMySQLStore(t *testing.T) *Store {
	t.Helper()
	dsn := strings.TrimSpace(os.Getenv("APP_TEST_DATABASE_DSN"))
	if dsn == "" {
		t.Skip("APP_TEST_DATABASE_DSN is not set")
	}
	cfg, err := mysql.ParseDSN(dsn)
	if err != nil {
		t.Fatalf("parse APP_TEST_DATABASE_DSN: %v", err)
	}
	if !strings.HasSuffix(strings.ToLower(cfg.DBName), "_test") {
		t.Fatalf("refusing to reset non-test database %q", cfg.DBName)
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	// 使用生产迁移建立本轮专用数据库，避免手写夹具与真实事务涉及的表漂移。
	// 不重置调用方给出的库；测试账号需有创建/删除隔离数据库的权限。
	database := "noa_users_" + strings.ToLower(rand.Text()) + "_test"
	if _, err = db.ExecContext(t.Context(), "CREATE DATABASE `"+database+"` CHARACTER SET utf8mb4"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := db.Exec("DROP DATABASE `" + database + "`"); err != nil {
			t.Errorf("drop isolated MySQL test database: %v", err)
		}
	})
	cfg.DBName = database
	migrationDB, err := migration.Open(cfg.FormatDSN())
	if err != nil {
		t.Fatal(err)
	}
	defer migrationDB.Close()
	if err := migration.Run(t.Context(), migrationDB, migrations.FS); err != nil {
		t.Fatal(err)
	}
	testDB, err := Open(cfg.FormatDSN(), 20, 10)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = testDB.Close() })
	return &Store{db: testDB}
}

func TestMySQLConcurrentFirstRegistrationCreatesOneAdmin(t *testing.T) {
	store := testMySQLStore(t)
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for i := range 2 {
		wg.Go(func() {
			errs <- store.WithUserRegistrationLock(ctx, func(ctx context.Context, tx *UserRegistrationTx) error {
				count, err := tx.CountUsers(ctx)
				if err != nil {
					return err
				}
				role := "user"
				if count == 0 {
					role = "admin"
				}
				_, err = tx.CreateUser(ctx, UserCreate{Account: fmt.Sprintf("user%d", i), PasswordHash: "hash", Email: fmt.Sprintf("u%d@example.com", i), Role: role})
				return err
			})
		})
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	var admins int
	if err := store.db.QueryRow(`SELECT COUNT(*) FROM users WHERE role='admin'`).Scan(&admins); err != nil {
		t.Fatal(err)
	}
	if admins != 1 {
		t.Fatalf("admin count = %d, want 1", admins)
	}
}

func TestMySQLConcurrentAdminDisableKeepsOneActive(t *testing.T) {
	store := testMySQLStore(t)
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	for i := 1; i <= 2; i++ {
		if _, err := store.db.Exec(`INSERT INTO users(account,password_hash,email,role,status) VALUES(?,?,?,'admin','active')`, fmt.Sprintf("admin%d", i), "hash", fmt.Sprintf("a%d@example.com", i)); err != nil {
			t.Fatal(err)
		}
		if _, err := store.db.Exec(`INSERT INTO refresh_tokens(user_id,token_hash,expires_at) VALUES(?,?,?)`, i, fmt.Sprintf("session%d", i), time.Now().Add(time.Hour)); err != nil {
			t.Fatal(err)
		}
	}
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for _, p := range [][2]uint64{{1, 2}, {2, 1}} {
		wg.Go(func() {
			errs <- store.UpdateUserStatusSafely(ctx, p[0], p[1], "disabled")
		})
	}
	wg.Wait()
	close(errs)
	succeeded := 0
	rejected := 0
	for err := range errs {
		switch {
		case err == nil:
			succeeded++
		case errors.Is(err, ErrLastActiveAdmin):
			rejected++
		default:
			t.Fatalf("unexpected concurrent update error: %v", err)
		}
	}
	if succeeded != 1 || rejected != 1 {
		t.Fatalf("concurrent results: succeeded=%d rejected=%d, want 1/1", succeeded, rejected)
	}
	var active int
	if err := store.db.QueryRow(`SELECT COUNT(*) FROM users WHERE role='admin' AND status='active'`).Scan(&active); err != nil {
		t.Fatal(err)
	}
	if active != 1 {
		t.Fatalf("active admin count = %d, want 1", active)
	}
	var revoked, valid, versioned int
	if err := store.db.QueryRow(`SELECT COUNT(*) FROM refresh_tokens r JOIN users u ON u.id=r.user_id WHERE u.status='disabled' AND r.revoked_at IS NOT NULL`).Scan(&revoked); err != nil {
		t.Fatal(err)
	}
	if err := store.db.QueryRow(`SELECT COUNT(*) FROM refresh_tokens r JOIN users u ON u.id=r.user_id WHERE u.status='active' AND r.revoked_at IS NULL`).Scan(&valid); err != nil {
		t.Fatal(err)
	}
	if err := store.db.QueryRow(`SELECT COUNT(*) FROM users WHERE status='disabled' AND token_version=1`).Scan(&versioned); err != nil {
		t.Fatal(err)
	}
	if revoked != 1 || valid != 1 || versioned != 1 {
		t.Fatalf("session invariants: revoked=%d valid=%d versioned=%d, want 1/1/1", revoked, valid, versioned)
	}
}
