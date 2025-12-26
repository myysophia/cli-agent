package handler

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAdminUIHealthAuth(t *testing.T) {
	handler := newAdminUIHandlerForTest(t, 120)

	req := httptest.NewRequest(http.MethodGet, "/v1/admin/health", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/v1/admin/health", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"status":"ok"`) {
		t.Fatalf("expected health payload, got %s", rec.Body.String())
	}
}

func TestAdminUIStaticFallbackAndCache(t *testing.T) {
	handler := newAdminUIHandlerForTest(t, 300)

	req := httptest.NewRequest(http.MethodGet, "/v1/admin/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (location=%s)", rec.Code, rec.Header().Get("Location"))
	}
	if !strings.Contains(rec.Body.String(), "INDEX_OK") {
		t.Fatalf("expected index content")
	}
	if cache := rec.Header().Get("Cache-Control"); cache != "no-cache, no-store, must-revalidate" {
		t.Fatalf("unexpected cache header: %s", cache)
	}

	req = httptest.NewRequest(http.MethodGet, "/v1/admin/unknown", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for fallback, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "INDEX_OK") {
		t.Fatalf("expected fallback index content")
	}

	req = httptest.NewRequest(http.MethodGet, "/v1/admin/../app.js", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for traversal guard, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "INDEX_OK") {
		t.Fatalf("expected traversal guard to return index")
	}

	req = httptest.NewRequest(http.MethodGet, "/v1/admin/missing.js", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/v1/admin/app.js", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if cache := rec.Header().Get("Cache-Control"); cache != "public, max-age=300" {
		t.Fatalf("unexpected cache header: %s", cache)
	}
	if !strings.Contains(rec.Body.String(), "APP_OK") {
		t.Fatalf("expected app content")
	}
}

func newAdminUIHandlerForTest(t *testing.T, cacheMaxAge int) http.Handler {
	t.Helper()
	staticDir := t.TempDir()

	writeTestFile(t, filepath.Join(staticDir, "index.html"), "INDEX_OK")
	writeTestFile(t, filepath.Join(staticDir, "app.js"), "APP_OK")

	handler, err := NewAdminUIHandler(AdminUIConfig{
		Token:              "test-token",
		BasePath:           "/v1/admin",
		StaticDir:          staticDir,
		CacheMaxAgeSeconds: cacheMaxAge,
	})
	if err != nil {
		t.Fatalf("failed to create admin handler: %v", err)
	}
	return handler
}

func writeTestFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
}
