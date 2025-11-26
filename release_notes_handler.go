package main

import (
	"context"
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"dify-cli-gateway/release_notes"
)

// ReleaseNotesHandler handles HTTP requests for release notes
type ReleaseNotesHandler struct {
	service *release_notes.ReleaseNotesService
}

// NewReleaseNotesHandler creates a new release notes handler
func NewReleaseNotesHandler(service *release_notes.ReleaseNotesService) *ReleaseNotesHandler {
	return &ReleaseNotesHandler{service: service}
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error         string   `json:"error"`
	Message       string   `json:"message,omitempty"`
	SupportedCLIs []string `json:"supported_clis,omitempty"`
}

// HandleGetAllReleaseNotes handles GET /release-notes
// Returns release notes for all supported CLI tools
func (h *ReleaseNotesHandler) HandleGetAllReleaseNotes(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	log.Printf("📥 Received request: %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)

	if r.Method != http.MethodGet {
		log.Printf("❌ Method not allowed: %s", r.Method)
		h.writeError(w, http.StatusMethodNotAllowed, "Method not allowed", "Only GET method is supported")
		return
	}

	// Parse query parameters
	includeLocal := r.URL.Query().Get("include_local") == "true"
	forceRefresh := r.URL.Query().Get("force_refresh") == "true"

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	// Force refresh if requested
	if forceRefresh {
		log.Println("🔄 Force refresh requested")
		if err := h.service.Refresh(ctx, true); err != nil {
			log.Printf("⚠️ Force refresh failed: %v", err)
			// Continue with cached data if available
		}
	}

	// Get all release notes
	allNotes, err := h.service.GetAll(ctx, includeLocal)
	if err != nil {
		log.Printf("❌ Failed to get release notes: %v", err)
		h.writeError(w, http.StatusServiceUnavailable, "Service unavailable", err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, allNotes)
	log.Printf("✅ Response sent successfully (took %v)", time.Since(startTime))
}

// HandleGetCLIReleaseNotes handles GET /release-notes/{cli_name}
// Returns release notes for a specific CLI tool
func (h *ReleaseNotesHandler) HandleGetCLIReleaseNotes(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	log.Printf("📥 Received request: %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)

	if r.Method != http.MethodGet {
		log.Printf("❌ Method not allowed: %s", r.Method)
		h.writeError(w, http.StatusMethodNotAllowed, "Method not allowed", "Only GET method is supported")
		return
	}

	// Extract CLI name from path: /release-notes/{cli_name}
	path := strings.TrimPrefix(r.URL.Path, "/release-notes/")
	cliName := strings.TrimSuffix(path, "/")

	// Handle /release-notes/view separately (for HTML viewer)
	if cliName == "view" {
		// This will be handled by a separate handler in task 11
		h.writeError(w, http.StatusNotFound, "Not found", "HTML viewer not yet implemented")
		return
	}

	// Validate CLI name
	if !release_notes.IsValidCLI(cliName) {
		log.Printf("❌ Invalid CLI name: %s", cliName)
		h.writeErrorWithCLIs(w, http.StatusBadRequest, "Invalid CLI name",
			"CLI '"+cliName+"' is not supported", release_notes.SupportedCLIs)
		return
	}

	// Parse query parameters
	includeLocal := r.URL.Query().Get("include_local") == "true"
	forceRefresh := r.URL.Query().Get("force_refresh") == "true"

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	// Get release notes for specific CLI
	cliNotes, err := h.service.GetByCLI(ctx, cliName, includeLocal, forceRefresh)
	if err != nil {
		log.Printf("❌ Failed to get release notes for %s: %v", cliName, err)
		h.writeError(w, http.StatusServiceUnavailable, "Service unavailable",
			"Failed to fetch release notes for "+cliName+": "+err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, cliNotes)
	log.Printf("✅ Response sent successfully for %s (took %v)", cliName, time.Since(startTime))
}

// writeJSON writes a JSON response
func (h *ReleaseNotesHandler) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("❌ Failed to encode JSON response: %v", err)
	}
}

// writeError writes an error response
func (h *ReleaseNotesHandler) writeError(w http.ResponseWriter, status int, error, message string) {
	h.writeJSON(w, status, ErrorResponse{
		Error:   error,
		Message: message,
	})
}

// writeErrorWithCLIs writes an error response with supported CLIs list
func (h *ReleaseNotesHandler) writeErrorWithCLIs(w http.ResponseWriter, status int, error, message string, clis []string) {
	h.writeJSON(w, status, ErrorResponse{
		Error:         error,
		Message:       message,
		SupportedCLIs: clis,
	})
}

// HandleReleaseNotesView handles GET /release-notes/view
// Returns HTML page for viewing release notes
func (h *ReleaseNotesHandler) HandleReleaseNotesView(w http.ResponseWriter, r *http.Request) {
	log.Printf("📥 Received request: %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)

	if r.Method != http.MethodGet {
		h.writeError(w, http.StatusMethodNotAllowed, "Method not allowed", "Only GET method is supported")
		return
	}

	// Find template file
	templatePath := filepath.Join("templates", "release_notes.html")
	if _, err := os.Stat(templatePath); os.IsNotExist(err) {
		log.Printf("❌ Template file not found: %s", templatePath)
		h.writeError(w, http.StatusInternalServerError, "Template not found", "HTML template file is missing")
		return
	}

	// Parse and execute template
	tmpl, err := template.ParseFiles(templatePath)
	if err != nil {
		log.Printf("❌ Failed to parse template: %v", err)
		h.writeError(w, http.StatusInternalServerError, "Template error", err.Error())
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.Execute(w, nil); err != nil {
		log.Printf("❌ Failed to execute template: %v", err)
	}
	log.Println("✅ HTML view served successfully")
}
