package media

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf8"

	"notes-of-ashen/internal/authutil"
	apperrors "notes-of-ashen/internal/errors"
	"notes-of-ashen/internal/logicutil"
	"notes-of-ashen/internal/svc"
	"notes-of-ashen/internal/types"
	"notes-of-ashen/model"

	_ "github.com/gen2brain/avif"
	"github.com/zeromicro/go-zero/core/logx"
	_ "golang.org/x/image/webp"
)

type mediaFormat struct {
	mimeType         string
	storageExtension string
	sourceExtensions map[string]struct{}
}

var allowedMediaFormats = map[string]mediaFormat{
	"jpeg": {mimeType: "image/jpeg", storageExtension: ".jpg", sourceExtensions: map[string]struct{}{".jpg": {}, ".jpeg": {}}},
	"png":  {mimeType: "image/png", storageExtension: ".png", sourceExtensions: map[string]struct{}{".png": {}}},
	"gif":  {mimeType: "image/gif", storageExtension: ".gif", sourceExtensions: map[string]struct{}{".gif": {}}},
	"webp": {mimeType: "image/webp", storageExtension: ".webp", sourceExtensions: map[string]struct{}{".webp": {}}},
	"avif": {mimeType: "image/avif", storageExtension: ".avif", sourceExtensions: map[string]struct{}{".avif": {}}},
}

var mediaStorageKeyPattern = regexp.MustCompile(`^[a-f0-9]{64}\.(jpg|png|gif|webp|avif)$`)
var stagedUploadPattern = regexp.MustCompile(`^\.upload-([a-f0-9]{64}\.(?:jpg|png|gif|webp|avif))-.+$`)
var stagedDeletePattern = regexp.MustCompile(`^\.delete-([0-9]+)-([a-f0-9]{64}\.(?:jpg|png|gif|webp|avif))$`)

func List(ctx context.Context, svcCtx *svc.ServiceContext, page, size int, query string) (*types.ListResp[types.MediaAssetResp], error) {
	if err := authutil.RequireContentManager(ctx); err != nil {
		return nil, err
	}
	page, size = logicutil.Page(page, size)
	items, total, err := svcCtx.Store.ListMediaAssets(ctx, page, size, query)
	if err != nil {
		return nil, err
	}
	resp := make([]types.MediaAssetResp, 0, len(items))
	for _, item := range items {
		resp = append(resp, mediaResp(item))
	}
	return &types.ListResp[types.MediaAssetResp]{Items: resp, Total: total, Page: page, Size: size}, nil
}

func Upload(ctx context.Context, svcCtx *svc.ServiceContext, originalName, altText string, data []byte) (*types.MediaAssetResp, error) {
	if err := authutil.RequireContentManager(ctx); err != nil {
		return nil, err
	}
	userID, err := authutil.UserID(ctx)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 || int64(len(data)) > svcCtx.Config.Media.EffectiveMaxUploadBytes() {
		return nil, apperrors.BadRequest("media file size is invalid")
	}
	altText = strings.TrimSpace(altText)
	if utf8.RuneCountInString(altText) > 255 {
		return nil, apperrors.BadRequest("altText is too long")
	}
	sourceName := strings.TrimSpace(originalName)
	format, config, err := validateMediaFile(sourceName, data)
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256(data)
	hash := hex.EncodeToString(sum[:])
	if existing, err := svcCtx.Store.FindMediaAssetBySHA256(ctx, hash); err == nil {
		resp := mediaResp(*existing)
		return &resp, nil
	} else if !errors.Is(err, model.ErrNotFound) {
		return nil, err
	}

	storageKey := hash + format.storageExtension
	root, err := mediaRoot(svcCtx)
	if err != nil {
		return nil, err
	}
	recoverMediaStaging(ctx, svcCtx, root)
	stagedPath, err := stageUpload(root, storageKey, data)
	if err != nil {
		return nil, err
	}
	cleanupStaged := true
	defer func() {
		if cleanupStaged {
			_ = os.Remove(stagedPath)
		}
	}()
	originalName = filepath.Base(sourceName)
	if originalName == "." || originalName == "" {
		originalName = storageKey
	}
	if len(originalName) > 255 {
		originalName = string([]rune(originalName)[:min(255, utf8.RuneCountInString(originalName))])
	}
	id, err := svcCtx.Store.CreateMediaAsset(ctx, model.MediaAssetCreate{
		StorageKey: storageKey, OriginalName: originalName, MIMEType: format.mimeType,
		SizeBytes: uint64(len(data)), Width: uint(config.Width), Height: uint(config.Height),
		AltText: altText, SHA256: hash, CreatedBy: userID,
	})
	if err != nil {
		if logicutil.IsDuplicate(err) {
			existing, findErr := svcCtx.Store.FindMediaAssetBySHA256(ctx, hash)
			if findErr == nil {
				_ = os.Remove(stagedPath)
				resp := mediaResp(*existing)
				return &resp, nil
			}
		}
		return nil, err
	}
	if err := publishStagedUpload(root, storageKey, stagedPath); err != nil {
		if cleanupErr := svcCtx.Store.DeleteMediaAsset(ctx, id); cleanupErr != nil {
			// 元数据仍存在时保留暂存文件，后续媒体操作可按数据库记录完成发布。
			cleanupStaged = false
			logx.Errorf("rollback media metadata after publish failure failed: id=%d err=%v", id, cleanupErr)
		}
		return nil, err
	}
	cleanupStaged = false
	item, err := svcCtx.Store.FindMediaAsset(ctx, id)
	if err != nil {
		return nil, err
	}
	resp := mediaResp(*item)
	return &resp, nil
}

func validateMediaFile(originalName string, data []byte) (mediaFormat, image.Config, error) {
	detectedMIMEType := http.DetectContentType(data)
	config, detectedFormat, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		if _, ok := allowedMediaFormatForMIME(detectedMIMEType); ok {
			return mediaFormat{}, image.Config{}, apperrors.BadRequest("media content is invalid")
		}
		return mediaFormat{}, image.Config{}, apperrors.BadRequest("media type is not supported")
	}
	format, ok := allowedMediaFormats[detectedFormat]
	if !ok {
		return mediaFormat{}, image.Config{}, apperrors.BadRequest("media type is not supported")
	}
	// net/http 尚未识别 AVIF；此格式由已注册的解码器完成内容验证。
	if detectedMIMEType != format.mimeType && !(detectedFormat == "avif" && detectedMIMEType == "application/octet-stream") {
		return mediaFormat{}, image.Config{}, apperrors.BadRequest("media content is invalid")
	}
	sourceName := strings.TrimSpace(originalName)
	sourceExtension := strings.ToLower(filepath.Ext(sourceName))
	if _, ok := format.sourceExtensions[sourceExtension]; !ok {
		return mediaFormat{}, image.Config{}, apperrors.BadRequest("media extension does not match its content")
	}
	if config.Width <= 0 || config.Height <= 0 {
		return mediaFormat{}, image.Config{}, apperrors.BadRequest("media content is invalid")
	}
	return format, config, nil
}

func allowedMediaFormatForMIME(mimeType string) (mediaFormat, bool) {
	for _, format := range allowedMediaFormats {
		if format.mimeType == mimeType {
			return format, true
		}
	}
	return mediaFormat{}, false
}

func Update(ctx context.Context, svcCtx *svc.ServiceContext, id uint64, req types.UpdateMediaReq) (*types.MediaAssetResp, error) {
	if err := authutil.RequireContentManager(ctx); err != nil {
		return nil, err
	}
	req.AltText = strings.TrimSpace(req.AltText)
	if utf8.RuneCountInString(req.AltText) > 255 {
		return nil, apperrors.BadRequest("altText is too long")
	}
	if err := svcCtx.Store.UpdateMediaAlt(ctx, id, req.AltText); err != nil {
		return nil, logicutil.MapError(err)
	}
	item, err := svcCtx.Store.FindMediaAsset(ctx, id)
	if err != nil {
		return nil, logicutil.MapError(err)
	}
	resp := mediaResp(*item)
	return &resp, nil
}

func Delete(ctx context.Context, svcCtx *svc.ServiceContext, id uint64) error {
	if err := authutil.RequireAdmin(ctx); err != nil {
		return err
	}
	root, err := mediaRoot(svcCtx)
	if err != nil {
		return err
	}
	recoverMediaStaging(ctx, svcCtx, root)
	item, err := svcCtx.Store.FindMediaAsset(ctx, id)
	if err != nil {
		return logicutil.MapError(err)
	}
	referenced, err := svcCtx.Store.MediaURLReferenced(ctx, mediaURL(item.StorageKey))
	if err != nil {
		return err
	}
	if referenced {
		return apperrors.Conflict("media asset is still referenced")
	}
	target := filepath.Join(root, item.StorageKey)
	quarantine := filepath.Join(root, fmt.Sprintf(".delete-%d-%s", item.ID, item.StorageKey))
	moved := false
	if err := os.Rename(target, quarantine); err == nil {
		moved = true
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("quarantine media file: %w", err)
	}
	if err := svcCtx.Store.DeleteMediaAsset(ctx, id); err != nil {
		if moved {
			if restoreErr := os.Rename(quarantine, target); restoreErr != nil {
				logx.Errorf("restore quarantined media after database failure failed: id=%d err=%v", id, restoreErr)
			}
		}
		return logicutil.MapError(err)
	}
	if moved {
		if err := os.Remove(quarantine); err != nil && !errors.Is(err, os.ErrNotExist) {
			logx.Errorf("remove quarantined media failed; it will be retried: id=%d err=%v", id, err)
		}
	}
	return nil
}

func Root(svcCtx *svc.ServiceContext) (string, error) { return mediaRoot(svcCtx) }

// RootPath resolves the configured media root without creating it. Recovery
// uses this form because a crash may have already moved the root aside and an
// eager MkdirAll would obscure the journal's publication state.
func RootPath(svcCtx *svc.ServiceContext) (string, error) {
	if svcCtx == nil {
		return "", fmt.Errorf("media service context is nil")
	}
	return mediaRootPath(svcCtx.Config.Media.EffectiveRootDir())
}

func RestoreFile(svcCtx *svc.ServiceContext, key string, data []byte) error {
	return RestoreReader(svcCtx, key, bytes.NewReader(data), int64(len(data)))
}

func RestoreReader(svcCtx *svc.ServiceContext, key string, reader io.Reader, size int64) error {
	if !mediaStorageKeyPattern.MatchString(key) || size <= 0 {
		return apperrors.BadRequest("media restore entry is invalid")
	}
	root, err := mediaRoot(svcCtx)
	if err != nil {
		return err
	}
	return writeAtomicallyReader(root, key, reader, size)
}

func mediaRoot(svcCtx *svc.ServiceContext) (string, error) {
	root, err := mediaRootPath(svcCtx.Config.Media.EffectiveRootDir())
	if err != nil {
		return "", err
	}
	return ensureMediaRoot(root)
}

func mediaRootPath(root string) (string, error) {
	return filepath.Abs(root)
}

func ensureMediaRoot(root string) (string, error) {
	if err := os.MkdirAll(root, 0755); err != nil {
		return "", fmt.Errorf("create media directory: %w", err)
	}
	return root, nil
}

func writeAtomically(root, key string, data []byte) error {
	return writeAtomicallyReader(root, key, bytes.NewReader(data), int64(len(data)))
}

func stageUpload(root, key string, data []byte) (string, error) {
	if !mediaStorageKeyPattern.MatchString(key) || len(data) == 0 {
		return "", apperrors.BadRequest("media storage key is invalid")
	}
	tmp, err := os.CreateTemp(root, ".upload-"+key+"-")
	if err != nil {
		return "", err
	}
	path := tmp.Name()
	cleanup := true
	defer func() {
		_ = tmp.Close()
		if cleanup {
			_ = os.Remove(path)
		}
	}()
	// 暂存路径由 Nginx 明确拒绝访问；文件发布后需允许只读 Web 容器读取。
	if err := tmp.Chmod(0644); err != nil {
		return "", err
	}
	if _, err := tmp.Write(data); err != nil {
		return "", err
	}
	if err := tmp.Sync(); err != nil {
		return "", err
	}
	if err := tmp.Close(); err != nil {
		return "", err
	}
	cleanup = false
	return path, nil
}

func publishStagedUpload(root, key, stagedPath string) error {
	target := filepath.Join(root, key)
	if existingHash, err := fileSHA256(target); err == nil && existingHash == strings.SplitN(key, ".", 2)[0] {
		return os.Remove(stagedPath)
	}
	if err := os.Remove(target); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("replace corrupt media file: %w", err)
	}
	if err := os.Rename(stagedPath, target); err != nil {
		return fmt.Errorf("publish media file: %w", err)
	}
	return nil
}

func recoverMediaStaging(ctx context.Context, svcCtx *svc.ServiceContext, root string) {
	entries, err := os.ReadDir(root)
	if err != nil {
		logx.Errorf("scan media staging failed: %v", err)
		return
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		path := filepath.Join(root, name)
		if match := stagedUploadPattern.FindStringSubmatch(name); len(match) == 2 {
			key := match[1]
			asset, findErr := svcCtx.Store.FindMediaAssetByStorageKey(ctx, key)
			switch {
			case findErr == nil:
				if err := publishStagedUpload(root, asset.StorageKey, path); err != nil {
					logx.Errorf("recover staged upload failed: key=%s err=%v", key, err)
				}
			case errors.Is(findErr, model.ErrNotFound):
				_ = os.Remove(path)
			}
			continue
		}
		if match := stagedDeletePattern.FindStringSubmatch(name); len(match) == 3 {
			asset, findErr := svcCtx.Store.FindMediaAssetByStorageKey(ctx, match[2])
			switch {
			case findErr == nil:
				target := filepath.Join(root, asset.StorageKey)
				if _, statErr := os.Stat(target); errors.Is(statErr, os.ErrNotExist) {
					if err := os.Rename(path, target); err != nil {
						logx.Errorf("recover quarantined media failed: key=%s err=%v", asset.StorageKey, err)
					}
				} else {
					_ = os.Remove(path)
				}
			case errors.Is(findErr, model.ErrNotFound):
				_ = os.Remove(path)
			}
		}
	}
}

func writeAtomicallyReader(root, key string, reader io.Reader, size int64) error {
	if filepath.Base(key) != key || size <= 0 {
		return apperrors.BadRequest("media storage key is invalid")
	}
	tmp, err := os.CreateTemp(root, ".upload-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(0644); err != nil {
		_ = tmp.Close()
		return err
	}
	hash := sha256.New()
	written, err := io.Copy(io.MultiWriter(tmp, hash), io.LimitReader(reader, size+1))
	if err != nil {
		_ = tmp.Close()
		return err
	}
	if written != size {
		_ = tmp.Close()
		return apperrors.BadRequest("media file size is invalid")
	}
	expectedHash := strings.SplitN(key, ".", 2)[0]
	if hex.EncodeToString(hash.Sum(nil)) != expectedHash {
		_ = tmp.Close()
		return apperrors.BadRequest("media checksum mismatch")
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	target := filepath.Join(root, key)
	if _, err := os.Stat(target); err == nil {
		existingHash, hashErr := fileSHA256(target)
		if hashErr == nil && existingHash == expectedHash {
			return nil
		}
		if err := os.Remove(target); err != nil {
			return err
		}
	}
	if err := os.Rename(tmpName, target); err != nil {
		if existingHash, hashErr := fileSHA256(target); hashErr == nil && existingHash == expectedHash {
			return nil
		}
		return err
	}
	return nil
}

func fileSHA256(filePath string) (string, error) {
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

func mediaResp(item model.MediaAsset) types.MediaAssetResp {
	return types.MediaAssetResp{
		ID: item.ID, StorageKey: item.StorageKey, URL: mediaURL(item.StorageKey), OriginalName: item.OriginalName,
		MIMEType: item.MIMEType, SizeBytes: item.SizeBytes, Width: item.Width, Height: item.Height,
		AltText: item.AltText, SHA256: item.SHA256, CreatedBy: item.CreatedBy,
		CreatedAt: item.CreatedAt.UTC(), UpdatedAt: item.UpdatedAt.UTC(),
	}
}

func mediaURL(key string) string { return "/media/" + key }
