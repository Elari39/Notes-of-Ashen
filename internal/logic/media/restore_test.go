package media

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func TestRestoreTransactionPublishesFilesInsideMediaVolume(t *testing.T) {
	root := filepath.Join(t.TempDir(), "media")
	if err := os.MkdirAll(root, 0755); err != nil {
		t.Fatal(err)
	}
	oldData := []byte("old media")
	oldKey := restoreTestKey(oldData)
	if err := os.WriteFile(filepath.Join(root, oldKey), oldData, 0644); err != nil {
		t.Fatal(err)
	}

	tx, err := BeginRestore(root, "restore-1")
	if err != nil {
		t.Fatalf("BeginRestore() error = %v", err)
	}
	assertPathInsideRoot(t, root, tx.journal.Staging)
	assertPathInsideRoot(t, root, tx.journal.Rollback)
	data := []byte("new media")
	key := restoreTestKey(data)
	stageAndSeal(t, tx, key, data)
	if _, err := os.Stat(filepath.Join(root, key)); !os.IsNotExist(err) {
		t.Fatalf("staged media appeared in current root, stat error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(tx.journal.Staging, key)); err != nil {
		t.Fatalf("staged media missing: %v", err)
	}

	if err := tx.Publish(); err != nil {
		t.Fatalf("Publish() error = %v", err)
	}
	assertFileContent(t, filepath.Join(root, key), data)
	if _, err := os.Stat(filepath.Join(root, oldKey)); !os.IsNotExist(err) {
		t.Fatalf("old media remained in current root, stat error = %v", err)
	}
	assertFileContent(t, filepath.Join(tx.journal.Rollback, oldKey), oldData)

	if err := tx.Finalize(); err != nil {
		t.Fatalf("Finalize() error = %v", err)
	}
	if _, err := os.Stat(tx.journal.Rollback); !os.IsNotExist(err) {
		t.Fatalf("rollback directory remains, stat error = %v", err)
	}
	if _, err := os.Stat(restoreJournalPath(root)); !os.IsNotExist(err) {
		t.Fatalf("restore journal remains, stat error = %v", err)
	}
}

func TestRecoverRestoreRollsBackUncommittedStaging(t *testing.T) {
	root := filepath.Join(t.TempDir(), "media")
	if err := os.MkdirAll(root, 0755); err != nil {
		t.Fatal(err)
	}
	oldData := []byte("old media")
	oldKey := restoreTestKey(oldData)
	if err := os.WriteFile(filepath.Join(root, oldKey), oldData, 0644); err != nil {
		t.Fatal(err)
	}
	tx, err := BeginRestore(root, "restore-2")
	if err != nil {
		t.Fatalf("BeginRestore() error = %v", err)
	}
	data := []byte("new media")
	if err := tx.RestoreReader(restoreTestKey(data), bytes.NewReader(data), int64(len(data))); err != nil {
		t.Fatalf("RestoreReader() error = %v", err)
	}

	recovered, err := RecoverRestore(root, "")
	if err != nil {
		t.Fatalf("RecoverRestore() error = %v", err)
	}
	if recovered != nil {
		t.Fatal("RecoverRestore() returned a transaction for uncommitted staging")
	}
	assertFileContent(t, filepath.Join(root, oldKey), oldData)
	if _, err := os.Stat(tx.journal.Staging); !os.IsNotExist(err) {
		t.Fatalf("staging directory remains, stat error = %v", err)
	}
	if _, err := os.Stat(tx.journal.Rollback); !os.IsNotExist(err) {
		t.Fatalf("rollback directory remains, stat error = %v", err)
	}
	if _, err := os.Stat(restoreJournalPath(root)); !os.IsNotExist(err) {
		t.Fatalf("restore journal remains, stat error = %v", err)
	}
}

func TestRestoreTransactionPublishesEmptyMediaSet(t *testing.T) {
	root := filepath.Join(t.TempDir(), "media")
	if err := os.MkdirAll(root, 0755); err != nil {
		t.Fatal(err)
	}
	oldData := []byte("old media")
	oldKey := restoreTestKey(oldData)
	if err := os.WriteFile(filepath.Join(root, oldKey), oldData, 0644); err != nil {
		t.Fatal(err)
	}
	tx, err := BeginRestore(root, "restore-empty")
	if err != nil {
		t.Fatalf("BeginRestore() error = %v", err)
	}
	if err := tx.Seal([]string{}); err != nil {
		t.Fatalf("Seal() error = %v", err)
	}
	if err := tx.Publish(); err != nil {
		t.Fatalf("Publish() error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, oldKey)); !os.IsNotExist(err) {
		t.Fatalf("old media remained after empty restore, stat error = %v", err)
	}
	if err := tx.Finalize(); err != nil {
		t.Fatalf("Finalize() error = %v", err)
	}
}

func TestRecoverRestoreCompletesCommittedFilePublication(t *testing.T) {
	root := filepath.Join(t.TempDir(), "media")
	if err := os.MkdirAll(root, 0755); err != nil {
		t.Fatal(err)
	}
	oldData := []byte("old media")
	oldKey := restoreTestKey(oldData)
	if err := os.WriteFile(filepath.Join(root, oldKey), oldData, 0644); err != nil {
		t.Fatal(err)
	}
	tx, err := BeginRestore(root, "restore-3")
	if err != nil {
		t.Fatalf("BeginRestore() error = %v", err)
	}
	data := []byte("new media")
	key := restoreTestKey(data)
	stageAndSeal(t, tx, key, data)

	// Simulate a crash after the old media is moved to rollback but before the
	// staged file is renamed into the public media root.
	journal := tx.journal
	journal.State = restoreStatePublishingOld
	if err := writeRestoreJournal(journal); err != nil {
		t.Fatalf("writeRestoreJournal() error = %v", err)
	}
	if err := moveCurrentMediaToRollback(journal); err != nil {
		t.Fatalf("moveCurrentMediaToRollback() error = %v", err)
	}
	journal.State = restoreStatePublishingNew
	if err := writeRestoreJournal(journal); err != nil {
		t.Fatalf("writeRestoreJournal() error = %v", err)
	}

	recovered, err := RecoverRestore(root, journal.ID)
	if err != nil {
		t.Fatalf("RecoverRestore() error = %v", err)
	}
	if recovered == nil {
		t.Fatal("RecoverRestore() returned nil for committed restore")
	}
	assertFileContent(t, filepath.Join(root, key), data)
	if _, err := os.Stat(filepath.Join(root, oldKey)); !os.IsNotExist(err) {
		t.Fatalf("old media remained after recovery, stat error = %v", err)
	}
	if err := recovered.Finalize(); err != nil {
		t.Fatalf("Finalize() error = %v", err)
	}
}

func TestRecoverRestoreReversesUnmarkedPartialPublication(t *testing.T) {
	root := filepath.Join(t.TempDir(), "media")
	if err := os.MkdirAll(root, 0755); err != nil {
		t.Fatal(err)
	}
	oldData := []byte("old media")
	oldKey := restoreTestKey(oldData)
	if err := os.WriteFile(filepath.Join(root, oldKey), oldData, 0644); err != nil {
		t.Fatal(err)
	}
	tx, err := BeginRestore(root, "restore-4")
	if err != nil {
		t.Fatalf("BeginRestore() error = %v", err)
	}
	data := []byte("new media")
	key := restoreTestKey(data)
	stageAndSeal(t, tx, key, data)

	journal := tx.journal
	journal.State = restoreStatePublishingOld
	if err := writeRestoreJournal(journal); err != nil {
		t.Fatal(err)
	}
	if err := moveCurrentMediaToRollback(journal); err != nil {
		t.Fatal(err)
	}
	journal.State = restoreStatePublishingNew
	if err := writeRestoreJournal(journal); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(filepath.Join(journal.Staging, key), filepath.Join(root, key)); err != nil {
		t.Fatalf("publish one staged media file: %v", err)
	}

	recovered, err := RecoverRestore(root, "")
	if err != nil {
		t.Fatalf("RecoverRestore() error = %v", err)
	}
	if recovered != nil {
		t.Fatal("RecoverRestore() returned a transaction for an unmarked restore")
	}
	assertFileContent(t, filepath.Join(root, oldKey), oldData)
	if _, err := os.Stat(filepath.Join(root, key)); !os.IsNotExist(err) {
		t.Fatalf("partially published media remains, stat error = %v", err)
	}
	if _, err := os.Stat(restoreJournalPath(root)); !os.IsNotExist(err) {
		t.Fatalf("restore journal remains, stat error = %v", err)
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != oldKey {
		t.Fatalf("root entries after rollback = %#v, want only %s", entries, oldKey)
	}
}

func TestRecoverRestoreRejectsCommittedMarkerWithoutJournal(t *testing.T) {
	root := filepath.Join(t.TempDir(), "media")
	if _, err := RecoverRestore(root, "restore-5"); err == nil {
		t.Fatal("RecoverRestore() error = nil, want missing journal error")
	}
}

func stageAndSeal(t *testing.T, tx *RestoreTransaction, key string, data []byte) {
	t.Helper()
	if err := tx.RestoreReader(key, bytes.NewReader(data), int64(len(data))); err != nil {
		t.Fatalf("RestoreReader() error = %v", err)
	}
	if err := tx.Seal([]string{key}); err != nil {
		t.Fatalf("Seal() error = %v", err)
	}
}

func assertFileContent(t *testing.T, path string, want []byte) {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("file %s = %q, want %q", path, got, want)
	}
}

func assertPathInsideRoot(t *testing.T, root, path string) {
	t.Helper()
	rel, err := filepath.Rel(root, path)
	if err != nil {
		t.Fatal(err)
	}
	if rel == ".." || (len(rel) > 3 && rel[:3] == ".."+string(filepath.Separator)) {
		t.Fatalf("path %s is not inside root %s", path, root)
	}
}

func restoreTestKey(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]) + ".png"
}
