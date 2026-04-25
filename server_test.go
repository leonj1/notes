package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestNewAppHandlerRedirectsUIRoot(t *testing.T) {
	handler := newAppHandler(t.TempDir())

	req := httptest.NewRequest(http.MethodGet, "/ui", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusMovedPermanently {
		t.Fatalf("expected status %d, got %d", http.StatusMovedPermanently, rr.Code)
	}
	if got := rr.Header().Get("Location"); got != "/ui/" {
		t.Fatalf("expected Location /ui/, got %q", got)
	}
}

func TestNewAppHandlerServesStaticFiles(t *testing.T) {
	root := t.TempDir()
	indexPath := filepath.Join(root, "index.html")
	if err := os.WriteFile(indexPath, []byte("audit ui"), 0644); err != nil {
		t.Fatalf("failed to write index file: %v", err)
	}

	handler := newAppHandler(root)
	req := httptest.NewRequest(http.MethodGet, "/ui/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}
	if body := rr.Body.String(); body != "audit ui" {
		t.Fatalf("expected body %q, got %q", "audit ui", body)
	}
}
