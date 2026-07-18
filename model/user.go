package model

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/zeromicro/go-zero/core/logx"
)

var (
	errRegistrationLockNotAcquired = errors.New("user registration lock not acquired")
	ErrCannotDisableSelf           = errors.New("cannot disable yourself")
	ErrCannotDowngradeSelf         = errors.New("cannot downgrade yourself")
	ErrLastActiveAdmin             = errors.New("at least one active admin is required")
)

const (
	createUserSQL = `
	INSERT INTO users (account, password_hash, email, avatar_url, nickname, role)
	VALUES (?, ?, ?, ?, ?, ?)`
	userRegistrationLockName       = "notes-of-ashen:user-registration"
	userRegistrationLockAcquireSQL = "SELECT GET_LOCK(?, 10)"
	userRegistrationLockReleaseSQL = "SELECT RELEASE_LOCK(?)"
)

type User struct {
	ID           uint64
	Account      string
	PasswordHash string
	Email        string
	AvatarURL    string
	Nickname     string
	Role         string
	Status       string
	TokenVersion uint64
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type UserCreate struct {
	Account      string
	PasswordHash string
	Email        string
	AvatarURL    string
	Nickname     string
	Role         string
}

type UserUpdate struct {
	Email     string
	AvatarURL string
	Nickname  string
}

// UserRegistrationTx 将首位用户判定和创建限制在同一事务、同一数据库连接内。
// 注册回调只能通过该 facade 访问相关数据，避免重新使用连接池破坏 GET_LOCK 的会话语义。
type UserRegistrationTx struct {
	tx *sql.Tx
}

func (s *Store) CountUsers(ctx context.Context) (int64, error) {
	var count int64
	err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM users").Scan(&count)
	return count, err
}

// AdminExists 在 GET_LOCK 保护的会话内对 users 表加行锁，确认是否已存在 admin。
// 用作首位 admin 提升的 DB 双保险：即使 GET_LOCK 失效，并发注册也只会让先落库者成为 admin，
// 后到者读到 admin 已存在则降级为普通用户，避免出现多个 admin。
func (s *Store) AdminExists(ctx context.Context) (bool, error) {
	var one int
	err := s.db.QueryRowContext(ctx, "SELECT 1 FROM users WHERE role = 'admin' LIMIT 1 FOR UPDATE").Scan(&one)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (s *Store) WithUserRegistrationLock(ctx context.Context, fn func(context.Context, *UserRegistrationTx) error) error {
	conn, err := s.db.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	// GET_LOCK 返回 1 表示加锁成功，0 表示超时，NULL 表示出错（如线程被杀死）。
	// 必须校验返回值，否则超时/失败会被当作加锁成功，并发注册可能让多个用户都成为 admin。
	var acquired sql.NullInt64
	if err := conn.QueryRowContext(ctx, userRegistrationLockAcquireSQL, userRegistrationLockName).Scan(&acquired); err != nil {
		return err
	}
	if !acquired.Valid || acquired.Int64 != 1 {
		return errRegistrationLockNotAcquired
	}
	defer func() {
		releaseCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		var released sql.NullInt64
		if err := conn.QueryRowContext(releaseCtx, userRegistrationLockReleaseSQL, userRegistrationLockName).Scan(&released); err != nil {
			logx.Errorf("release user registration lock failed: %v", err)
		} else if !released.Valid || released.Int64 != 1 {
			logx.Errorf("release user registration lock returned non-one: %v", released)
		}
	}()

	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if err := fn(ctx, &UserRegistrationTx{tx: tx}); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := tx.Commit(); err != nil {
		_ = tx.Rollback()
		return err
	}
	return nil
}

func (tx *UserRegistrationTx) CountUsers(ctx context.Context) (int64, error) {
	var count int64
	err := tx.tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM users").Scan(&count)
	return count, err
}

func (tx *UserRegistrationTx) AdminExists(ctx context.Context) (bool, error) {
	var one int
	err := tx.tx.QueryRowContext(ctx, "SELECT 1 FROM users WHERE role = 'admin' LIMIT 1 FOR UPDATE").Scan(&one)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

func (tx *UserRegistrationTx) RegistrationEnabled(ctx context.Context) (bool, error) {
	var raw string
	err := tx.tx.QueryRowContext(ctx, "SELECT setting_value FROM site_settings WHERE setting_key = ? LIMIT 1", RegistrationEnabledKey).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return true, nil
	}
	var mysqlErr *mysql.MySQLError
	if errors.As(err, &mysqlErr) && mysqlErr.Number == 1146 {
		return true, nil
	}
	if err != nil {
		return false, err
	}
	return raw == "true" || raw == "1", nil
}

func (tx *UserRegistrationTx) FindUserByAccountOrEmail(ctx context.Context, value string) (*User, error) {
	row := tx.tx.QueryRowContext(ctx, `
	SELECT id, account, password_hash, email, avatar_url, nickname, role, status, token_version, created_at, updated_at
	FROM users WHERE account = ? OR email = ? LIMIT 1`, value, value)
	return scanUser(row)
}

func (tx *UserRegistrationTx) CreateUser(ctx context.Context, in UserCreate) (uint64, error) {
	res, err := tx.tx.ExecContext(ctx, createUserSQL,
		in.Account, in.PasswordHash, in.Email, in.AvatarURL, in.Nickname, in.Role)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	return uint64(id), err
}

func (s *Store) CreateUser(ctx context.Context, in UserCreate) (uint64, error) {
	res, err := s.db.ExecContext(ctx, createUserSQL,
		in.Account, in.PasswordHash, in.Email, in.AvatarURL, in.Nickname, in.Role)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	return uint64(id), nil
}

func (s *Store) FindUserByID(ctx context.Context, id uint64) (*User, error) {
	row := s.db.QueryRowContext(ctx, `
	SELECT id, account, password_hash, email, avatar_url, nickname, role, status, token_version, created_at, updated_at
	FROM users WHERE id = ?`, id)
	return scanUser(row)
}

func (s *Store) FindUserByAccountOrEmail(ctx context.Context, accountOrEmail string) (*User, error) {
	row := s.db.QueryRowContext(ctx, `
	SELECT id, account, password_hash, email, avatar_url, nickname, role, status, token_version, created_at, updated_at
	FROM users WHERE account = ? OR email = ? LIMIT 1`, accountOrEmail, accountOrEmail)
	return scanUser(row)
}

func (s *Store) FindUserByAccount(ctx context.Context, account string) (*User, error) {
	row := s.db.QueryRowContext(ctx, `
	SELECT id, account, password_hash, email, avatar_url, nickname, role, status, token_version, created_at, updated_at
	FROM users WHERE account = ? LIMIT 1`, account)
	return scanUser(row)
}

func (s *Store) FindUserByEmail(ctx context.Context, email string) (*User, error) {
	row := s.db.QueryRowContext(ctx, `
	SELECT id, account, password_hash, email, avatar_url, nickname, role, status, token_version, created_at, updated_at
	FROM users WHERE email = ? LIMIT 1`, email)
	return scanUser(row)
}

func (s *Store) UpdateUserProfile(ctx context.Context, id uint64, in UserUpdate) error {
	res, err := s.db.ExecContext(ctx, `
	UPDATE users SET email = ?, avatar_url = ?, nickname = ? WHERE id = ?`,
		in.Email, in.AvatarURL, in.Nickname, id)
	if err != nil {
		return err
	}
	return s.requireUserUpdateAffected(ctx, id, res)
}

func (s *Store) UpdateUserPassword(ctx context.Context, id uint64, passwordHash string) error {
	res, err := s.db.ExecContext(ctx, "UPDATE users SET password_hash = ? WHERE id = ?", passwordHash, id)
	if err != nil {
		return err
	}
	return s.requireUserUpdateAffected(ctx, id, res)
}

// UpdateUserPasswordAndRevokeTokens 在同一事务中更新密码、递增 Access Token
// 版本并撤销全部 Refresh Token，避免任一步骤成功后留下旧会话窗口。
func (s *Store) UpdateUserPasswordAndRevokeTokens(ctx context.Context, id uint64, passwordHash string) error {
	return WithTx(ctx, s.db, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx, `
UPDATE users SET password_hash = ?, token_version = token_version + 1 WHERE id = ?`, passwordHash, id)
		if err != nil {
			return err
		}
		if err := requireUpdateAffected(ctx, res, func(ctx context.Context) error {
			var exists int
			return tx.QueryRowContext(ctx, "SELECT 1 FROM users WHERE id = ?", id).Scan(&exists)
		}); err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `
UPDATE refresh_tokens SET revoked_at = NOW()
WHERE user_id = ? AND revoked_at IS NULL`, id)
		return err
	})
}

func (s *Store) UserTokenVersion(ctx context.Context, id uint64) (uint64, error) {
	var version uint64
	if err := s.db.QueryRowContext(ctx, "SELECT token_version FROM users WHERE id = ?", id).Scan(&version); err != nil {
		return 0, scanErr(err)
	}
	return version, nil
}

func (s *Store) UpdateUserStatusSafely(ctx context.Context, id, currentID uint64, status string) error {
	return s.updateUserAdminFieldsSafely(ctx, id, currentID, "status", status)
}

func (s *Store) UpdateUserRoleSafely(ctx context.Context, id, currentID uint64, role string) error {
	return s.updateUserAdminFieldsSafely(ctx, id, currentID, "role", role)
}

func (s *Store) updateUserAdminFieldsSafely(ctx context.Context, id, currentID uint64, field, value string) error {
	return WithTx(ctx, s.db, func(tx *sql.Tx) error {
		rows, err := tx.QueryContext(ctx, "SELECT id FROM users WHERE role = 'admin' AND status = 'active' ORDER BY id FOR UPDATE")
		if err != nil {
			return err
		}
		activeAdminCount := 0
		for rows.Next() {
			var adminID uint64
			if err := rows.Scan(&adminID); err != nil {
				rows.Close()
				return err
			}
			activeAdminCount++
		}
		if err := rows.Close(); err != nil {
			return err
		}
		if err := rows.Err(); err != nil {
			return err
		}

		var target User
		if err := tx.QueryRowContext(ctx, "SELECT id, role, status FROM users WHERE id = ? FOR UPDATE", id).
			Scan(&target.ID, &target.Role, &target.Status); err != nil {
			return scanErr(err)
		}

		switch field {
		case "status":
			if target.ID == currentID && value != "active" {
				return ErrCannotDisableSelf
			}
			if target.Role == "admin" && target.Status == "active" && value != "active" && activeAdminCount <= 1 {
				return ErrLastActiveAdmin
			}
		case "role":
			if target.ID == currentID && value != "admin" {
				return ErrCannotDowngradeSelf
			}
			if target.Role == "admin" && target.Status == "active" && value != "admin" && activeAdminCount <= 1 {
				return ErrLastActiveAdmin
			}
		default:
			return errors.New("unsupported user field update")
		}

		query := "UPDATE users SET status = ?, token_version = token_version + 1, updated_at = NOW() WHERE id = ?"
		if field == "role" {
			query = "UPDATE users SET role = ?, token_version = token_version + 1, updated_at = NOW() WHERE id = ?"
		}
		if _, err = tx.ExecContext(ctx, query, value, id); err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `
UPDATE refresh_tokens SET revoked_at = NOW()
WHERE user_id = ? AND revoked_at IS NULL`, id)
		return err
	})
}

func (s *Store) CountActiveAdmins(ctx context.Context) (int64, error) {
	var count int64
	err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM users WHERE role = 'admin' AND status = 'active'").Scan(&count)
	return count, err
}

func (s *Store) ListUsers(ctx context.Context, page, size int) ([]User, int64, error) {
	offset := (page - 1) * size
	var total int64
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM users").Scan(&total); err != nil {
		return nil, 0, err
	}

	// 不查询 password_hash：用户列表仅用于展示，避免密码哈希进入进程内存。
	// 登录/改密等校验走 FindUserByAccount/FindUser 等专用查询。
	rows, err := s.db.QueryContext(ctx, `
	SELECT id, account, email, avatar_url, nickname, role, status, created_at, updated_at
	FROM users ORDER BY id DESC LIMIT ? OFFSET ?`, size, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items := make([]User, 0)
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Account, &u.Email, &u.AvatarURL, &u.Nickname, &u.Role, &u.Status, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, 0, err
		}
		items = append(items, u)
	}
	return items, total, rows.Err()
}

func scanUser(row *sql.Row) (*User, error) {
	var u User
	err := row.Scan(&u.ID, &u.Account, &u.PasswordHash, &u.Email, &u.AvatarURL, &u.Nickname, &u.Role, &u.Status, &u.TokenVersion, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, scanErr(err)
	}
	return &u, nil
}

func (s *Store) requireUserUpdateAffected(ctx context.Context, id uint64, res sql.Result) error {
	return requireUpdateAffected(ctx, res, func(ctx context.Context) error {
		_, err := s.FindUserByID(ctx, id)
		return err
	})
}
