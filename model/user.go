package model

import (
	"context"
	"database/sql"
	"time"
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

func (s *Store) CreateUser(ctx context.Context, in UserCreate) (uint64, error) {
	res, err := s.db.ExecContext(ctx, `
INSERT INTO users (account, password_hash, email, avatar_url, nickname, role)
VALUES (?, ?, ?, ?, ?, ?)`,
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
	_, err := s.db.ExecContext(ctx, `
UPDATE users SET email = ?, avatar_url = ?, nickname = ? WHERE id = ?`,
		in.Email, in.AvatarURL, in.Nickname, id)
	return err
}

func (s *Store) UpdateUserPassword(ctx context.Context, id uint64, passwordHash string) error {
	_, err := s.db.ExecContext(ctx, "UPDATE users SET password_hash = ? WHERE id = ?", passwordHash, id)
	return err
}

func (s *Store) UpdateUserStatus(ctx context.Context, id uint64, status string) error {
	res, err := s.db.ExecContext(ctx, "UPDATE users SET status = ? WHERE id = ?", status, id)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrNotFound
	}
	return nil
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
