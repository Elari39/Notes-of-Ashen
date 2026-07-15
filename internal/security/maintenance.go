package security

import "sync/atomic"

const RestoreMaintenanceKey = "system:restore:maintenance"
const AccessTokensNotBeforeKey = "auth:access:not-before"

var restoreInProgress atomic.Bool
var accessTokensNotBefore atomic.Int64

func TryStartRestore() bool { return restoreInProgress.CompareAndSwap(false, true) }

func EndRestore() { restoreInProgress.Store(false) }

func RestoreInProgress() bool { return restoreInProgress.Load() }

func SetAccessTokensNotBefore(value int64) { accessTokensNotBefore.Store(value) }

func AccessTokensNotBefore() int64 { return accessTokensNotBefore.Load() }
