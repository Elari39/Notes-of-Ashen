package model

import (
	"context"
	"errors"

	"github.com/go-sql-driver/mysql"
)

const (
	RegistrationEnabledKey = "registration_enabled"
	HomeArticleLayoutKey   = "home_article_layout"

	HomeArticleLayoutStandard    = "standard"
	HomeArticleLayoutAlternating = "alternating"
)

type SiteSettings struct {
	RegistrationEnabled bool
	HomeArticleLayout   string
}

func (s *Store) SiteSettings(ctx context.Context) (*SiteSettings, error) {
	enabled, err := s.GetBoolSetting(ctx, RegistrationEnabledKey, true)
	if err != nil {
		return nil, err
	}
	layout, err := s.GetStringSetting(ctx, HomeArticleLayoutKey, HomeArticleLayoutStandard)
	if err != nil {
		return nil, err
	}
	return &SiteSettings{
		RegistrationEnabled: enabled,
		HomeArticleLayout:   NormalizeHomeArticleLayout(layout),
	}, nil
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

func (s *Store) GetStringSetting(ctx context.Context, key string, defaultValue string) (string, error) {
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
		return "", err
	}
	return raw, nil
}

func (s *Store) UpdateRegistrationEnabled(ctx context.Context, enabled bool) error {
	value := "false"
	if enabled {
		value = "true"
	}
	return s.UpsertSetting(ctx, RegistrationEnabledKey, value)
}

func (s *Store) UpdateSiteSettings(ctx context.Context, settings SiteSettings) error {
	value := "false"
	if settings.RegistrationEnabled {
		value = "true"
	}
	_, err := s.db.ExecContext(ctx, `
INSERT INTO site_settings (setting_key, setting_value)
VALUES (?, ?), (?, ?)
ON DUPLICATE KEY UPDATE setting_value = VALUES(setting_value)`,
		RegistrationEnabledKey, value,
		HomeArticleLayoutKey, NormalizeHomeArticleLayout(settings.HomeArticleLayout))
	return err
}

func (s *Store) UpdateHomeArticleLayout(ctx context.Context, layout string) error {
	return s.UpsertSetting(ctx, HomeArticleLayoutKey, NormalizeHomeArticleLayout(layout))
}

func (s *Store) UpsertSetting(ctx context.Context, key, value string) error {
	_, err := s.db.ExecContext(ctx, `
INSERT INTO site_settings (setting_key, setting_value)
VALUES (?, ?)
ON DUPLICATE KEY UPDATE setting_value = VALUES(setting_value)`,
		key, value)
	return err
}

func NormalizeHomeArticleLayout(layout string) string {
	if layout == HomeArticleLayoutAlternating {
		return HomeArticleLayoutAlternating
	}
	return HomeArticleLayoutStandard
}
