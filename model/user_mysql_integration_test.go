package model

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-sql-driver/mysql"
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
	if _, err = db.Exec(`DROP TABLE IF EXISTS users`); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`CREATE TABLE users (id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY, account VARCHAR(64) NOT NULL UNIQUE, password_hash VARCHAR(128) NOT NULL, email VARCHAR(128) NOT NULL UNIQUE, avatar_url VARCHAR(255) DEFAULT '', nickname VARCHAR(64) DEFAULT '', role VARCHAR(20) DEFAULT 'user', status VARCHAR(20) DEFAULT 'active', token_version BIGINT UNSIGNED NOT NULL DEFAULT 0, created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP, updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP, INDEX idx_users_role_status_id (role,status,id)) ENGINE=InnoDB`); err != nil {
		t.Fatal(err)
	}
	return &Store{db: db}
}

func TestMySQLConcurrentFirstRegistrationCreatesOneAdmin(t *testing.T) {
	store := testMySQLStore(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for i := 0; i < 2; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
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
		}()
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
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	for i := 1; i <= 2; i++ {
		if _, err := store.db.Exec(`INSERT INTO users(account,password_hash,email,role,status) VALUES(?,?,?,'admin','active')`, fmt.Sprintf("admin%d", i), "hash", fmt.Sprintf("a%d@example.com", i)); err != nil {
			t.Fatal(err)
		}
	}
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for _, p := range [][2]uint64{{1, 2}, {2, 1}} {
		p := p
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs <- store.UpdateUserStatusSafely(ctx, p[0], p[1], "disabled")
		}()
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
}
