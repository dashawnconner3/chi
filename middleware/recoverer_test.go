package middleware

import (
	"bufio"
	"bytes"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRecoverer(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r = r.WithContext(WithLogEntry(r, &DefaultLogEntry{Logger: &DefaultLogger{Writer: &buf}}))
			next.ServeHTTP(w, r)
		})
	}

	h := logger(Recoverer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	})))

	r, _ := http.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}
	if !strings.Contains(buf.String(), "panic: test panic") {
		t.Errorf("expected panic message in log, got %q", buf.String())
	}
}

type testFlusher struct {
	http.ResponseWriter
	flushed bool
}

func (tf *testFlusher) Flush() {
	tf.flushed = true
	if f, ok := tf.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func TestRecovererFlusher(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	tf := &testFlusher{ResponseWriter: w}

	h := Recoverer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	}))

	r, _ := http.NewRequest("GET", "/", nil)
	h.ServeHTTP(tf, r)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}
	if !tf.flushed {
		t.Error("expected Flush() to be called")
	}
}
