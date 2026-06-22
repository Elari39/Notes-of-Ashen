package model

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/go-sql-driver/mysql"
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
		return nil, mysqlConnectError(dataSource, err)
	}
	if maxOpenConns > 0 {
		db.SetMaxOpenConns(maxOpenConns)
	}
	if maxIdleConns > 0 {
		db.SetMaxIdleConns(maxIdleConns)
	}
	db.SetConnMaxLifetime(time.Hour)
	// ConnMaxIdleTime 主动回收空闲连接，避免远程防火墙/NAT 静默关闭空闲 TCP
	// 连接后，连接池仍持有死连接导致下次请求拿到 bad connection / EOF / i/o timeout。
	// 默认值 0 表示永不回收，对远程 MySQL 场景不安全；设 10 分钟（比常见云安全组
	// 空闲超时 5-10 分钟短，又不至于频繁重建连接增加延迟）。
	db.SetConnMaxIdleTime(10 * time.Minute)
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, mysqlConnectErrorWithProbe(dataSource, err, probeMySQLHandshake)
	}
	return db, nil
}

const mysqlStartupHint = "check APP_DATABASE_DSN points to a reachable MySQL server, remote MySQL allows this machine IP, the database is initialized, the account has permissions, and DSN includes charset=utf8mb4&parseTime=true&loc=Local"

func mysqlConnectError(dataSource string, cause error) error {
	return mysqlConnectErrorWithProbe(dataSource, cause, nil)
}

type mysqlProbeFunc func(mysqlTarget) string

func mysqlConnectErrorWithProbe(dataSource string, cause error, probe mysqlProbeFunc) error {
	target, err := safeMySQLTarget(dataSource)
	if err != nil {
		return fmt.Errorf("connect mysql failed (invalid APP_DATABASE_DSN format): %w; %s", cause, mysqlStartupHint)
	}
	diagnosis := ""
	if probe != nil {
		diagnosis = probe(target)
	}
	if diagnosis != "" {
		return fmt.Errorf("connect mysql endpoint %s database %s failed: %w; %s; %s", target.endpoint, target.database, cause, diagnosis, mysqlStartupHint)
	}
	return fmt.Errorf("connect mysql endpoint %s database %s failed: %w; %s", target.endpoint, target.database, cause, mysqlStartupHint)
}

type mysqlTarget struct {
	network  string
	address  string
	endpoint string
	database string
}

func safeMySQLTarget(dataSource string) (mysqlTarget, error) {
	cfg, err := mysql.ParseDSN(dataSource)
	if err != nil {
		return mysqlTarget{}, err
	}

	network := cfg.Net
	if network == "" {
		network = "tcp"
	}

	address := cfg.Addr
	endpoint := address
	if network != "tcp" {
		endpoint = fmt.Sprintf("%s(%s)", network, address)
	}
	if endpoint == "" {
		endpoint = "unknown"
	}

	database := cfg.DBName
	if database == "" {
		database = "unknown"
	}

	return mysqlTarget{network: network, address: address, endpoint: endpoint, database: database}, nil
}

const mysqlHandshakeProbeTimeout = 5 * time.Second

func probeMySQLHandshake(target mysqlTarget) string {
	switch target.network {
	case "tcp", "tcp4", "tcp6":
	default:
		return ""
	}
	if target.address == "" {
		return ""
	}

	conn, err := net.DialTimeout(target.network, target.address, mysqlHandshakeProbeTimeout)
	if err != nil {
		return fmt.Sprintf("diagnosis: TCP probe failed before MySQL handshake: %v", err)
	}
	defer conn.Close()

	if err := conn.SetReadDeadline(time.Now().Add(mysqlHandshakeProbeTimeout)); err != nil {
		return fmt.Sprintf("diagnosis: TCP probe could not set read deadline: %v", err)
	}

	header := make([]byte, 4)
	if _, err := io.ReadFull(conn, header); err != nil {
		if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
			return "diagnosis: TCP connection is accepted, but the server closes before sending a MySQL handshake; check whether APP_DATABASE_DSN uses the real MySQL port, and whether remote firewall/security group/IP whitelist/proxy policy allows MySQL protocol traffic"
		}
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			return "diagnosis: TCP connection is accepted, but no MySQL handshake is received before timeout; check whether APP_DATABASE_DSN points to a MySQL protocol endpoint"
		}
		return fmt.Sprintf("diagnosis: MySQL handshake probe failed after TCP connect: %v", err)
	}

	if header[3] != 0 {
		return fmt.Sprintf("diagnosis: endpoint returned data, but it does not look like an initial MySQL handshake (sequence=%d); check whether APP_DATABASE_DSN points to the MySQL port", header[3])
	}
	return "diagnosis: endpoint returns a MySQL handshake; check credentials, user host permissions, TLS requirements, database existence, and account privileges"
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
