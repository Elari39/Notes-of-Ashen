package model

import (
	"context"
	"errors"

	"github.com/go-sql-driver/mysql"
)

const RegistrationEnabledKey = "registration_enabled"

type SiteSettings struct {
	RegistrationEnabled bool
}

func (s *Store) SiteSettings(ctx context.Context) (*SiteSettings, error) {
	enabled, err := s.GetBoolSetting(ctx, RegistrationEnabledKey, true)
	if err != nil {
		return nil, err
	}
	return &SiteSettings{RegistrationEnabled: enabled}, nil
}

func (s *Store) GetBoolSetting(ctx context.Context, key string, defaultValue bool) (bool, error) {
	var raw string
	err := s.db.QueryRowContext(ctx, "SELECT setting_value FROM site_settings WHERE setting_key = ? LIMIT 1", key).Scan(&raw)
	if err != nil {
		if errors.Is(scanErr(err), ErrNotFound) {
			return defaultValue, nil
		}
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1146 {
			return defaultValue, nil
		}
		return false, err
	}
	return raw == "true" || raw == "1", nil
}

func (s *Store) UpdateRegistrationEnabled(ctx context.Context, enabled bool) error {
	value := "false"
	if enabled {
		value = "true"
	}
	_, err := s.db.ExecContext(ctx, `
INSERT INTO site_settings (setting_key, setting_value)
VALUES (?, ?)
ON DUPLICATE KEY UPDATE setting_value = VALUES(setting_value)`,
		RegistrationEnabledKey, value)
	return err
}
