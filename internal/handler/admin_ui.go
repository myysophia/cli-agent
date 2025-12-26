package handler

import (
	"crypto/subtle"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"mime"
	"net/http"
	"os"
	"path"
	"strings"
	"time"
)

//go:embed admin_ui_static/**
var adminUIEmbedFS embed.FS

type adminUIHandler struct {
	basePath    string
	token       string
	fsys        fs.FS
	cacheMaxAge time.Duration
}

func NewAdminUIHandler(cfg AdminUIConfig) (http.Handler, error) {
	basePath := normalizeBasePath(cfg.BasePath)
	fsys, err := adminUIStaticFS(cfg.StaticDir)
	if err != nil {
		return nil, err
	}
	cacheMaxAge := time.Duration(cfg.CacheMaxAgeSeconds) * time.Second
	if cacheMaxAge <= 0 {
		cacheMaxAge = time.Hour
	}

	return &adminUIHandler{
		basePath:    basePath,
		token:       cfg.Token,
		fsys:        fsys,
		cacheMaxAge: cacheMaxAge,
	}, nil
}

func adminUIStaticFS(staticDir string) (fs.FS, error) {
	if staticDir != "" {
		return osDirFS(staticDir)
	}
	return fs.Sub(adminUIEmbedFS, "admin_ui_static")
}

func normalizeBasePath(basePath string) string {
	if basePath == "" {
		return "/v1/admin"
	}
	if !strings.HasPrefix(basePath, "/") {
		basePath = "/" + basePath
	}
	if basePath != "/" && strings.HasSuffix(basePath, "/") {
		basePath = strings.TrimSuffix(basePath, "/")
	}
	return basePath
}

func (h *adminUIHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.URL.Path, h.basePath) {
		http.NotFound(w, r)
		return
	}

	relativePath := strings.TrimPrefix(r.URL.Path, h.basePath)
	if relativePath == "" {
		relativePath = "/"
	}

	if strings.HasPrefix(relativePath, "/api/") || relativePath == "/health" {
		h.serveAPI(w, r, relativePath)
		return
	}

	h.serveStatic(w, r, relativePath)
}

func (h *adminUIHandler) serveAPI(w http.ResponseWriter, r *http.Request, relativePath string) {
	if !h.isAuthorized(r) {
		writeUnauthorized(w)
		return
	}

	switch {
	case relativePath == "/health" || relativePath == "/api/health":
		writeJSON(w, http.StatusOK, map[string]string{
			"status": "ok",
			"time":   time.Now().Format(time.RFC3339),
		})
	case strings.HasPrefix(relativePath, "/api/mcp/"):
		handleAdminMCP(w, r, relativePath)
	case strings.HasPrefix(relativePath, "/api/config/profiles"):
		handleAdminProfiles(w, r, relativePath)
	case relativePath == "/api/config":
		switch r.Method {
		case http.MethodGet:
			handleAdminConfigGet(w, r)
		case http.MethodPost, http.MethodPut:
			handleAdminConfigUpdate(w, r)
		default:
			writeMethodNotAllowed(w)
		}
	case relativePath == "/api/config/reload":
		if r.Method != http.MethodPost {
			writeMethodNotAllowed(w)
			return
		}
		handleAdminConfigReload(w, r)
	default:
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
	}
}

func (h *adminUIHandler) serveStatic(w http.ResponseWriter, r *http.Request, relativePath string) {
	assetPath := cleanPath(relativePath)
	if strings.HasSuffix(relativePath, "/") {
		if assetPath == "" || assetPath == "." {
			assetPath = "index.html"
		} else {
			assetPath = path.Join(assetPath, "index.html")
		}
	} else if assetPath == "" || assetPath == "." {
		assetPath = "index.html"
	}

	filePath, ok := h.resolveAssetPath(assetPath)
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}

	if strings.HasSuffix(filePath, ".html") {
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	} else {
		w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d", int(h.cacheMaxAge.Seconds())))
	}

	data, err := fs.ReadFile(h.fsys, filePath)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}

	contentType := mime.TypeByExtension(path.Ext(filePath))
	if contentType == "" {
		contentType = http.DetectContentType(data)
	}
	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(data); err != nil {
		log.Printf("⚠️  Failed to write admin UI asset: %v", err)
	}
}

func (h *adminUIHandler) resolveAssetPath(assetPath string) (string, bool) {
	assetPath = strings.TrimPrefix(assetPath, "/")
	if assetPath == "" {
		assetPath = "index.html"
	}

	info, err := fs.Stat(h.fsys, assetPath)
	if err == nil {
		if info.IsDir() {
			indexPath := path.Join(assetPath, "index.html")
			if _, err := fs.Stat(h.fsys, indexPath); err == nil {
				return indexPath, true
			}
			return "index.html", true
		}
		return assetPath, true
	}

	if path.Ext(assetPath) != "" {
		return "", false
	}
	if _, err := fs.Stat(h.fsys, "index.html"); err != nil {
		return "", false
	}
	return "index.html", true
}

func (h *adminUIHandler) isAuthorized(r *http.Request) bool {
	if h.token == "" {
		log.Printf("⚠️  Admin UI token missing; request denied")
		return false
	}
	requestToken := extractAdminToken(r)
	return tokensMatch(h.token, requestToken)
}

func extractAdminToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if len(auth) > 7 && strings.EqualFold(auth[:7], "bearer ") {
		return strings.TrimSpace(auth[7:])
	}
	if token := r.Header.Get("X-Admin-Token"); token != "" {
		return token
	}
	query := r.URL.Query()
	if token := query.Get("token"); token != "" {
		return token
	}
	if token := query.Get("admin_token"); token != "" {
		return token
	}
	if cookie, err := r.Cookie("admin_token"); err == nil {
		return cookie.Value
	}
	return ""
}

func tokensMatch(expected, actual string) bool {
	if expected == "" || actual == "" {
		return false
	}
	if len(expected) != len(actual) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(expected), []byte(actual)) == 1
}

func writeUnauthorized(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("WWW-Authenticate", "Bearer")
	w.WriteHeader(http.StatusUnauthorized)
	json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
}

func writeMethodNotAllowed(w http.ResponseWriter) {
	writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Printf("⚠️  Failed to write JSON response: %v", err)
	}
}

func osDirFS(path string) (fs.FS, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("admin static dir is not a directory: %s", path)
	}
	return os.DirFS(path), nil
}

func cleanPath(p string) string {
	trimmed := strings.TrimPrefix(p, "/")
	if trimmed != "" {
		for _, segment := range strings.Split(trimmed, "/") {
			if segment == ".." {
				return "index.html"
			}
		}
	}
	cleaned := path.Clean("/" + p)
	cleaned = strings.TrimPrefix(cleaned, "/")
	if strings.HasPrefix(cleaned, "..") {
		return "index.html"
	}
	return cleaned
}
