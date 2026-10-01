//go:build integration

package integration

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"mime/multipart"
	"net/http"
	"sync"
	"testing"
)

func TestCoreConcurrentMediaUploads(t *testing.T) {
	h := newHarness(t)
	admin := h.registerUser(t, "concurrentmedia")
	// 延长真实数据库插入窗口，稳定覆盖“文件已暂存、元数据尚未提交”。
	// 不替换服务，也不改变返回值；12 个请求均必须首次成功。
	if _, err := h.rootDB.ExecContext(t.Context(), `CREATE TRIGGER e2e_slow_media_insert BEFORE INSERT ON media_assets FOR EACH ROW SET @e2e_media_sleep = SLEEP(0.1)`); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := h.rootDB.Exec(`DROP TRIGGER IF EXISTS e2e_slow_media_insert`); err != nil {
			t.Error(err)
		}
	})
	for _, duplicate := range []bool{false, true} {
		t.Run(fmt.Sprintf("duplicate=%t", duplicate), func(t *testing.T) {
			const count = 12
			data := make([][]byte, count)
			for i := range count {
				img := image.NewRGBA(image.Rect(0, 0, 128, 128))
				shade := uint8(i + 1)
				if duplicate {
					shade = 99
				}
				img.Set(0, 0, color.RGBA{R: shade, G: 150, A: 255})
				var buffer bytes.Buffer
				if err := png.Encode(&buffer, img); err != nil {
					t.Fatal(err)
				}
				data[i] = buffer.Bytes()
			}
			results := make([]*apiResult, count)
			errs := make([]error, count)
			start := make(chan struct{})
			var wg sync.WaitGroup
			for i := range count {
				wg.Go(func() {
					<-start
					var body bytes.Buffer
					writer := multipart.NewWriter(&body)
					part, err := writer.CreateFormFile("file", "concurrent.png")
					if err != nil {
						errs[i] = err
						return
					}
					if _, err := part.Write(data[i]); err != nil {
						errs[i] = err
						return
					}
					if err := writer.Close(); err != nil {
						errs[i] = err
						return
					}
					headers := http.Header{"Content-Type": []string{writer.FormDataContentType()}}
					result, err := h.request(t.Context(), http.MethodPost, h.apiURL("/api/v1/admin/media"), &body, headers, admin.accessToken, "")
					results[i], errs[i] = &result, err
				})
			}
			close(start)
			wg.Wait()
			ids := make(map[uint64]struct{})
			for i := range count {
				if errs[i] != nil {
					t.Fatalf("upload %d: %v", i, errs[i])
				}
				result := *results[i]
				expectAPISuccess(t, result)
				asset := decodeData[mediaAsset](t, result)
				h.assertMedia(t, asset, data[i])
				ids[asset.ID] = struct{}{}
			}
			want := count
			if duplicate {
				want = 1
			}
			if len(ids) != want {
				t.Fatalf("unique assets = %d, want %d", len(ids), want)
			}
		})
	}
}
