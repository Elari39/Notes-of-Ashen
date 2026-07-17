package media

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const (
	restoreJournalVersion       = 2
	restoreStateStaging         = "staging"
	restoreStateStaged          = "staged"
	restoreStatePublishingOld   = "publishing_old"
	restoreStatePublishingNew   = "publishing_new"
	restoreStatePublished       = "published"
	restoreStagingPrefix        = ".restore-staging-"
	restoreRollbackPrefix       = ".restore-rollback-"
	restoreJournalFilename      = ".restore-journal.json"
	restoreDirectoryPermissions = 0700
)

var restoreIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)

// RestoreTransaction keeps a backup restore's media files isolated until its
// database transaction has committed. Staging and rollback directories live
// inside the media root, so every file publication and rollback rename remains
// on the media volume even when the root itself is a separate Docker mount.
type RestoreTransaction struct {
	journal restoreJournal
}

type restoreJournal struct {
	Version  int      `json:"version"`
	ID       string   `json:"id"`
	Root     string   `json:"root"`
	Staging  string   `json:"staging"`
	Rollback string   `json:"rollback"`
	State    string   `json:"state"`
	Files    []string `json:"files"`
}

func BeginRestore(root string, id string) (*RestoreTransaction, error) {
	if !restoreIDPattern.MatchString(id) {
		return nil, fmt.Errorf("invalid media restore id")
	}
	resolvedRoot, err := mediaRootPath(root)
	if err != nil {
		return nil, err
	}
	root, err = ensureMediaRoot(resolvedRoot)
	if err != nil {
		return nil, err
	}
	info, err := os.Lstat(root)
	if err != nil {
		return nil, err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return nil, fmt.Errorf("media root must be a directory for restore")
	}
	journal, err := newRestoreJournal(root, id)
	if err != nil {
		return nil, err
	}
	if _, err := os.Lstat(restoreJournalPath(root)); err == nil {
		return nil, fmt.Errorf("media restore journal already exists")
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	if err := ensureRestorePathAbsent(journal.Staging); err != nil {
		return nil, err
	}
	if err := ensureRestorePathAbsent(journal.Rollback); err != nil {
		return nil, err
	}
	// Persist intent before creating either internal directory. A crash at any
	// later point leaves a journal that startup can use to remove both paths.
	if err := writeRestoreJournal(journal); err != nil {
		return nil, fmt.Errorf("write media restore journal: %w", err)
	}
	if err := createRestoreDirectory(journal.Staging); err != nil {
		return nil, beginRestoreCleanup(journal, fmt.Errorf("create media restore staging directory: %w", err))
	}
	if err := createRestoreDirectory(journal.Rollback); err != nil {
		return nil, beginRestoreCleanup(journal, fmt.Errorf("create media restore rollback directory: %w", err))
	}
	return &RestoreTransaction{journal: journal}, nil
}

func beginRestoreCleanup(journal restoreJournal, cause error) error {
	if err := removeRestoreDirectory(journal.Staging); err != nil {
		return fmt.Errorf("%w (cleanup staging: %v)", cause, err)
	}
	if err := removeRestoreDirectory(journal.Rollback); err != nil {
		return fmt.Errorf("%w (cleanup rollback: %v)", cause, err)
	}
	if err := os.Remove(restoreJournalPath(journal.Root)); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("%w (cleanup journal: %v)", cause, err)
	}
	return cause
}

func (t *RestoreTransaction) ID() string {
	if t == nil {
		return ""
	}
	return t.journal.ID
}

func (t *RestoreTransaction) RestoreReader(key string, reader io.Reader, size int64) error {
	if t == nil {
		return fmt.Errorf("media restore transaction is nil")
	}
	if !mediaStorageKeyPattern.MatchString(key) || size <= 0 {
		return fmt.Errorf("media restore entry is invalid")
	}
	if err := t.ensureCurrentJournal(); err != nil {
		return err
	}
	if t.journal.State != restoreStateStaging {
		return fmt.Errorf("media restore staging is no longer writable")
	}
	return writeAtomicallyReader(t.journal.Staging, key, reader, size)
}

// Seal records the complete, validated target set once every archive media
// entry is present in staging. This durable list is what makes publication and
// rollback idempotent after a crash between individual file renames.
func (t *RestoreTransaction) Seal(keys []string) error {
	if t == nil {
		return fmt.Errorf("media restore transaction is nil")
	}
	if err := t.ensureCurrentJournal(); err != nil {
		return err
	}
	if t.journal.State != restoreStateStaging {
		return fmt.Errorf("media restore staging is already sealed")
	}
	// Keep an empty (but non-nil) slice for a valid backup with no media. The
	// journal distinguishes that sealed empty target set from unsealed staging.
	files := make([]string, len(keys))
	copy(files, keys)
	sort.Strings(files)
	if err := validateRestoreFiles(files); err != nil {
		return err
	}
	if err := verifyStagingFiles(t.journal.Staging, files); err != nil {
		return err
	}
	t.journal.Files = files
	t.journal.State = restoreStateStaged
	if err := writeRestoreJournal(t.journal); err != nil {
		return fmt.Errorf("seal media restore journal: %w", err)
	}
	return nil
}

// Rollback reverses any partially published file moves when no matching
// database marker exists. Normal restore failures occur before publication and
// only remove staging, but the reverse path also makes interrupted states safe.
func (t *RestoreTransaction) Rollback() error {
	if t == nil {
		return nil
	}
	if err := t.ensureCurrentJournal(); err != nil {
		return err
	}
	switch t.journal.State {
	case restoreStateStaging, restoreStateStaged:
		return t.cleanup()
	case restoreStatePublishingOld:
		if err := restoreRollbackMedia(t.journal); err != nil {
			return err
		}
		return t.cleanup()
	case restoreStatePublishingNew:
		if err := removePublishedMedia(t.journal); err != nil {
			return err
		}
		if err := restoreRollbackMedia(t.journal); err != nil {
			return err
		}
		return t.cleanup()
	default:
		return fmt.Errorf("media restore cannot be rolled back after publication completes")
	}
}

// Publish moves current media into rollback, then moves staged files into the
// media root. Each state change is journaled before the associated rename loop,
// so RecoverRestore can resume the exact phase without cross-device moves.
func (t *RestoreTransaction) Publish() error {
	if t == nil {
		return fmt.Errorf("media restore transaction is nil")
	}
	if err := t.ensureCurrentJournal(); err != nil {
		return err
	}
	return publishRestoreJournal(&t.journal)
}

// Finalize removes the rollback data and journal only after the caller has
// cleared the database restore marker. It is idempotent for startup recovery.
func (t *RestoreTransaction) Finalize() error {
	if t == nil {
		return nil
	}
	if err := t.ensureCurrentJournal(); err != nil {
		return err
	}
	if t.journal.State != restoreStatePublished {
		return fmt.Errorf("media restore has not been published")
	}
	return t.cleanup()
}

func (t *RestoreTransaction) cleanup() error {
	if err := removeRestoreDirectory(t.journal.Staging); err != nil {
		return err
	}
	if err := removeRestoreDirectory(t.journal.Rollback); err != nil {
		return err
	}
	if err := os.Remove(restoreJournalPath(t.journal.Root)); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove media restore journal: %w", err)
	}
	return nil
}

// RecoverRestore reconciles the journal with the database marker. An empty
// marker means the database replace did not commit, so staged or partially
// published files are reversed. A matching marker finishes publication.
func RecoverRestore(root string, committedID string) (*RestoreTransaction, error) {
	resolvedRoot, err := mediaRootPath(root)
	if err != nil {
		return nil, err
	}
	journal, err := loadRestoreJournal(resolvedRoot)
	if errors.Is(err, os.ErrNotExist) {
		if committedID != "" {
			return nil, fmt.Errorf("media restore journal is missing for committed restore")
		}
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	tx := &RestoreTransaction{journal: *journal}
	if committedID == "" {
		if journal.State == restoreStatePublished {
			return tx, nil
		}
		return nil, tx.Rollback()
	}
	if committedID != journal.ID {
		return nil, fmt.Errorf("media restore journal does not match database marker")
	}
	if err := tx.Publish(); err != nil {
		return nil, err
	}
	return tx, nil
}

func newRestoreJournal(root, id string) (restoreJournal, error) {
	resolvedRoot, err := mediaRootPath(root)
	if err != nil {
		return restoreJournal{}, err
	}
	if !restoreIDPattern.MatchString(id) {
		return restoreJournal{}, fmt.Errorf("invalid media restore id")
	}
	return restoreJournal{
		Version:  restoreJournalVersion,
		ID:       id,
		Root:     resolvedRoot,
		Staging:  filepath.Join(resolvedRoot, restoreStagingPrefix+id),
		Rollback: filepath.Join(resolvedRoot, restoreRollbackPrefix+id),
		State:    restoreStateStaging,
	}, nil
}

func restoreJournalPath(root string) string {
	return filepath.Join(root, restoreJournalFilename)
}

func ensureRestorePathAbsent(path string) error {
	if _, err := os.Lstat(path); err == nil {
		return fmt.Errorf("media restore path already exists")
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func createRestoreDirectory(path string) error {
	if err := os.Mkdir(path, restoreDirectoryPermissions); err != nil {
		return err
	}
	return nil
}

func (t *RestoreTransaction) ensureCurrentJournal() error {
	journal, err := loadRestoreJournal(t.journal.Root)
	if err != nil {
		return err
	}
	if journal.ID != t.journal.ID {
		return fmt.Errorf("media restore journal changed")
	}
	t.journal = *journal
	return nil
}

func loadRestoreJournal(root string) (*restoreJournal, error) {
	data, err := os.ReadFile(restoreJournalPath(root))
	if err != nil {
		return nil, err
	}
	var journal restoreJournal
	if err := json.Unmarshal(data, &journal); err != nil {
		return nil, fmt.Errorf("decode media restore journal: %w", err)
	}
	expected, err := newRestoreJournal(root, journal.ID)
	if err != nil {
		return nil, err
	}
	if journal.Version != restoreJournalVersion || journal.Root != expected.Root || journal.Staging != expected.Staging || journal.Rollback != expected.Rollback {
		return nil, fmt.Errorf("media restore journal is invalid")
	}
	switch journal.State {
	case restoreStateStaging:
		if journal.Files != nil {
			return nil, fmt.Errorf("media restore journal is invalid")
		}
	case restoreStateStaged, restoreStatePublishingOld, restoreStatePublishingNew, restoreStatePublished:
		if journal.Files == nil || validateRestoreFiles(journal.Files) != nil {
			return nil, fmt.Errorf("media restore journal is invalid")
		}
	default:
		return nil, fmt.Errorf("media restore journal state is invalid")
	}
	return &journal, nil
}

func validateRestoreFiles(files []string) error {
	for index, key := range files {
		if !mediaStorageKeyPattern.MatchString(key) {
			return fmt.Errorf("media restore file key is invalid")
		}
		if index > 0 && files[index-1] >= key {
			return fmt.Errorf("media restore file keys are not unique")
		}
	}
	return nil
}

func verifyStagingFiles(staging string, files []string) error {
	expected := make(map[string]struct{}, len(files))
	for _, key := range files {
		expected[key] = struct{}{}
		if exists, err := restoreRegularFileExists(filepath.Join(staging, key)); err != nil {
			return err
		} else if !exists {
			return fmt.Errorf("staged media file is missing")
		}
	}
	entries, err := os.ReadDir(staging)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if _, ok := expected[entry.Name()]; !ok {
			return fmt.Errorf("staged media directory contains an unexpected entry")
		}
	}
	return nil
}

func writeRestoreJournal(journal restoreJournal) error {
	data, err := json.Marshal(journal)
	if err != nil {
		return err
	}
	journalPath := restoreJournalPath(journal.Root)
	tmp, err := os.CreateTemp(journal.Root, ".restore-journal-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(0600); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, journalPath)
}

func publishRestoreJournal(journal *restoreJournal) error {
	switch journal.State {
	case restoreStateStaging:
		return fmt.Errorf("media restore staging has not been sealed")
	case restoreStateStaged:
		journal.State = restoreStatePublishingOld
		if err := writeRestoreJournal(*journal); err != nil {
			return fmt.Errorf("mark media restore old-file publication: %w", err)
		}
		fallthrough
	case restoreStatePublishingOld:
		if err := moveCurrentMediaToRollback(*journal); err != nil {
			return err
		}
		journal.State = restoreStatePublishingNew
		if err := writeRestoreJournal(*journal); err != nil {
			return fmt.Errorf("mark media restore new-file publication: %w", err)
		}
		fallthrough
	case restoreStatePublishingNew:
		if err := moveStagedMediaToRoot(*journal); err != nil {
			return err
		}
		journal.State = restoreStatePublished
		if err := writeRestoreJournal(*journal); err != nil {
			return fmt.Errorf("mark media restore published: %w", err)
		}
		return nil
	case restoreStatePublished:
		if exists, err := restoreDirectoryExists(journal.Root); err != nil {
			return err
		} else if !exists {
			return fmt.Errorf("published media root is missing")
		}
		return nil
	default:
		return fmt.Errorf("media restore journal state is invalid")
	}
}

func moveCurrentMediaToRollback(journal restoreJournal) error {
	if err := ensureExistingRestoreDirectory(journal.Rollback); err != nil {
		return err
	}
	entries, err := os.ReadDir(journal.Root)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		key := entry.Name()
		if !mediaStorageKeyPattern.MatchString(key) {
			continue
		}
		currentPath := filepath.Join(journal.Root, key)
		if exists, err := restoreRegularFileExists(currentPath); err != nil {
			return err
		} else if !exists {
			continue
		}
		rollbackPath := filepath.Join(journal.Rollback, key)
		if exists, err := restoreRegularFileExists(rollbackPath); err != nil {
			return err
		} else if exists {
			return fmt.Errorf("media restore rollback file already exists")
		}
		if err := os.Rename(currentPath, rollbackPath); err != nil {
			return fmt.Errorf("move current media file to rollback: %w", err)
		}
	}
	return nil
}

func moveStagedMediaToRoot(journal restoreJournal) error {
	for _, key := range journal.Files {
		stagingPath := filepath.Join(journal.Staging, key)
		targetPath := filepath.Join(journal.Root, key)
		stagingExists, err := restoreRegularFileExists(stagingPath)
		if err != nil {
			return err
		}
		targetExists, err := restoreRegularFileExists(targetPath)
		if err != nil {
			return err
		}
		switch {
		case stagingExists && targetExists:
			return fmt.Errorf("media restore target already exists")
		case stagingExists:
			if err := os.Rename(stagingPath, targetPath); err != nil {
				return fmt.Errorf("publish staged media file: %w", err)
			}
		case targetExists:
			hash, err := fileSHA256(targetPath)
			if err != nil {
				return err
			}
			if hash != strings.SplitN(key, ".", 2)[0] {
				return fmt.Errorf("published media file checksum mismatch")
			}
		default:
			return fmt.Errorf("staged media file is missing")
		}
	}
	return nil
}

func removePublishedMedia(journal restoreJournal) error {
	for _, key := range journal.Files {
		stagingPath := filepath.Join(journal.Staging, key)
		stagingExists, err := restoreRegularFileExists(stagingPath)
		if err != nil {
			return err
		}
		if stagingExists {
			continue
		}
		targetPath := filepath.Join(journal.Root, key)
		targetExists, err := restoreRegularFileExists(targetPath)
		if err != nil {
			return err
		}
		if targetExists {
			if err := os.Remove(targetPath); err != nil {
				return fmt.Errorf("remove partially published media file: %w", err)
			}
		}
	}
	return nil
}

func restoreRollbackMedia(journal restoreJournal) error {
	entries, err := os.ReadDir(journal.Rollback)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	for _, entry := range entries {
		key := entry.Name()
		if !mediaStorageKeyPattern.MatchString(key) {
			continue
		}
		rollbackPath := filepath.Join(journal.Rollback, key)
		if exists, err := restoreRegularFileExists(rollbackPath); err != nil {
			return err
		} else if !exists {
			continue
		}
		targetPath := filepath.Join(journal.Root, key)
		if exists, err := restoreRegularFileExists(targetPath); err != nil {
			return err
		} else if exists {
			return fmt.Errorf("media restore rollback target already exists")
		}
		if err := os.Rename(rollbackPath, targetPath); err != nil {
			return fmt.Errorf("restore rollback media file: %w", err)
		}
	}
	return nil
}

func ensureExistingRestoreDirectory(path string) error {
	exists, err := restoreDirectoryExists(path)
	if err != nil {
		return err
	}
	if !exists {
		return createRestoreDirectory(path)
	}
	return nil
}

func restoreDirectoryExists(path string) (bool, error) {
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return false, fmt.Errorf("media restore path is not a directory")
	}
	return true, nil
}

func restoreRegularFileExists(path string) (bool, error) {
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return false, fmt.Errorf("media restore path is not a regular file")
	}
	return true, nil
}

func removeRestoreDirectory(path string) error {
	exists, err := restoreDirectoryExists(path)
	if err != nil || !exists {
		return err
	}
	if err := os.RemoveAll(path); err != nil {
		return fmt.Errorf("remove media restore directory: %w", err)
	}
	return nil
}
