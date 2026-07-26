// Package migration 提供应用内置 SQL 的发现、执行与状态校验。
//
// 迁移文件由 deploy/mysql/migrations 通过 embed.FS 随二进制发布。迁移一旦
// 发布即不可修改；如需修复历史问题，应新增一个更高版本的前向迁移。
package migration

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
)

const (
	// LockName 是 MySQL advisory lock 的固定名称。所有 API 镜像版本使用同一
	// 名称，确保滚动发布或误配的多个 migrate job 不会并发执行 DDL。
	LockName = "notes_of_ashen_schema_migration"

	lockWaitTimeout    = 60 * time.Second
	lockReleaseTimeout = 5 * time.Second

	maxMigrationRunErrorBytes = 16 << 10
)

var (
	// ErrNoMigrations 表示嵌入的迁移目录为空。空目录通常说明 Docker 构建
	// 上下文或 go:embed 配置错误，不能继续把数据库标记为已升级。
	ErrNoMigrations = errors.New("no embedded migration files found")
	// ErrLockNotAcquired 表示另一个迁移进程在等待窗口内仍持有数据库锁。
	ErrLockNotAcquired = errors.New("database migration lock was not acquired")
	// ErrRollbackSchemaIncompatible 表示目标应用版本无法读取当前数据库的
	// 更高 schema。代码回退不会自动回滚数据库，必须先恢复备份或使用兼容镜像。
	ErrRollbackSchemaIncompatible = errors.New("target image migration version is older than database schema")
)

var migrationFilename = regexp.MustCompile(`^([0-9]{6})_([A-Za-z0-9][A-Za-z0-9_-]*)\.sql$`)

var destructiveMigrationWarnings = map[uint64]string{
	2:  "包含历史数据写入或清理操作",
	13: "会删除已废弃的 traffic_geo 表",
	23: "会清理孤儿 article_versions 数据",
	24: "会清理孤儿关系并增加数据库外键约束",
}

// Migration 是一个已发现的、不可变的 SQL 迁移文件。
// Name 保存完整文件名，使数据库中的执行记录可以直接对应仓库文件。
type Migration struct {
	Version  uint64
	Name     string
	Checksum string

	sql string
}

// Drift 描述数据库记录和内置迁移文件之间的一个不可变性冲突。
type Drift struct {
	Version  uint64
	Expected string
	Actual   string
}

// StateError 是迁移状态不满足当前二进制要求时的可定位错误。
// 它既供 migrate job 阻止继续执行，也供 /healthz 展示缺失版本或校验和漂移。
type StateError struct {
	MetadataMissing bool
	Missing         []uint64
	ChecksumDrifts  []Drift
	NameDrifts      []Drift
}

func (e *StateError) Error() string {
	if e == nil {
		return ""
	}
	if e.MetadataMissing {
		return "schema migration metadata is missing; run the migrate service"
	}

	parts := make([]string, 0, 3)
	if len(e.Missing) > 0 {
		parts = append(parts, "missing migration versions: "+formatVersions(e.Missing))
	}
	if len(e.ChecksumDrifts) > 0 {
		parts = append(parts, "migration checksum drift: "+formatDrifts(e.ChecksumDrifts))
	}
	if len(e.NameDrifts) > 0 {
		parts = append(parts, "migration filename drift: "+formatDrifts(e.NameDrifts))
	}
	if len(parts) == 0 {
		return "schema migration state is invalid"
	}
	return strings.Join(parts, "; ")
}

func formatVersions(versions []uint64) string {
	labels := make([]string, 0, len(versions))
	for _, version := range versions {
		labels = append(labels, formatVersion(version))
	}
	return strings.Join(labels, ", ")
}

func formatDrifts(drifts []Drift) string {
	labels := make([]string, 0, len(drifts))
	for _, drift := range drifts {
		labels = append(labels, fmt.Sprintf("%s (expected %q, got %q)", formatVersion(drift.Version), drift.Expected, drift.Actual))
	}
	return strings.Join(labels, ", ")
}

func formatVersion(version uint64) string {
	return fmt.Sprintf("%06d", version)
}

// Discover 读取根目录中的编号 SQL 文件，验证其命名、连续编号和 SHA-256 校验和。
// SQL 原始字节参与校验，因此换行、注释和空白的修改都会被视为已发布迁移漂移。
func Discover(source fs.FS) ([]Migration, error) {
	if source == nil {
		return nil, fmt.Errorf("read migration files: nil filesystem")
	}

	entries, err := fs.ReadDir(source, ".")
	if err != nil {
		return nil, fmt.Errorf("read migration files: %w", err)
	}

	migrations := make([]Migration, 0, len(entries))
	seenVersions := make(map[uint64]string, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if path.Ext(entry.Name()) != ".sql" {
			continue
		}

		matches := migrationFilename.FindStringSubmatch(entry.Name())
		if matches == nil {
			return nil, fmt.Errorf("invalid migration filename %q: expected NNNNNN_name.sql", entry.Name())
		}
		version, err := strconv.ParseUint(matches[1], 10, 64)
		if err != nil || version == 0 {
			return nil, fmt.Errorf("invalid migration version in %q", entry.Name())
		}
		if previous, exists := seenVersions[version]; exists {
			return nil, fmt.Errorf("duplicate migration version %s in %q and %q", formatVersion(version), previous, entry.Name())
		}

		contents, err := fs.ReadFile(source, entry.Name())
		if err != nil {
			return nil, fmt.Errorf("read migration %q: %w", entry.Name(), err)
		}
		if strings.TrimSpace(string(contents)) == "" {
			return nil, fmt.Errorf("migration %q is empty", entry.Name())
		}
		digest := sha256.Sum256(contents)
		migrations = append(migrations, Migration{
			Version:  version,
			Name:     entry.Name(),
			Checksum: hex.EncodeToString(digest[:]),
			sql:      string(contents),
		})
		seenVersions[version] = entry.Name()
	}

	if len(migrations) == 0 {
		return nil, ErrNoMigrations
	}
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})
	for index, item := range migrations {
		want := uint64(index + 1)
		if item.Version != want {
			return nil, fmt.Errorf("migration versions must be contiguous from %s: found %s after %s", formatVersion(1), formatVersion(item.Version), formatVersion(uint64(index)))
		}
	}
	return migrations, nil
}

// LatestVersion 返回嵌入迁移文件中的最高版本，供发布工具读取目标镜像能力。
func LatestVersion(source fs.FS) (uint64, error) {
	migrations, err := Discover(source)
	if err != nil {
		return 0, err
	}
	return migrations[len(migrations)-1].Version, nil
}

// Open creates a migration-only MySQL pool. multiStatements is deliberately
// enabled because each immutable SQL file is submitted as one whole script on
// the dedicated connection used for its advisory lock.
func Open(dataSource string) (*sql.DB, error) {
	config, err := mysql.ParseDSN(dataSource)
	if err != nil {
		return nil, fmt.Errorf("parse MySQL DSN for migration: %w", err)
	}
	config.MultiStatements = true

	db, err := sql.Open("mysql", config.FormatDSN())
	if err != nil {
		return nil, fmt.Errorf("open MySQL for migration: %w", err)
	}
	// migrate-only 进程不会服务业务请求；单连接既减少资源占用，也保证锁、脚本
	// 执行和元数据记录都落在同一个 MySQL session。
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping MySQL for migration: %w", err)
	}
	return db, nil
}

// Run 在获得 MySQL advisory lock 后执行所有尚未应用的迁移。每个 SQL 文件以
// 单条 multiStatements 请求提交；只有脚本成功且元数据事务提交后才标记为已应用。
func Run(ctx context.Context, db *sql.DB, source fs.FS) (err error) {
	migrations, err := Discover(source)
	if err != nil {
		return err
	}
	if db == nil {
		return errors.New("database is not configured")
	}

	conn, err := db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("acquire migration database connection: %w", err)
	}
	defer func() {
		if closeErr := conn.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("close migration database connection: %w", closeErr)
		}
	}()

	if err = acquireLock(ctx, conn); err != nil {
		return err
	}
	defer func() {
		if releaseErr := releaseLock(conn); releaseErr != nil && err == nil {
			err = releaseErr
		}
	}()

	if err = ensureMetadata(ctx, conn); err != nil {
		return err
	}
	applied, err := loadApplied(ctx, conn)
	if err != nil {
		return err
	}
	// migrate job 允许尚未执行的版本存在；但已经标记为已应用的版本必须
	// 与内置文件完全一致，避免被改写的历史 SQL 在新环境继续扩散。
	if stateErr := validateState(migrations, applied, false); stateErr != nil {
		return stateErr
	}

	for _, item := range migrations {
		if _, exists := applied[item.Version]; exists {
			continue
		}
		if warning, destructive := destructiveMigrationWarnings[item.Version]; destructive {
			log.Printf("[migration] warning version=%s name=%s: %s；生产执行前应确认已完成数据库备份", formatVersion(item.Version), item.Name, warning)
		}
		log.Printf("[migration] applying version=%s name=%s", formatVersion(item.Version), item.Name)
		if err = runOne(ctx, conn, item); err != nil {
			return err
		}
		log.Printf("[migration] applied version=%s name=%s", formatVersion(item.Version), item.Name)
	}
	return nil
}

// Check 只读取迁移状态，不创建元数据表、不获取锁、不执行 SQL。它适用于服务
// readiness 探针：数据库缺少版本、历史文件被改写或文件被重命名时均能给出原因。
func Check(ctx context.Context, db *sql.DB, source fs.FS) error {
	migrations, err := Discover(source)
	if err != nil {
		return err
	}
	if db == nil {
		return errors.New("database is not configured")
	}
	applied, err := loadApplied(ctx, db)
	if err != nil {
		if isMetadataTableMissing(err) {
			return &StateError{MetadataMissing: true}
		}
		return err
	}
	return validateState(migrations, applied, true)
}

type appliedMigration struct {
	name     string
	checksum string
}

type migrationQueryer interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

type migrationRowQueryer interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// CurrentVersion 只读取数据库已应用的最高迁移版本，不创建元数据表或执行 DDL。
func CurrentVersion(ctx context.Context, queryer migrationRowQueryer) (uint64, error) {
	var version uint64
	if err := queryer.QueryRowContext(ctx, `SELECT COALESCE(MAX(version), 0) FROM schema_migrations`).Scan(&version); err != nil {
		return 0, fmt.Errorf("read current schema migration version: %w", err)
	}
	return version, nil
}

// ValidateRollbackCompatibility 校验代码回退是否会让目标镜像落后于数据库 schema。
// allow=true 仅用于显式危险确认，不改变错误语义，调用方仍应记录并提示备份恢复。
func ValidateRollbackCompatibility(databaseVersion, targetVersion uint64, allow bool) error {
	if databaseVersion <= targetVersion || allow {
		return nil
	}
	return fmt.Errorf("%w: database=%s target=%s", ErrRollbackSchemaIncompatible, formatVersion(databaseVersion), formatVersion(targetVersion))
}

func loadApplied(ctx context.Context, queryer migrationQueryer) (map[uint64]appliedMigration, error) {
	rows, err := queryer.QueryContext(ctx, `SELECT version, name, checksum FROM schema_migrations ORDER BY version`)
	if err != nil {
		return nil, fmt.Errorf("read schema migration state: %w", err)
	}
	defer rows.Close()

	applied := make(map[uint64]appliedMigration)
	for rows.Next() {
		var version uint64
		var item appliedMigration
		if err := rows.Scan(&version, &item.name, &item.checksum); err != nil {
			return nil, fmt.Errorf("scan schema migration state: %w", err)
		}
		if _, exists := applied[version]; exists {
			return nil, fmt.Errorf("schema migration state contains duplicate version %s", formatVersion(version))
		}
		applied[version] = item
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate schema migration state: %w", err)
	}
	return applied, nil
}

func validateState(migrations []Migration, applied map[uint64]appliedMigration, requireAll bool) error {
	stateErr := &StateError{}
	for _, item := range migrations {
		recorded, exists := applied[item.Version]
		if !exists {
			if requireAll {
				stateErr.Missing = append(stateErr.Missing, item.Version)
			}
			continue
		}
		if recorded.checksum != item.Checksum {
			stateErr.ChecksumDrifts = append(stateErr.ChecksumDrifts, Drift{
				Version:  item.Version,
				Expected: item.Checksum,
				Actual:   recorded.checksum,
			})
		}
		if recorded.name != item.Name {
			stateErr.NameDrifts = append(stateErr.NameDrifts, Drift{
				Version:  item.Version,
				Expected: item.Name,
				Actual:   recorded.name,
			})
		}
	}
	if len(stateErr.Missing) == 0 && len(stateErr.ChecksumDrifts) == 0 && len(stateErr.NameDrifts) == 0 {
		return nil
	}
	return stateErr
}

func acquireLock(ctx context.Context, conn *sql.Conn) error {
	var acquired sql.NullInt64
	if err := conn.QueryRowContext(ctx, `SELECT GET_LOCK(?, ?)`, LockName, int(lockWaitTimeout.Seconds())).Scan(&acquired); err != nil {
		return fmt.Errorf("acquire database migration lock: %w", err)
	}
	if !acquired.Valid || acquired.Int64 != 1 {
		return fmt.Errorf("%w after %s", ErrLockNotAcquired, lockWaitTimeout)
	}
	return nil
}

func releaseLock(conn *sql.Conn) error {
	ctx, cancel := context.WithTimeout(context.Background(), lockReleaseTimeout)
	defer cancel()

	var released sql.NullInt64
	if err := conn.QueryRowContext(ctx, `SELECT RELEASE_LOCK(?)`, LockName).Scan(&released); err != nil {
		return fmt.Errorf("release database migration lock: %w", err)
	}
	if !released.Valid || released.Int64 != 1 {
		return errors.New("release database migration lock: lock is not held by this connection")
	}
	return nil
}

const createMigrationsTable = `CREATE TABLE IF NOT EXISTS schema_migrations (
    version BIGINT UNSIGNED NOT NULL,
    name VARCHAR(255) NOT NULL,
    checksum CHAR(64) NOT NULL,
    execution_ms BIGINT UNSIGNED NOT NULL,
    applied_at DATETIME(6) NOT NULL,
    PRIMARY KEY (version)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci`

const createMigrationRunsTable = `CREATE TABLE IF NOT EXISTS schema_migration_runs (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    version BIGINT UNSIGNED NOT NULL,
    name VARCHAR(255) NOT NULL,
    checksum CHAR(64) NOT NULL,
    status VARCHAR(16) NOT NULL,
    started_at DATETIME(6) NOT NULL,
    finished_at DATETIME(6) NOT NULL,
    execution_ms BIGINT UNSIGNED NOT NULL,
    error_message TEXT NULL,
    PRIMARY KEY (id),
    INDEX idx_schema_migration_runs_version_started (version, started_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci`

func ensureMetadata(ctx context.Context, conn *sql.Conn) error {
	if _, err := conn.ExecContext(ctx, createMigrationsTable); err != nil {
		return fmt.Errorf("create schema_migrations table: %w", err)
	}
	if _, err := conn.ExecContext(ctx, createMigrationRunsTable); err != nil {
		return fmt.Errorf("create schema_migration_runs table: %w", err)
	}
	return nil
}

func runOne(ctx context.Context, conn *sql.Conn, item Migration) error {
	started := time.Now().UTC()
	_, err := conn.ExecContext(ctx, item.sql)
	finished := time.Now().UTC()
	duration := uint64(finished.Sub(started).Milliseconds())
	if err != nil {
		recordErr := recordRun(ctx, conn, item, "failed", started, finished, duration, truncateError(err.Error()))
		if recordErr != nil {
			return fmt.Errorf("execute migration %s (%s): %w; record failed run: %v", formatVersion(item.Version), item.Name, err, recordErr)
		}
		return fmt.Errorf("execute migration %s (%s): %w", formatVersion(item.Version), item.Name, err)
	}
	if err := recordSuccess(ctx, conn, item, started, finished, duration); err != nil {
		return fmt.Errorf("record successful migration %s (%s): %w", formatVersion(item.Version), item.Name, err)
	}
	return nil
}

func recordSuccess(ctx context.Context, conn *sql.Conn, item Migration, started, finished time.Time, duration uint64) error {
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	if _, err := tx.ExecContext(ctx, `INSERT INTO schema_migrations (version, name, checksum, execution_ms, applied_at) VALUES (?, ?, ?, ?, ?)`, item.Version, item.Name, item.Checksum, duration, finished); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO schema_migration_runs (version, name, checksum, status, started_at, finished_at, execution_ms, error_message) VALUES (?, ?, ?, ?, ?, ?, ?, NULL)`, item.Version, item.Name, item.Checksum, "success", started, finished, duration); err != nil {
		return err
	}
	return tx.Commit()
}

func recordRun(ctx context.Context, conn *sql.Conn, item Migration, status string, started, finished time.Time, duration uint64, message string) error {
	_, err := conn.ExecContext(ctx, `INSERT INTO schema_migration_runs (version, name, checksum, status, started_at, finished_at, execution_ms, error_message) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, item.Version, item.Name, item.Checksum, status, started, finished, duration, message)
	return err
}

func truncateError(message string) string {
	if len(message) <= maxMigrationRunErrorBytes {
		return message
	}
	limit := maxMigrationRunErrorBytes - len("…(truncated)")
	for limit > 0 && limit < len(message) && (message[limit]&0xc0) == 0x80 {
		limit--
	}
	return message[:limit] + "…(truncated)"
}

func isMetadataTableMissing(err error) bool {
	var mysqlErr *mysql.MySQLError
	return errors.As(err, &mysqlErr) && mysqlErr.Number == 1146
}
