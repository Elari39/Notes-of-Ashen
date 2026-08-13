package rag

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSetupSSEOnlyWritesHeadersForFlushCapableWriter(t *testing.T) {
	nonFlushing := &headerOnlyResponseWriter{header: make(http.Header)}
	if flusher, ok := setupSSE(nonFlushing); ok || flusher != nil {
		t.Fatal("setupSSE() unexpectedly accepted a non-flushing writer")
	}
	if got := nonFlushing.Header().Get("Content-Type"); got != "" {
		t.Fatalf("non-flushing writer Content-Type = %q, want empty", got)
	}

	recorder := httptest.NewRecorder()
	if flusher, ok := setupSSE(recorder); !ok || flusher == nil {
		t.Fatal("setupSSE() rejected a flushing response recorder")
	}
	if got := recorder.Header().Get("Content-Type"); got != "text/event-stream; charset=utf-8" {
		t.Fatalf("Content-Type = %q", got)
	}
	if got := recorder.Header().Get("X-Accel-Buffering"); got != "no" {
		t.Fatalf("X-Accel-Buffering = %q", got)
	}
}

func TestWriteEventReturnsFixedErrorWhenClientWriteFails(t *testing.T) {
	writer := &failingSSEWriter{header: make(http.Header)}
	err := writeEvent(writer, writer, "delta", map[string]string{"delta": "private content"})
	if !errors.Is(err, errSSEWrite) {
		t.Fatalf("writeEvent() error = %v, want errSSEWrite", err)
	}
}

type headerOnlyResponseWriter struct {
	header http.Header
}

func (w *headerOnlyResponseWriter) Header() http.Header { return w.header }
func (w *headerOnlyResponseWriter) Write(value []byte) (int, error) {
	return len(value), nil
}
func (w *headerOnlyResponseWriter) WriteHeader(int) {}

type failingSSEWriter struct {
	header http.Header
}

func (w *failingSSEWriter) Header() http.Header { return w.header }
func (w *failingSSEWriter) Write([]byte) (int, error) {
	return 0, errors.New("client closed while receiving private content")
}
func (w *failingSSEWriter) WriteHeader(int) {}
func (w *failingSSEWriter) Flush()          {}
