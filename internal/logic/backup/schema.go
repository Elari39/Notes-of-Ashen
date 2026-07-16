package backup

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"

	apperrors "notes-of-ashen/internal/errors"
	"notes-of-ashen/internal/svc"
)

const (
	backupSchemaMigrationRequiredMessage = "database schema migration is required"
	backupSchemaCheckUnavailableMessage  = "backup schema check is unavailable"
)

// EnsureSchemaReady keeps the backup endpoints from reaching destructive backup
// operations when an existing deployment has not yet applied the required
// incremental database migration.
func EnsureSchemaReady(ctx context.Context, svcCtx *svc.ServiceContext) error {
	if svcCtx == nil || svcCtx.Store == nil {
		return apperrors.ServiceUnavailable(backupSchemaCheckUnavailableMessage)
	}

	ready, err := svcCtx.Store.BackupSchemaReady(ctx)
	if err != nil {
		logx.Errorf("check backup schema readiness failed: %v", err)
		return apperrors.ServiceUnavailable(backupSchemaCheckUnavailableMessage)
	}
	if !ready {
		return apperrors.ServiceUnavailable(backupSchemaMigrationRequiredMessage)
	}
	return nil
}
