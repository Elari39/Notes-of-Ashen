package model

import "testing"

func TestUserRegistrationLockUsesNamedDatabaseLock(t *testing.T) {
	assertContains(t, userRegistrationLockAcquireSQL, "GET_LOCK")
	assertContains(t, userRegistrationLockReleaseSQL, "RELEASE_LOCK")
	if userRegistrationLockName == "" {
		t.Fatal("userRegistrationLockName is empty")
	}
}
