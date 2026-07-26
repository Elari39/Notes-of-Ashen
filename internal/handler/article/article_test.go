package article

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"notes-of-ashen/internal/authutil"
)

func TestAIAssistHandlerRejectsOversizedRequestBody(t *testing.T) {
	body := []byte(`{"action":"metadata","content":"` + strings.Repeat("x", int(maxAIAssistRequestBodySize)) + `"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/articles/ai/assist", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	AIAssistHandler(nil).ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}

type countingReadCloser struct {
	io.Reader
	reads int
}

func (r *countingReadCloser) Read(p []byte) (int, error) {
	r.reads++
	return r.Reader.Read(p)
}

func (r *countingReadCloser) Close() error { return nil }

func TestImportMarkdownHandlerChecksPermissionBeforeParsingMultipart(t *testing.T) {
	body := &countingReadCloser{Reader: strings.NewReader("not parsed")}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/articles/import", body)
	req = req.WithContext(authutil.WithUser(req.Context(), 7, authutil.RoleUser))
	req.Header.Set("Content-Type", "multipart/form-data; boundary=invalid")
	rec := httptest.NewRecorder()

	ImportMarkdownHandler(nil).ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusForbidden, rec.Body.String())
	}
	if body.reads != 0 {
		t.Fatalf("multipart body was read %d times before permission rejection", body.reads)
	}
}

func TestParseMarkdownUploadRemovesMultipartTemporaryFile(t *testing.T) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "large.md")
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := part.Write(bytes.Repeat([]byte("x"), maxMarkdownUploadBytes+1)); err != nil {
		t.Fatalf("write form file: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/articles/import", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rec := httptest.NewRecorder()

	_, _, err = parseMarkdownUpload(rec, req)
	if err == nil || !strings.Contains(err.Error(), "markdown file is too large") {
		t.Fatalf("parseMarkdownUpload() error = %v, want oversized markdown error", err)
	}
	if req.MultipartForm == nil || len(req.MultipartForm.File["file"]) != 1 {
		t.Fatal("multipart form file metadata missing")
	}
	if _, openErr := req.MultipartForm.File["file"][0].Open(); openErr == nil {
		t.Fatal("multipart temporary file remains openable after parser returned")
	}
}
