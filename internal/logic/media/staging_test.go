package media

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"notes-of-ashen/internal/authutil"
	"notes-of-ashen/internal/config"
	"notes-of-ashen/internal/svc"
	"notes-of-ashen/model"
)

func TestMediaOperationLockSurvivesContentionAndReleasesAfterProcessExit(t *testing.T) {
	if root := os.Getenv("NOA_MEDIA_LOCK_TEST_ROOT"); root != "" {
		unlock, err := lockMediaOperations(t.Context(), root)
		if err != nil {
			t.Fatal(err)
		}
		defer unlock()
		fmt.Println("LOCKED")
		_, _ = bufio.NewReader(os.Stdin).ReadString('\n')
		return
	}
	root := t.TempDir()
	cmd := exec.CommandContext(t.Context(), os.Args[0], "-test.run=^TestMediaOperationLockSurvivesContentionAndReleasesAfterProcessExit$")
	cmd.Env = append(os.Environ(), "NOA_MEDIA_LOCK_TEST_ROOT="+root)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	defer stdin.Close()
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cmd.Process.Kill() })
	ready := make(chan string, 1)
	go func() {
		line, _ := bufio.NewReader(stdout).ReadString('\n')
		ready <- line
	}()
	select {
	case line := <-ready:
		if line != "LOCKED\n" {
			t.Fatalf("child did not acquire media lock: %q", line)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("child media lock readiness timed out")
	}
	ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
	defer cancel()
	if unlock, err := lockMediaOperations(ctx, root); !errors.Is(err, context.DeadlineExceeded) {
		if unlock != nil {
			unlock()
		}
		t.Fatalf("competing lock error = %v, want deadline exceeded", err)
	}
	if err := cmd.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Wait(); err == nil {
		t.Fatal("killed child unexpectedly exited successfully")
	}
	ctx, cancel = context.WithTimeout(t.Context(), time.Second)
	defer cancel()
	unlock, err := lockMediaOperations(ctx, root)
	if err != nil {
		t.Fatalf("lock after process death: %v", err)
	}
	unlock()
}

func TestUploadCannotRecoverAnotherActiveTransaction(t *testing.T) {
	root := t.TempDir()
	unlock, err := lockMediaOperations(t.Context(), root)
	if err != nil {
		t.Fatal(err)
	}
	defer unlock()
	data := testPNG(t)
	staged, err := stageUpload(root, restoreTestKey(data), data)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(authutil.WithUser(t.Context(), 1, authutil.RoleAdmin), 50*time.Millisecond)
	defer cancel()
	service := &svc.ServiceContext{Config: config.Config{Media: config.MediaConf{RootDir: root}}}
	if _, err := Upload(ctx, service, "other.png", "", data); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("competing upload error = %v, want deadline exceeded", err)
	}
	assertFileContent(t, staged, data)
}

func TestRecoverMediaStagingAfterInterruptedTransactions(t *testing.T) {
	for _, tc := range []struct {
		name, kind                             string
		exists, dbFailure, published, retained bool
	}{
		{name: "upload before metadata", kind: "upload"},
		{name: "upload after metadata", kind: "upload", exists: true, published: true},
		{name: "upload database unavailable", kind: "upload", dbFailure: true, retained: true},
		{name: "delete before metadata", kind: "delete", exists: true, published: true},
		{name: "delete after metadata", kind: "delete"},
		{name: "delete database unavailable", kind: "delete", dbFailure: true, retained: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			unlock, err := lockMediaOperations(t.Context(), root)
			if err != nil {
				t.Fatal(err)
			}
			defer unlock()
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			data := testPNG(t)
			key := restoreTestKey(data)
			name := ".upload-" + key + "-interrupted"
			if tc.kind == "delete" {
				name = ".delete-7-" + key
			}
			staged := filepath.Join(root, name)
			if err := os.WriteFile(staged, data, 0o644); err != nil {
				t.Fatal(err)
			}
			query := mock.ExpectQuery("FROM media_assets WHERE storage_key = ").WithArgs(key)
			if tc.dbFailure {
				query.WillReturnError(errors.New("database unavailable"))
			} else {
				rows := mediaTestRows()
				if tc.exists {
					rows.AddRow(7, key, "image.png", "image/png", len(data), 2, 2, "", key[:64], 1, time.Now(), time.Now())
				}
				query.WillReturnRows(rows)
			}
			recoverMediaStaging(t.Context(), &svc.ServiceContext{Store: model.NewStore(db)}, root)
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
			if tc.published {
				assertFileContent(t, filepath.Join(root, key), data)
			} else if _, err := os.Stat(filepath.Join(root, key)); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("unexpected public file: %v", err)
			}
			if tc.retained {
				assertFileContent(t, staged, data)
			} else if _, err := os.Stat(staged); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("staged file remained: %v", err)
			}
		})
	}
}

func TestUploadMetadataFailureCleansStagingAndReleasesLock(t *testing.T) {
	root := t.TempDir()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mock.ExpectQuery("FROM media_assets WHERE sha256 = ").WillReturnRows(mediaTestRows())
	want := errors.New("metadata insert failed")
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO media_assets")).WillReturnError(want)
	service := &svc.ServiceContext{Store: model.NewStore(db), Config: config.Config{Media: config.MediaConf{RootDir: root}}}
	ctx := authutil.WithUser(t.Context(), 1, authutil.RoleAdmin)
	if _, err := Upload(ctx, service, "image.png", "", testPNG(t)); !errors.Is(err, want) {
		t.Fatalf("Upload error = %v, want %v", err, want)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != ".operations.lock" {
		t.Fatalf("unexpected files after rollback: %v", entries)
	}
	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()
	unlock, err := lockMediaOperations(ctx, root)
	if err != nil {
		t.Fatal(err)
	}
	unlock()
}

func mediaTestRows() *sqlmock.Rows {
	return sqlmock.NewRows([]string{"id", "storage_key", "original_name", "mime_type", "size_bytes", "width", "height", "alt_text", "sha256", "created_by", "created_at", "updated_at"})
}
