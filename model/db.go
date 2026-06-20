package model

import (
	"context"
	"database/sql"
	"errors"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

var ErrNotFound = errors.New("record not found")

type Store struct {
	db *sql.DB
}

type rowScanner interface {
	Scan(dest ...interface{}) error
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

func Open(dataSource string, maxOpenConns, maxIdleConns int) (*sql.DB, error) {
	db, err := sql.Open("mysql", dataSource)
	if err != nil {
		return nil, err
	}
	if maxOpenConns > 0 {
		db.SetMaxOpenConns(maxOpenConns)
	}
	if maxIdleConns > 0 {
		db.SetMaxIdleConns(maxIdleConns)
	}
	db.SetConnMaxLifetime(time.Hour)
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

func (s *Store) DB() *sql.DB {
	return s.db
}

func scanErr(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

func uint64FromNull(v sql.NullInt64) uint64 {
	if !v.Valid || v.Int64 <= 0 {
		return 0
	}
	return uint64(v.Int64)
}

func stringFromNull(v sql.NullString) string {
	if !v.Valid {
		return ""
	}
	return v.String
}

func timeFromNull(v sql.NullTime) *time.Time {
	if !v.Valid {
		return nil
	}
	t := v.Time
	return &t
}

func nullableUint64(v uint64) interface{} {
	if v == 0 {
		return nil
	}
	return v
}

func nullableTime(v *time.Time) interface{} {
	if v == nil {
		return nil
	}
	return *v
}

func WithTx(ctx context.Context, db *sql.DB, fn func(*sql.Tx) error) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := tx.Commit(); err != nil {
		// Commit 失败时事务状态未定，回滚以避免业务误以为写入成功。
		_ = tx.Rollback()
		return err
	}
	return nil
}
