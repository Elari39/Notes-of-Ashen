package backup

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"filippo.io/age"
	"github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/core/logx"
	"golang.org/x/crypto/bcrypt"

	"notes-of-ashen/internal/authutil"
	apperrors "notes-of-ashen/internal/errors"
	articlelogic "notes-of-ashen/internal/logic/article"
	medialogic "notes-of-ashen/internal/logic/media"
	"notes-of-ashen/internal/security"
	"notes-of-ashen/internal/svc"
	"notes-of-ashen/internal/types"
	"notes-of-ashen/model"
)

const (
	archiveVersion          = 1
	maxBackupArchiveEntries = 100000
	maxBackupManifestBytes  = int64(8 << 20)
)

var (
	mediaKeyPattern = regexp.MustCompile(`^[a-f0-9]{64}\.(jpg|png|gif|webp)$`)
	sha256Pattern   = regexp.MustCompile(`^[a-f0-9]{64}$`)
)

type ManifestFile struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
}

type Manifest struct {
	Version    int            `json:"version"`
	ExportedAt time.Time      `json:"exportedAt"`
	DataSHA256 string         `json:"dataSha256"`
	Files      []ManifestFile `json:"files"`
	Counts     map[string]int `json:"counts"`
}

func Export(ctx context.Context, svcCtx *svc.ServiceContext, req types.BackupExportReq) (string, error) {
	if err := authorize(ctx, svcCtx, req.CurrentPassword, req.Passphrase); err != nil {
		return "", err
	}
	snapshot, err := svcCtx.Store.ExportBackup(ctx)
	if err != nil {
		return "", err
	}
	if err := validateExportArchiveLimits(len(snapshot.MediaAssets), 0); err != nil {
		return "", err
	}
	data, err := json.Marshal(snapshot)
	if err != nil {
		return "", err
	}
	maxBytes := svcCtx.Config.Backup.EffectiveMaxUploadBytes()
	if exceedsBackupSizeLimit(0, int64(len(data)), maxBytes) {
		return "", apperrors.BadRequest("backup exceeds configured size limit")
	}
	dataHash := sha256.Sum256(data)
	manifest := Manifest{
		Version: archiveVersion, ExportedAt: time.Now().UTC(), DataSHA256: hex.EncodeToString(dataHash[:]),
		Files: make([]ManifestFile, 0, len(snapshot.MediaAssets)),
		Counts: map[string]int{
			"users": len(snapshot.Users), "settings": len(snapshot.Settings), "categories": len(snapshot.Categories),
			"tags": len(snapshot.Tags), "projects": len(snapshot.Projects), "articles": len(snapshot.Articles),
			"versions": len(snapshot.Versions), "media": len(snapshot.MediaAssets),
		},
	}
	root, err := medialogic.Root(svcCtx)
	if err != nil {
		return "", err
	}
	var total int64 = int64(len(data))
	for _, asset := range snapshot.MediaAssets {
		if !mediaKeyPattern.MatchString(asset.StorageKey) {
			return "", fmt.Errorf("invalid media storage key")
		}
		filePath := filepath.Join(root, asset.StorageKey)
		info, err := os.Stat(filePath)
		if err != nil {
			return "", fmt.Errorf("read media file: %w", err)
		}
		hash, err := hashFile(filePath)
		if err != nil {
			return "", err
		}
		if hash != asset.SHA256 {
			return "", fmt.Errorf("media checksum mismatch")
		}
		if exceedsBackupSizeLimit(total, info.Size(), maxBytes) {
			return "", apperrors.BadRequest("backup exceeds configured size limit")
		}
		total += info.Size()
		manifest.Files = append(manifest.Files, ManifestFile{Path: "media/" + asset.StorageKey, SHA256: hash, Size: info.Size()})
	}
	manifestData, err := json.Marshal(manifest)
	if err != nil {
		return "", err
	}
	if err := validateExportArchiveLimits(len(snapshot.MediaAssets), int64(len(manifestData))); err != nil {
		return "", err
	}
	// 恢复时 manifest.json 也计入解压后的大小，导出端必须使用相同边界。
	if exceedsBackupSizeLimit(total, int64(len(manifestData)), maxBytes) {
		return "", apperrors.BadRequest("backup exceeds configured size limit")
	}
	tmp, err := os.CreateTemp("", "notes-of-ashen-*.noa-backup")
	if err != nil {
		return "", err
	}
	tmpPath := tmp.Name()
	cleanup := true
	defer func() {
		_ = tmp.Close()
		if cleanup {
			_ = os.Remove(tmpPath)
		}
	}()
	recipient, err := age.NewScryptRecipient(req.Passphrase)
	if err != nil {
		return "", err
	}
	encrypted, err := age.Encrypt(tmp, recipient)
	if err != nil {
		return "", err
	}
	archive := zip.NewWriter(encrypted)
	if err := writeZipBytes(archive, "manifest.json", manifestData); err != nil {
		return "", err
	}
	if err := writeZipBytes(archive, "data.json", data); err != nil {
		return "", err
	}
	for _, asset := range snapshot.MediaAssets {
		if err := writeZipFile(archive, "media/"+asset.StorageKey, filepath.Join(root, asset.StorageKey)); err != nil {
			return "", err
		}
	}
	if err := archive.Close(); err != nil {
		return "", err
	}
	if err := encrypted.Close(); err != nil {
		return "", err
	}
	if err := tmp.Close(); err != nil {
		return "", err
	}
	info, err := os.Stat(tmpPath)
	if err != nil {
		return "", err
	}
	// 上传入口限制的是最终的加密文件；不能返回本服务自身无法重新接收的归档。
	if exceedsBackupSizeLimit(0, info.Size(), maxBytes) {
		return "", apperrors.BadRequest("backup exceeds configured size limit")
	}
	cleanup = false
	return tmpPath, nil
}

func Restore(ctx context.Context, svcCtx *svc.ServiceContext, currentPassword, passphrase, confirmation string, encrypted io.Reader) (*types.BackupRestoreResp, error) {
	if confirmation != "REPLACE" {
		return nil, apperrors.BadRequest("restore confirmation is invalid")
	}
	if err := authorize(ctx, svcCtx, currentPassword, passphrase); err != nil {
		return nil, err
	}
	if !security.TryStartRestore() {
		return nil, apperrors.Conflict("another restore is running")
	}
	defer security.EndRestore()
	lease, restoreCtx, err := security.AcquireRestoreLease(ctx, svcCtx.Redis)
	if err != nil {
		if errors.Is(err, security.ErrRestoreLeaseHeld) {
			return nil, apperrors.Conflict("another restore is running")
		}
		return nil, apperrors.ServiceUnavailable("restore lock is unavailable")
	}
	defer func() {
		if releaseErr := lease.Release(); releaseErr != nil {
			logx.Errorf("release restore lease failed: %v", releaseErr)
		}
	}()

	identity, err := age.NewScryptIdentity(passphrase)
	if err != nil {
		return nil, apperrors.BadRequest("backup passphrase is invalid")
	}
	decrypted, err := age.Decrypt(encrypted, identity)
	if err != nil {
		return nil, apperrors.BadRequest("backup passphrase or file is invalid")
	}
	tmp, err := os.CreateTemp("", "notes-of-ashen-restore-*.zip")
	if err != nil {
		return nil, err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	defer tmp.Close()
	maxBytes := svcCtx.Config.Backup.EffectiveMaxUploadBytes()
	written, err := io.Copy(tmp, io.LimitReader(decrypted, maxBytes+1))
	if err != nil {
		return nil, apperrors.BadRequest("backup cannot be decrypted")
	}
	if written > maxBytes {
		return nil, apperrors.BadRequest("backup exceeds configured size limit")
	}
	if err := tmp.Close(); err != nil {
		return nil, err
	}
	reader, err := zip.OpenReader(tmpPath)
	if err != nil {
		return nil, apperrors.BadRequest("backup archive is invalid")
	}
	defer reader.Close()
	manifest, snapshot, err := readArchive(&reader.Reader, maxBytes)
	if err != nil {
		return nil, err
	}
	if err := validateSnapshot(snapshot, manifest); err != nil {
		return nil, err
	}
	if err := requireRestoreLease(lease); err != nil {
		return nil, err
	}
	if err := restoreArchiveMedia(svcCtx, &reader.Reader, manifest, func() error {
		return requireLocalRestoreLease(lease)
	}); err != nil {
		return nil, err
	}
	if err := requireRestoreLease(lease); err != nil {
		return nil, err
	}
	if err := svcCtx.Store.RestoreBackup(restoreCtx, *snapshot); err != nil {
		return nil, err
	}
	warnings := make([]string, 0)
	postRestoreCtx, cancelPostRestore := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelPostRestore()
	if err := clearApplicationCache(postRestoreCtx, svcCtx.Redis); err != nil {
		warnings = append(warnings, "应用缓存清理失败")
		logx.Errorf("clear application cache after restore failed: %v", err)
	}
	tokenCutoff := time.Now().Unix()
	security.SetAccessTokensNotBefore(tokenCutoff)
	if err := svcCtx.Redis.Set(postRestoreCtx, security.AccessTokensNotBeforeKey, tokenCutoff, 0).Err(); err != nil {
		warnings = append(warnings, "访问令牌失效标记写入失败")
	}
	if _, err := articlelogic.ReindexSearch(ctx, svcCtx); err != nil {
		warnings = append(warnings, "搜索索引重建失败，已保留 MySQL 搜索降级")
		logx.Errorf("reindex after restore failed: %v", err)
	}
	if err := pruneMedia(svcCtx, snapshot.MediaAssets); err != nil {
		warnings = append(warnings, "旧媒体文件清理失败")
		logx.Errorf("prune media after restore failed: %v", err)
	}
	return &types.BackupRestoreResp{Users: len(snapshot.Users), Articles: len(snapshot.Articles), Media: len(snapshot.MediaAssets), Warnings: warnings}, nil
}

func authorize(ctx context.Context, svcCtx *svc.ServiceContext, password, passphrase string) error {
	if err := authutil.RequireAdmin(ctx); err != nil {
		return err
	}
	if utf8.RuneCountInString(passphrase) < 12 || utf8.RuneCountInString(passphrase) > 128 {
		return apperrors.BadRequest("backup passphrase length is invalid")
	}
	userID, err := authutil.UserID(ctx)
	if err != nil {
		return err
	}
	user, err := svcCtx.Store.FindUserByID(ctx, userID)
	if err != nil {
		return err
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil {
		return apperrors.Unauthorized("current password is incorrect")
	}
	return nil
}

func exceedsBackupSizeLimit(current, addition, limit int64) bool {
	return current < 0 || addition < 0 || limit < 0 || current > limit || addition > limit-current
}

func validateExportArchiveLimits(mediaCount int, manifestSize int64) error {
	// manifest.json 和 data.json 也占用两个 ZIP entry，导出端必须保持在恢复端的总入口上限内。
	if mediaCount < 0 || mediaCount > maxBackupArchiveEntries-2 {
		return apperrors.BadRequest("backup has too many files")
	}
	if manifestSize < 0 || manifestSize > maxBackupManifestBytes {
		return apperrors.BadRequest("backup manifest exceeds size limit")
	}
	return nil
}

func requireRestoreLease(lease *security.RestoreLease) error {
	return normalizeRestoreLeaseError(lease.Check())
}

func requireLocalRestoreLease(lease *security.RestoreLease) error {
	return normalizeRestoreLeaseError(lease.CheckLocal())
}

func normalizeRestoreLeaseError(err error) error {
	if err != nil {
		if errors.Is(err, security.ErrRestoreLeaseLost) {
			return apperrors.ServiceUnavailable("restore lock is unavailable")
		}
		return err
	}
	return nil
}

func readArchive(reader *zip.Reader, maxBytes int64) (*Manifest, *model.BackupSnapshot, error) {
	if maxBytes <= 0 {
		return nil, nil, apperrors.BadRequest("backup size limit is invalid")
	}
	if len(reader.File) > maxBackupArchiveEntries {
		return nil, nil, apperrors.BadRequest("backup has too many files")
	}
	entries := make(map[string]*zip.File, len(reader.File))
	var total uint64
	for _, file := range reader.File {
		name := path.Clean(strings.ReplaceAll(file.Name, "\\", "/"))
		if name == "." || strings.HasPrefix(name, "../") || strings.HasPrefix(name, "/") || name != file.Name || file.Mode()&os.ModeSymlink != 0 {
			return nil, nil, apperrors.BadRequest("backup contains an unsafe path")
		}
		if _, exists := entries[name]; exists {
			return nil, nil, apperrors.BadRequest("backup contains duplicate paths")
		}
		if file.UncompressedSize64 > uint64(maxBytes)-total {
			return nil, nil, apperrors.BadRequest("backup expanded size exceeds limit")
		}
		total += file.UncompressedSize64
		entries[name] = file
	}
	manifestData, err := readZipEntry(entries["manifest.json"], maxBackupManifestBytes)
	if err != nil {
		return nil, nil, apperrors.BadRequest("backup manifest is missing")
	}
	var manifest Manifest
	if json.Unmarshal(manifestData, &manifest) != nil || manifest.Version != archiveVersion {
		return nil, nil, apperrors.BadRequest("backup version is not supported")
	}
	if len(entries) != len(manifest.Files)+2 {
		return nil, nil, apperrors.BadRequest("backup contains unexpected files")
	}
	data, err := readZipEntry(entries["data.json"], maxBytes)
	if err != nil {
		return nil, nil, apperrors.BadRequest("backup data is missing")
	}
	sum := sha256.Sum256(data)
	if hex.EncodeToString(sum[:]) != manifest.DataSHA256 {
		return nil, nil, apperrors.BadRequest("backup data checksum mismatch")
	}
	var snapshot model.BackupSnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return nil, nil, apperrors.BadRequest("backup data is invalid")
	}
	manifestPaths := make(map[string]struct{}, len(manifest.Files))
	for _, item := range manifest.Files {
		key := strings.TrimPrefix(item.Path, "media/")
		file := entries[item.Path]
		if file == nil || item.Path != "media/"+key || !mediaKeyPattern.MatchString(key) || item.Size <= 0 {
			return nil, nil, apperrors.BadRequest("backup media manifest is invalid")
		}
		if _, exists := manifestPaths[item.Path]; exists {
			return nil, nil, apperrors.BadRequest("backup media manifest is duplicated")
		}
		manifestPaths[item.Path] = struct{}{}
		hash, size, err := hashZipEntry(file, maxBytes)
		if err != nil || size != item.Size || hash != item.SHA256 {
			return nil, nil, apperrors.BadRequest("backup media checksum mismatch")
		}
	}
	return &manifest, &snapshot, nil
}

func restoreArchiveMedia(svcCtx *svc.ServiceContext, reader *zip.Reader, manifest *Manifest, canContinue func() error) error {
	entries := make(map[string]*zip.File, len(reader.File))
	for _, file := range reader.File {
		entries[file.Name] = file
	}
	for _, item := range manifest.Files {
		if canContinue != nil {
			if err := canContinue(); err != nil {
				return err
			}
		}
		file := entries[item.Path]
		if file == nil {
			return apperrors.BadRequest("backup media file is missing")
		}
		entry, err := file.Open()
		if err != nil {
			return err
		}
		key := strings.TrimPrefix(item.Path, "media/")
		restoreErr := medialogic.RestoreReader(svcCtx, key, entry, item.Size)
		closeErr := entry.Close()
		if restoreErr != nil {
			return restoreErr
		}
		if closeErr != nil {
			return closeErr
		}
	}
	return nil
}

func validateSnapshot(snapshot *model.BackupSnapshot, manifest *Manifest) error {
	expectedCounts := map[string]int{
		"users": len(snapshot.Users), "settings": len(snapshot.Settings), "categories": len(snapshot.Categories),
		"tags": len(snapshot.Tags), "projects": len(snapshot.Projects), "articles": len(snapshot.Articles),
		"versions": len(snapshot.Versions), "media": len(snapshot.MediaAssets),
	}
	for name, count := range expectedCounts {
		if manifest.Counts[name] != count {
			return apperrors.BadRequest("backup counts do not match")
		}
	}
	if manifest.ExportedAt.IsZero() || !sha256Pattern.MatchString(manifest.DataSHA256) {
		return apperrors.BadRequest("backup manifest is invalid")
	}
	if len(manifest.Files) != len(snapshot.MediaAssets) {
		return apperrors.BadRequest("backup counts do not match")
	}
	users := map[uint64]struct{}{}
	accounts := map[string]struct{}{}
	emails := map[string]struct{}{}
	hasAdmin := false
	for _, u := range snapshot.Users {
		if u.ID == 0 {
			return apperrors.BadRequest("backup user is invalid")
		}
		if _, ok := users[u.ID]; ok {
			return apperrors.BadRequest("backup contains duplicate users")
		}
		users[u.ID] = struct{}{}
		account := strings.ToLower(u.Account)
		email := strings.ToLower(u.Email)
		if _, ok := accounts[account]; ok {
			return apperrors.BadRequest("backup contains duplicate accounts")
		}
		if _, ok := emails[email]; ok {
			return apperrors.BadRequest("backup contains duplicate emails")
		}
		accounts[account] = struct{}{}
		emails[email] = struct{}{}
		if !authutil.IsValidRole(u.Role) || (u.Status != "active" && u.Status != "disabled") {
			return apperrors.BadRequest("backup user role or status is invalid")
		}
		if _, err := bcrypt.Cost([]byte(u.PasswordHash)); err != nil {
			return apperrors.BadRequest("backup password hash is invalid")
		}
		if u.Role == "admin" && u.Status == "active" {
			hasAdmin = true
		}
	}
	if !hasAdmin {
		return apperrors.BadRequest("backup must contain an active admin")
	}
	settingKeys := map[string]struct{}{}
	for _, setting := range snapshot.Settings {
		if setting.Key == "" || len(setting.Key) > 64 || setting.Key == "ai_api_key_cipher" {
			return apperrors.BadRequest("backup setting is invalid")
		}
		if _, exists := settingKeys[setting.Key]; exists {
			return apperrors.BadRequest("backup contains duplicate settings")
		}
		settingKeys[setting.Key] = struct{}{}
	}
	categories := map[uint64]struct{}{}
	categoryNames := map[string]struct{}{}
	categorySlugs := map[string]struct{}{}
	for _, category := range snapshot.Categories {
		if category.ID == 0 || category.Name == "" || category.Slug == "" {
			return apperrors.BadRequest("backup category is invalid")
		}
		if _, ok := users[category.CreatedBy]; !ok {
			return apperrors.BadRequest("backup category creator is invalid")
		}
		if _, exists := categories[category.ID]; exists || duplicateFolded(categoryNames, category.Name) || duplicateFolded(categorySlugs, category.Slug) {
			return apperrors.BadRequest("backup contains duplicate categories")
		}
		categories[category.ID] = struct{}{}
	}
	tags := map[uint64]struct{}{}
	tagNames := map[string]struct{}{}
	tagSlugs := map[string]struct{}{}
	for _, t := range snapshot.Tags {
		if t.ID == 0 || t.Name == "" || t.Slug == "" {
			return apperrors.BadRequest("backup tag is invalid")
		}
		if _, ok := users[t.CreatedBy]; !ok {
			return apperrors.BadRequest("backup tag creator is invalid")
		}
		if _, exists := tags[t.ID]; exists || duplicateFolded(tagNames, t.Name) || duplicateFolded(tagSlugs, t.Slug) {
			return apperrors.BadRequest("backup contains duplicate tags")
		}
		tags[t.ID] = struct{}{}
	}
	projects := map[uint64]struct{}{}
	for _, project := range snapshot.Projects {
		id, err := strconv.ParseUint(project.ID, 10, 64)
		if err != nil || id == 0 || strings.TrimSpace(project.Title) == "" {
			return apperrors.BadRequest("backup project is invalid")
		}
		if _, exists := projects[id]; exists || hasDuplicateUint64(project.TagIDs) {
			return apperrors.BadRequest("backup contains duplicate project data")
		}
		projects[id] = struct{}{}
		for _, tagID := range project.TagIDs {
			if _, ok := tags[tagID]; !ok {
				return apperrors.BadRequest("backup project tag is invalid")
			}
		}
	}
	articles := map[uint64]struct{}{}
	slugs := map[string]struct{}{}
	for _, entry := range snapshot.Articles {
		a := entry.Article
		if a.ID == 0 || strings.TrimSpace(a.Title) == "" || strings.TrimSpace(a.Slug) == "" {
			return apperrors.BadRequest("backup article is invalid")
		}
		if _, exists := articles[a.ID]; exists {
			return apperrors.BadRequest("backup contains duplicate articles")
		}
		if _, ok := users[a.AuthorID]; !ok {
			return apperrors.BadRequest("backup article author is invalid")
		}
		if a.CategoryID > 0 {
			if _, ok := categories[a.CategoryID]; !ok {
				return apperrors.BadRequest("backup article category is invalid")
			}
		}
		if !validArticleStatus(a.Status) {
			return apperrors.BadRequest("backup article status is invalid")
		}
		if duplicateFolded(slugs, a.Slug) {
			return apperrors.BadRequest("backup article slug is duplicated")
		}
		articles[a.ID] = struct{}{}
		if hasDuplicateUint64(entry.TagIDs) {
			return apperrors.BadRequest("backup article tags are duplicated")
		}
		for _, tagID := range entry.TagIDs {
			if _, ok := tags[tagID]; !ok {
				return apperrors.BadRequest("backup article tag is invalid")
			}
		}
	}
	versionIDs := map[uint64]struct{}{}
	versionNumbers := map[string]struct{}{}
	for _, version := range snapshot.Versions {
		if version.ID == 0 || version.VersionNo < 1 {
			return apperrors.BadRequest("backup article version is invalid")
		}
		if _, exists := versionIDs[version.ID]; exists {
			return apperrors.BadRequest("backup contains duplicate article versions")
		}
		versionIDs[version.ID] = struct{}{}
		if _, ok := articles[version.ArticleID]; !ok {
			return apperrors.BadRequest("backup article version is invalid")
		}
		versionKey := strconv.FormatUint(version.ArticleID, 10) + ":" + strconv.Itoa(version.VersionNo)
		if _, exists := versionNumbers[versionKey]; exists {
			return apperrors.BadRequest("backup contains duplicate article versions")
		}
		versionNumbers[versionKey] = struct{}{}
		if _, ok := users[version.AuthorID]; !ok {
			return apperrors.BadRequest("backup article version author is invalid")
		}
		if _, ok := users[version.ChangedBy]; !ok {
			return apperrors.BadRequest("backup article version editor is invalid")
		}
		if version.CategoryID > 0 {
			if _, ok := categories[version.CategoryID]; !ok {
				return apperrors.BadRequest("backup article version category is invalid")
			}
		}
		if !validArticleStatus(version.Status) || hasDuplicateUint64(version.TagIDs) {
			return apperrors.BadRequest("backup article version data is invalid")
		}
		for _, tagID := range version.TagIDs {
			if _, ok := tags[tagID]; !ok {
				return apperrors.BadRequest("backup article version tag is invalid")
			}
		}
	}
	manifestMedia := map[string]ManifestFile{}
	for _, file := range manifest.Files {
		key := strings.TrimPrefix(file.Path, "media/")
		if file.Path != "media/"+key || !mediaKeyPattern.MatchString(key) || file.Size <= 0 || file.SHA256 != strings.Split(key, ".")[0] {
			return apperrors.BadRequest("backup media manifest is invalid")
		}
		if _, exists := manifestMedia[key]; exists {
			return apperrors.BadRequest("backup media manifest is duplicated")
		}
		manifestMedia[key] = file
	}
	mediaIDs := map[uint64]struct{}{}
	mediaKeys := map[string]struct{}{}
	for _, asset := range snapshot.MediaAssets {
		if asset.ID == 0 || asset.SizeBytes == 0 || !mediaKeyPattern.MatchString(asset.StorageKey) || asset.SHA256 != strings.Split(asset.StorageKey, ".")[0] {
			return apperrors.BadRequest("backup media metadata is invalid")
		}
		if _, ok := users[asset.CreatedBy]; !ok {
			return apperrors.BadRequest("backup media creator is invalid")
		}
		if _, exists := mediaIDs[asset.ID]; exists {
			return apperrors.BadRequest("backup media metadata is duplicated")
		}
		mediaIDs[asset.ID] = struct{}{}
		if _, exists := mediaKeys[asset.StorageKey]; exists {
			return apperrors.BadRequest("backup media metadata is duplicated")
		}
		mediaKeys[asset.StorageKey] = struct{}{}
		file, ok := manifestMedia[asset.StorageKey]
		if !ok || uint64(file.Size) != asset.SizeBytes || file.SHA256 != asset.SHA256 {
			return apperrors.BadRequest("backup media file is missing")
		}
	}
	return nil
}

func duplicateFolded(values map[string]struct{}, value string) bool {
	key := strings.ToLower(strings.TrimSpace(value))
	if _, exists := values[key]; exists {
		return true
	}
	values[key] = struct{}{}
	return false
}

func hasDuplicateUint64(values []uint64) bool {
	seen := make(map[uint64]struct{}, len(values))
	for _, value := range values {
		if value == 0 {
			return true
		}
		if _, exists := seen[value]; exists {
			return true
		}
		seen[value] = struct{}{}
	}
	return false
}

func validArticleStatus(status string) bool {
	switch status {
	case model.ArticleStatusDraft, model.ArticleStatusPublished, model.ArticleStatusArchived, model.ArticleStatusScheduled:
		return true
	default:
		return false
	}
}

func writeZipBytes(archive *zip.Writer, name string, data []byte) error {
	writer, err := archive.Create(name)
	if err != nil {
		return err
	}
	_, err = writer.Write(data)
	return err
}
func writeZipFile(archive *zip.Writer, name, filePath string) error {
	source, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer source.Close()
	writer, err := archive.Create(name)
	if err != nil {
		return err
	}
	_, err = io.Copy(writer, source)
	return err
}
func readZipEntry(file *zip.File, limit int64) ([]byte, error) {
	if file == nil {
		return nil, os.ErrNotExist
	}
	reader, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	raw, err := io.ReadAll(io.LimitReader(reader, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(raw)) > limit {
		return nil, errors.New("entry too large")
	}
	return raw, nil
}
func hashZipEntry(file *zip.File, limit int64) (string, int64, error) {
	if file == nil {
		return "", 0, os.ErrNotExist
	}
	reader, err := file.Open()
	if err != nil {
		return "", 0, err
	}
	defer reader.Close()
	hash := sha256.New()
	size, err := io.Copy(hash, io.LimitReader(reader, limit+1))
	if err != nil {
		return "", 0, err
	}
	if size > limit {
		return "", 0, errors.New("entry too large")
	}
	return hex.EncodeToString(hash.Sum(nil)), size, nil
}
func hashFile(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

type cacheRedis interface {
	Scan(ctx context.Context, cursor uint64, match string, count int64) *redis.ScanCmd
	Del(ctx context.Context, keys ...string) *redis.IntCmd
}

func clearApplicationCache(ctx context.Context, client cacheRedis) error {
	if client == nil {
		return errors.New("cache redis client is nil")
	}
	patterns := []string{
		"site:*", "article:*", "auth:*", "notes-of-ashen:*", "captcha:*",
		"verify_code:*", "verify_code_cooldown:*", "traffic:*",
	}
	for _, pattern := range patterns {
		var cursor uint64
		for {
			keys, next, err := client.Scan(ctx, cursor, pattern, 200).Result()
			if err != nil {
				return err
			}
			if len(keys) > 0 {
				if err := client.Del(ctx, keys...).Err(); err != nil {
					return err
				}
			}
			cursor = next
			if cursor == 0 {
				break
			}
		}
	}
	return nil
}
func pruneMedia(svcCtx *svc.ServiceContext, assets []model.MediaAsset) error {
	root, err := medialogic.Root(svcCtx)
	if err != nil {
		return err
	}
	keep := map[string]struct{}{}
	for _, asset := range assets {
		keep[asset.StorageKey] = struct{}{}
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() || !mediaKeyPattern.MatchString(entry.Name()) {
			continue
		}
		if _, ok := keep[entry.Name()]; !ok {
			if err := os.Remove(filepath.Join(root, entry.Name())); err != nil {
				return err
			}
		}
	}
	return nil
}
