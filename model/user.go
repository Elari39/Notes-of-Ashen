package model

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
)

var errRegistrationLockNotAcquired = errors.New("user registration lock not acquired")

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

func (s *Store) WithUserRegistrationLock(ctx context.Context, fn func(context.Context) error) error {
	// GET_LOCK 返回 1 表示加锁成功，0 表示超时，NULL 表示出错（如线程被杀死）。
	// 必须校验返回值，否则超时/失败会被当作加锁成功，并发注册可能让多个用户都成为 admin。
	var acquired sql.NullInt64
	if err := s.db.QueryRowContext(ctx, userRegistrationLockAcquireSQL, userRegistrationLockName).Scan(&acquired); err != nil {
		return err
	}
	if !acquired.Valid || acquired.Int64 != 1 {
		return errRegistrationLockNotAcquired
	}
	defer func() {
		var released sql.NullInt64
		if err := s.db.QueryRowContext(context.Background(), userRegistrationLockReleaseSQL, userRegistrationLockName).Scan(&released); err != nil {
			logx.Errorf("release user registration lock failed: %v", err)
		} else if !released.Valid || released.Int64 != 1 {
			logx.Errorf("release user registration lock returned non-one: %v", released)
		}
	}()
	return fn(ctx)
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
	SELECT id, account, password_hash, email, avatar_url, nickname, role, status, created_at, updated_at
	FROM users WHERE id = ?`, id)
	return scanUser(row)
}

func (s *Store) FindUserByAccountOrEmail(ctx context.Context, accountOrEmail string) (*User, error) {
	row := s.db.QueryRowContext(ctx, `
	SELECT id, account, password_hash, email, avatar_url, nickname, role, status, created_at, updated_at
	FROM users WHERE account = ? OR email = ? LIMIT 1`, accountOrEmail, accountOrEmail)
	return scanUser(row)
}

func (s *Store) FindUserByAccount(ctx context.Context, account string) (*User, error) {
	row := s.db.QueryRowContext(ctx, `
	SELECT id, account, password_hash, email, avatar_url, nickname, role, status, created_at, updated_at
	FROM users WHERE account = ? LIMIT 1`, account)
	return scanUser(row)
}

func (s *Store) FindUserByEmail(ctx context.Context, email string) (*User, error) {
	row := s.db.QueryRowContext(ctx, `
	SELECT id, account, password_hash, email, avatar_url, nickname, role, status, created_at, updated_at
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

func (s *Store) UpdateUserStatus(ctx context.Context, id uint64, status string) error {
	res, err := s.db.ExecContext(ctx, "UPDATE users SET status = ?, updated_at = NOW() WHERE id = ?", status, id)
	if err != nil {
		return err
	}
	return s.requireUserUpdateAffected(ctx, id, res)
}

func (s *Store) UpdateUserRole(ctx context.Context, id uint64, role string) error {
	res, err := s.db.ExecContext(ctx, "UPDATE users SET role = ?, updated_at = NOW() WHERE id = ?", role, id)
	if err != nil {
		return err
	}
	return s.requireUserUpdateAffected(ctx, id, res)
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

	rows, err := s.db.QueryContext(ctx, `
	SELECT id, account, password_hash, email, avatar_url, nickname, role, status, created_at, updated_at
	FROM users ORDER BY id DESC LIMIT ? OFFSET ?`, size, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items := make([]User, 0)
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Account, &u.PasswordHash, &u.Email, &u.AvatarURL, &u.Nickname, &u.Role, &u.Status, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, 0, err
		}
		items = append(items, u)
	}
	return items, total, rows.Err()
}

func scanUser(row *sql.Row) (*User, error) {
	var u User
	err := row.Scan(&u.ID, &u.Account, &u.PasswordHash, &u.Email, &u.AvatarURL, &u.Nickname, &u.Role, &u.Status, &u.CreatedAt, &u.UpdatedAt)
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
