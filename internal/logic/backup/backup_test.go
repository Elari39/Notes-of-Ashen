package backup

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"

	"notes-of-ashen/internal/config"
	medialogic "notes-of-ashen/internal/logic/media"
	"notes-of-ashen/internal/svc"
	"notes-of-ashen/model"
)

type fakeCacheRedis struct {
	patterns []string
	deleted  [][]string
	scanErr  error
	delErr   error
}

func (f *fakeCacheRedis) Scan(_ context.Context, _ uint64, match string, _ int64) *redis.ScanCmd {
	f.patterns = append(f.patterns, match)
	if f.scanErr != nil {
		return redis.NewScanCmdResult(nil, 0, f.scanErr)
	}
	return redis.NewScanCmdResult([]string{"cache:" + match}, 0, nil)
}

func (f *fakeCacheRedis) Del(_ context.Context, keys ...string) *redis.IntCmd {
	f.deleted = append(f.deleted, append([]string(nil), keys...))
	return redis.NewIntResult(int64(len(keys)), f.delErr)
}

func TestValidateSnapshotRequiresActiveAdminAndValidRelations(t *testing.T) {
	snapshot, manifest := minimalBackup(t)
	if err := validateSnapshot(&snapshot, &manifest); err != nil {
		t.Fatalf("validateSnapshot() error = %v", err)
	}

	missingAdmin := snapshot
	missingAdmin.Users = append([]model.User(nil), snapshot.Users...)
	missingAdmin.Users[0].Role = "editor"
	if err := validateSnapshot(&missingAdmin, &manifest); err == nil {
		t.Fatal("validateSnapshot() missing admin error = nil")
	}

	badRelation := snapshot
	badRelation.Categories = []model.Category{{ID: 1, Name: "Go", Slug: "go", CreatedBy: 999}}
	badManifest := manifest
	badManifest.Counts = backupCounts(badRelation)
	if err := validateSnapshot(&badRelation, &badManifest); err == nil {
		t.Fatal("validateSnapshot() invalid creator error = nil")
	}
}

func TestReadArchiveRejectsFutureVersionAndUnexpectedFiles(t *testing.T) {
	snapshot, manifest := minimalBackup(t)
	valid := buildZip(t, snapshot, manifest, nil)
	reader, err := zip.NewReader(bytes.NewReader(valid), int64(len(valid)))
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := readArchive(reader, 1<<20); err != nil {
		t.Fatalf("readArchive(valid) error = %v", err)
	}

	future := manifest
	future.Version = archiveVersion + 1
	futureData := buildZip(t, snapshot, future, nil)
	futureReader, _ := zip.NewReader(bytes.NewReader(futureData), int64(len(futureData)))
	if _, _, err := readArchive(futureReader, 1<<20); err == nil {
		t.Fatal("readArchive(future) error = nil")
	}

	extraData := buildZip(t, snapshot, manifest, map[string][]byte{"extra.txt": []byte("unexpected")})
	extraReader, _ := zip.NewReader(bytes.NewReader(extraData), int64(len(extraData)))
	if _, _, err := readArchive(extraReader, 1<<20); err == nil {
		t.Fatal("readArchive(extra file) error = nil")
	}
}

func TestReadArchiveCountsManifestTowardExpandedSizeLimit(t *testing.T) {
	snapshot, manifest := minimalBackup(t)
	raw := buildZip(t, snapshot, manifest, nil)
	reader, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		t.Fatal(err)
	}
	var expanded int64
	for _, file := range reader.File {
		expanded += int64(file.UncompressedSize64)
	}
	if _, _, err := readArchive(reader, expanded-1); err == nil {
		t.Fatal("readArchive() accepted archive when manifest pushed expanded content over the limit")
	}
}

func TestBackupSizeLimitIncludesManifestAndFinalArchive(t *testing.T) {
	const limit int64 = 100
	if exceedsBackupSizeLimit(80, 20, limit) {
		t.Fatal("exactly limited data plus manifest should be accepted")
	}
	if !exceedsBackupSizeLimit(80, 21, limit) {
		t.Fatal("manifest bytes beyond the limit should be rejected")
	}
	if !exceedsBackupSizeLimit(0, 101, limit) {
		t.Fatal("final encrypted archive beyond the upload limit should be rejected")
	}
}

func TestExportArchiveLimitsMatchRestoreLimits(t *testing.T) {
	if err := validateExportArchiveLimits(maxBackupArchiveEntries-2, maxBackupManifestBytes); err != nil {
		t.Fatalf("validateExportArchiveLimits() exact boundary error = %v", err)
	}
	if err := validateExportArchiveLimits(maxBackupArchiveEntries-1, 0); err == nil {
		t.Fatal("validateExportArchiveLimits() accepted media count that exceeds the ZIP entry limit")
	}
	if err := validateExportArchiveLimits(0, maxBackupManifestBytes+1); err == nil {
		t.Fatal("validateExportArchiveLimits() accepted manifest that exceeds the restore limit")
	}
}

func TestValidateSnapshotRejectsInternalRestoreMarker(t *testing.T) {
	snapshot, manifest := minimalBackup(t)
	snapshot.Settings = []model.BackupSetting{{Key: model.BackupRestoreMarkerKey, Value: "restore-transaction"}}
	manifest.Counts = backupCounts(snapshot)
	if err := validateSnapshot(&snapshot, &manifest); err == nil {
		t.Fatal("validateSnapshot() accepted internal restore marker setting")
	}
}

func TestRecoverPendingRestorePublishesCommittedMedia(t *testing.T) {
	root := filepath.Join(t.TempDir(), "media")
	if err := os.MkdirAll(root, 0755); err != nil {
		t.Fatal(err)
	}
	oldData := []byte("old media")
	oldSum := sha256.Sum256(oldData)
	oldKey := hex.EncodeToString(oldSum[:]) + ".png"
	if err := os.WriteFile(filepath.Join(root, oldKey), oldData, 0644); err != nil {
		t.Fatal(err)
	}
	const restoreID = "restore-transaction"
	mediaRestore, err := medialogic.BeginRestore(root, restoreID)
	if err != nil {
		t.Fatalf("BeginRestore() error = %v", err)
	}
	data := []byte("restored media")
	sum := sha256.Sum256(data)
	key := hex.EncodeToString(sum[:]) + ".png"
	if err := mediaRestore.RestoreReader(key, bytes.NewReader(data), int64(len(data))); err != nil {
		t.Fatalf("RestoreReader() error = %v", err)
	}
	if err := mediaRestore.Seal([]string{key}); err != nil {
		t.Fatalf("Seal() error = %v", err)
	}

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	defer db.Close()
	mock.ExpectQuery(`SELECT setting_value FROM site_settings WHERE setting_key = \? LIMIT 1`).
		WithArgs(model.BackupRestoreMarkerKey).
		WillReturnRows(sqlmock.NewRows([]string{"setting_value"}).AddRow(restoreID))
	mock.ExpectExec(`DELETE FROM site_settings WHERE setting_key = \? AND setting_value = \?`).
		WithArgs(model.BackupRestoreMarkerKey, restoreID).
		WillReturnResult(sqlmock.NewResult(0, 1))
	service := &svc.ServiceContext{
		Config: config.Config{Media: config.MediaConf{RootDir: root}},
		Store:  model.NewStore(db),
	}
	if err := RecoverPendingRestore(context.Background(), service); err != nil {
		t.Fatalf("RecoverPendingRestore() error = %v", err)
	}
	got, err := os.ReadFile(filepath.Join(root, key))
	if err != nil {
		t.Fatalf("read recovered media: %v", err)
	}
	if !bytes.Equal(got, data) {
		t.Fatalf("recovered media = %q, want %q", got, data)
	}
	if _, err := os.Stat(filepath.Join(root, oldKey)); !os.IsNotExist(err) {
		t.Fatalf("old media remained after recovery, stat error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestReadArchiveRejectsMediaChecksumMismatch(t *testing.T) {
	snapshot, manifest := minimalBackup(t)
	raw := []byte("media")
	sum := sha256.Sum256(raw)
	key := hex.EncodeToString(sum[:]) + ".png"
	snapshot.MediaAssets = []model.MediaAsset{{ID: 1, StorageKey: key, OriginalName: "image.png", MIMEType: "image/png", SizeBytes: uint64(len(raw)), SHA256: hex.EncodeToString(sum[:]), CreatedBy: 1}}
	manifest.Counts = backupCounts(snapshot)
	manifest.Files = []ManifestFile{{Path: "media/" + key, SHA256: hex.EncodeToString(sum[:]), Size: int64(len(raw))}}
	archive := buildZip(t, snapshot, manifest, map[string][]byte{"media/" + key: []byte("tampered")})
	reader, _ := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
	if _, _, err := readArchive(reader, 1<<20); err == nil {
		t.Fatal("readArchive(tampered media) error = nil")
	}
}

func TestClearApplicationCacheClearsVerificationCodeKeys(t *testing.T) {
	client := &fakeCacheRedis{}
	if err := clearApplicationCache(context.Background(), client); err != nil {
		t.Fatalf("clearApplicationCache() error = %v", err)
	}
	patterns := make(map[string]bool, len(client.patterns))
	for _, pattern := range client.patterns {
		patterns[pattern] = true
	}
	if !patterns["verify_code:*"] || !patterns["verify_code_cooldown:*"] {
		t.Fatalf("verification code patterns missing: %v", client.patterns)
	}
	if len(client.deleted) != len(client.patterns) {
		t.Fatalf("deleted batches = %d, want %d", len(client.deleted), len(client.patterns))
	}
}

func TestClearApplicationCacheReturnsRedisErrors(t *testing.T) {
	scanErr := errors.New("scan failed")
	if err := clearApplicationCache(context.Background(), &fakeCacheRedis{scanErr: scanErr}); !errors.Is(err, scanErr) {
		t.Fatalf("clearApplicationCache() error = %v, want %v", err, scanErr)
	}
	delErr := errors.New("delete failed")
	if err := clearApplicationCache(context.Background(), &fakeCacheRedis{delErr: delErr}); !errors.Is(err, delErr) {
		t.Fatalf("clearApplicationCache() error = %v, want %v", err, delErr)
	}
}

func minimalBackup(t *testing.T) (model.BackupSnapshot, Manifest) {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	snapshot := model.BackupSnapshot{Users: []model.User{{ID: 1, Account: "admin", PasswordHash: string(hash), Email: "admin@example.test", Role: "admin", Status: "active"}}}
	manifest := Manifest{Version: archiveVersion, ExportedAt: time.Now().UTC(), DataSHA256: "0000000000000000000000000000000000000000000000000000000000000000", Counts: backupCounts(snapshot), Files: []ManifestFile{}}
	return snapshot, manifest
}

func backupCounts(snapshot model.BackupSnapshot) map[string]int {
	return map[string]int{
		"users": len(snapshot.Users), "settings": len(snapshot.Settings), "categories": len(snapshot.Categories),
		"tags": len(snapshot.Tags), "projects": len(snapshot.Projects), "articles": len(snapshot.Articles),
		"versions": len(snapshot.Versions), "media": len(snapshot.MediaAssets),
	}
}

func buildZip(t *testing.T, snapshot model.BackupSnapshot, manifest Manifest, files map[string][]byte) []byte {
	t.Helper()
	data, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(data)
	manifest.DataSHA256 = hex.EncodeToString(sum[:])
	manifestData, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	var buffer bytes.Buffer
	archive := zip.NewWriter(&buffer)
	if err := writeZipBytes(archive, "manifest.json", manifestData); err != nil {
		t.Fatal(err)
	}
	if err := writeZipBytes(archive, "data.json", data); err != nil {
		t.Fatal(err)
	}
	for name, content := range files {
		if err := writeZipBytes(archive, name, content); err != nil {
			t.Fatal(err)
		}
	}
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}
