package media

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"notes-of-ashen/internal/config"
	"notes-of-ashen/internal/svc"
)

func TestUploadHandlerRejectsMissingFileAsBadRequest(t *testing.T) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("altText", "missing"); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/media", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rec := httptest.NewRecorder()
	UploadHandler(&svc.ServiceContext{Config: config.Config{Media: config.MediaConf{MaxUploadBytes: 1 << 20}}}).ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", rec.Code, rec.Body.String())
	}
}

func TestUploadRequestLimitIncludesMultipartOverhead(t *testing.T) {
	svcCtx := &svc.ServiceContext{Config: config.Config{Media: config.MediaConf{MaxUploadBytes: 10 << 20}}}
	if got, want := UploadRequestLimit(svcCtx), int64(11<<20); got != want {
		t.Fatalf("UploadRequestLimit() = %d, want %d", got, want)
	}
}
