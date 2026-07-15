package media

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"notes-of-ashen/internal/authutil"
	"notes-of-ashen/internal/config"
	"notes-of-ashen/internal/svc"
)

func TestUploadRejectsMismatchedOrUnsupportedContent(t *testing.T) {
	ctx := authutil.WithUser(context.Background(), 1, authutil.RoleAdmin)
	service := &svc.ServiceContext{Config: config.Config{Media: config.MediaConf{MaxUploadBytes: 1 << 20}}}
	pngData := testPNG(t)

	if _, err := Upload(ctx, service, "cover.jpg", "", pngData); err == nil {
		t.Fatal("Upload() error = nil, want extension mismatch error")
	}
	if _, err := Upload(ctx, service, "cover.png", "", []byte("not an image")); err == nil {
		t.Fatal("Upload() error = nil, want unsupported content error")
	}
	if _, err := Upload(ctx, service, "cover.png", "", nil); err == nil {
		t.Fatal("Upload() error = nil, want empty file error")
	}
}

func TestWriteAtomicallyReaderVerifiesHashAndReplacesCorruption(t *testing.T) {
	root := t.TempDir()
	data := []byte("verified media bytes")
	sum := sha256.Sum256(data)
	key := hex.EncodeToString(sum[:]) + ".jpg"
	target := filepath.Join(root, key)
	if err := os.WriteFile(target, []byte("corrupt"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := writeAtomicallyReader(root, key, bytes.NewReader(data), int64(len(data))); err != nil {
		t.Fatalf("writeAtomicallyReader() error = %v", err)
	}
	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, data) {
		t.Fatalf("restored data = %q, want %q", got, data)
	}

	if err := writeAtomicallyReader(root, "../"+key, bytes.NewReader(data), int64(len(data))); err == nil {
		t.Fatal("writeAtomicallyReader() path traversal error = nil")
	}
	badFirst := byte('0')
	if key[0] == badFirst {
		badFirst = '1'
	}
	badKey := string(badFirst) + key[1:]
	if err := writeAtomicallyReader(root, badKey, bytes.NewReader(data), int64(len(data))); err == nil {
		t.Fatal("writeAtomicallyReader() checksum error = nil")
	}
}

func testPNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.Set(0, 0, color.RGBA{R: 255, A: 255})
	var buffer bytes.Buffer
	if err := png.Encode(&buffer, img); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}
