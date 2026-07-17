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

	"golang.org/x/image/webp"
)

var allowedMedia = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/gif":  ".gif",
	"image/webp": ".webp",
}

var allowedSourceExtensions = map[string]map[string]struct{}{
	"image/jpeg": {".jpg": {}, ".jpeg": {}},
	"image/png":  {".png": {}},
	"image/gif":  {".gif": {}},
	"image/webp": {".webp": {}},
}

var imageFormats = map[string]string{
	"image/jpeg": "jpeg",
	"image/png":  "png",
	"image/gif":  "gif",
	"image/webp": "webp",
}

var mediaStorageKeyPattern = regexp.MustCompile(`^[a-f0-9]{64}\.(jpg|png|gif|webp)$`)

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
	mimeType := http.DetectContentType(data)
	extension, ok := allowedMedia[mimeType]
	if !ok {
		return nil, apperrors.BadRequest("media type is not supported")
	}
	sourceName := strings.TrimSpace(originalName)
	sourceExtension := strings.ToLower(filepath.Ext(sourceName))
	if _, ok := allowedSourceExtensions[mimeType][sourceExtension]; !ok {
		return nil, apperrors.BadRequest("media extension does not match its content")
	}
	config, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || format != imageFormats[mimeType] || config.Width <= 0 || config.Height <= 0 {
		return nil, apperrors.BadRequest("media content is invalid")
	}
	sum := sha256.Sum256(data)
	hash := hex.EncodeToString(sum[:])
	if existing, err := svcCtx.Store.FindMediaAssetBySHA256(ctx, hash); err == nil {
		resp := mediaResp(*existing)
		return &resp, nil
	} else if !errors.Is(err, model.ErrNotFound) {
		return nil, err
	}

	storageKey := hash + extension
	root, err := mediaRoot(svcCtx)
	if err != nil {
		return nil, err
	}
	if err := writeAtomically(root, storageKey, data); err != nil {
		return nil, err
	}
	originalName = filepath.Base(sourceName)
	if originalName == "." || originalName == "" {
		originalName = storageKey
	}
	if len(originalName) > 255 {
		originalName = string([]rune(originalName)[:min(255, utf8.RuneCountInString(originalName))])
	}
	id, err := svcCtx.Store.CreateMediaAsset(ctx, model.MediaAssetCreate{
		StorageKey: storageKey, OriginalName: originalName, MIMEType: mimeType,
		SizeBytes: uint64(len(data)), Width: uint(config.Width), Height: uint(config.Height),
		AltText: altText, SHA256: hash, CreatedBy: userID,
	})
	if err != nil {
		if logicutil.IsDuplicate(err) {
			existing, findErr := svcCtx.Store.FindMediaAssetBySHA256(ctx, hash)
			if findErr == nil {
				resp := mediaResp(*existing)
				return &resp, nil
			}
		}
		return nil, err
	}
	item, err := svcCtx.Store.FindMediaAsset(ctx, id)
	if err != nil {
		return nil, err
	}
	resp := mediaResp(*item)
	return &resp, nil
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
	if err := svcCtx.Store.DeleteMediaAsset(ctx, id); err != nil {
		return logicutil.MapError(err)
	}
	root, err := mediaRoot(svcCtx)
	if err != nil {
		return err
	}
	if err := os.Remove(filepath.Join(root, item.StorageKey)); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove media file: %w", err)
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

var _ = webp.DecodeConfig
