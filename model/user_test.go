package model

import "testing"

func TestUserRegistrationLockUsesNamedDatabaseLock(t *testing.T) {
	assertContains(t, userRegistrationLockAcquireSQL, "GET_LOCK")
	assertContains(t, userRegistrationLockReleaseSQL, "RELEASE_LOCK")
	if userRegistrationLockName == "" {
		t.Fatal("userRegistrationLockName is empty")
	}
}

func TestErrRegistrationLockNotAcquiredIsSentinel(t *testing.T) {
	// P0-1: GET_LOCK 返回 0/NULL 必须被识别为加锁失败，
	// 业务层据此中止首个 admin 注册逻辑，避免并发多 admin。
	if errRegistrationLockNotAcquired == nil {
		t.Fatal("errRegistrationLockNotAcquired sentinel must not be nil")
	}
	if errRegistrationLockNotAcquired.Error() == "" {
		t.Fatal("errRegistrationLockNotAcquired must carry a message")
	}
}
