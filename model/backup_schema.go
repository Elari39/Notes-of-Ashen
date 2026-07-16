package model

import (
	"context"
	"errors"
)

var backupSchemaRequiredTables = [...]string{
	"media_assets",
	"traffic_content_daily_stats",
	"traffic_content_daily_visitors",
}

const backupSchemaReadyQuery = `SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name IN (?, ?, ?)`

// BackupSchemaReady reports whether every table required by encrypted backup and
// restore is available in the current database. It deliberately uses DATABASE()
// so deployments do not need to duplicate the configured schema name in code.
func (s *Store) BackupSchemaReady(ctx context.Context) (bool, error) {
	if s == nil || s.db == nil {
		return false, errors.New("database is not configured")
	}

	var count int
	if err := s.db.QueryRowContext(ctx, backupSchemaReadyQuery, backupSchemaRequiredTables[0], backupSchemaRequiredTables[1], backupSchemaRequiredTables[2]).Scan(&count); err != nil {
		return false, err
	}
	return count == len(backupSchemaRequiredTables), nil
}
