package backup

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"notes-of-ashen/model"
)

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
