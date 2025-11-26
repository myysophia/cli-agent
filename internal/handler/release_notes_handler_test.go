package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sort"
	"sync/atomic"
	"testing"
	"time"

	"dify-cli-gateway/internal/release_notes"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// MockFetcherForHandler is a test fetcher for handler tests
type MockFetcherForHandler struct {
	cliName       string
	displayName   string
	latestVersion string
	releases      []release_notes.ReleaseNote
	fetchErr      error
	fetchCount    atomic.Int32
}

func NewMockFetcherForHandler(cliName string, releases []release_notes.ReleaseNote) *MockFetcherForHandler {
	latestVersion := ""
	if len(releases) > 0 {
		latestVersion = releases[0].Version
	}
	return &MockFetcherForHandler{
		cliName:       cliName,
		displayName:   cliName + " CLI",
		latestVersion: latestVersion,
		releases:      releases,
	}
}

func (m *MockFetcherForHandler) CLIName() string     { return m.cliName }
func (m *MockFetcherForHandler) DisplayName() string { return m.displayName }

func (m *MockFetcherForHandler) Fetch(ctx context.Context) (*release_notes.CLIReleaseNotes, error) {
	m.fetchCount.Add(1)
	if m.fetchErr != nil {
		return nil, m.fetchErr
	}
	return &release_notes.CLIReleaseNotes{
		CLIName:       m.cliName,
		DisplayName:   m.displayName,
		LatestVersion: m.latestVersion,
		LastUpdated:   time.Now(),
		Releases:      m.releases,
	}, nil
}

func (m *MockFetcherForHandler) GetLocalVersion() (string, error) {
	return "", fmt.Errorf("CLI not installed")
}

func (m *MockFetcherForHandler) SetFetchError(err error) {
	m.fetchErr = err
}


// FailingFetcherForHandler always returns an error
type FailingFetcherForHandler struct {
	cliName     string
	displayName string
}

func NewFailingFetcherForHandler(cliName string) *FailingFetcherForHandler {
	return &FailingFetcherForHandler{
		cliName:     cliName,
		displayName: cliName + " CLI",
	}
}

func (f *FailingFetcherForHandler) CLIName() string     { return f.cliName }
func (f *FailingFetcherForHandler) DisplayName() string { return f.displayName }

func (f *FailingFetcherForHandler) Fetch(ctx context.Context) (*release_notes.CLIReleaseNotes, error) {
	return nil, fmt.Errorf("external source unavailable")
}

func (f *FailingFetcherForHandler) GetLocalVersion() (string, error) {
	return "", fmt.Errorf("CLI not installed")
}

// genReleaseNoteForHandler generates arbitrary ReleaseNote values for handler tests
func genReleaseNoteForHandler() gopter.Gen {
	return gopter.CombineGens(
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 }), // Version (non-empty)
		gen.TimeRange(time.Now().Add(-365*24*time.Hour), 365*24*time.Hour),    // ReleaseDate
		gen.AlphaString(), // Changelog
		gen.AlphaString(), // URL
	).Map(func(values []interface{}) release_notes.ReleaseNote {
		return release_notes.ReleaseNote{
			Version:     values[0].(string),
			ReleaseDate: values[1].(time.Time),
			Changelog:   values[2].(string),
			URL:         values[3].(string),
		}
	})
}

// genSortedReleases generates a slice of releases sorted by date descending
func genSortedReleases() gopter.Gen {
	return gen.SliceOfN(3, genReleaseNoteForHandler()).Map(func(releases []release_notes.ReleaseNote) []release_notes.ReleaseNote {
		// Sort by date descending
		sort.Slice(releases, func(i, j int) bool {
			return releases[i].ReleaseDate.After(releases[j].ReleaseDate)
		})
		return releases
	})
}

// createTestService creates a service with mock fetchers for testing
func createTestService(fetchers map[string]release_notes.ReleaseNoteFetcher) *release_notes.ReleaseNotesService {
	config := release_notes.ServiceConfig{
		CacheTTL:        time.Hour,
		RefreshInterval: time.Hour,
		StoragePath:     fmt.Sprintf("/tmp/test_handler_%d.json", time.Now().UnixNano()),
	}
	service := release_notes.NewReleaseNotesService(config)
	for name, fetcher := range fetchers {
		service.RegisterFetcher(name, fetcher)
	}
	return service
}


// **Feature: cli-release-notes, Property 1: API returns valid JSON with required fields**
// **Validates: Requirements 1.1, 1.2, 1.4**
// For any GET request to /release-notes or /release-notes/{cli_name}, the response
// SHALL be valid JSON containing version, release_date, and changelog fields for each release.
func TestProperty_APIReturnsValidJSONWithRequiredFields(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	properties.Property("API returns valid JSON with required fields", prop.ForAll(
		func(releases []release_notes.ReleaseNote) bool {
			if len(releases) == 0 {
				return true // Skip empty releases
			}

			// Create service with mock fetcher
			fetchers := map[string]release_notes.ReleaseNoteFetcher{
				"claude": NewMockFetcherForHandler("claude", releases),
			}
			service := createTestService(fetchers)
			ctx := context.Background()
			service.Refresh(ctx, true)

			handler := NewReleaseNotesHandler(service)

			// Test /release-notes/{cli_name} endpoint
			req := httptest.NewRequest(http.MethodGet, "/release-notes/claude", nil)
			w := httptest.NewRecorder()
			handler.HandleGetCLIReleaseNotes(w, req)

			if w.Code != http.StatusOK {
				return false
			}

			// Verify response is valid JSON
			var response release_notes.CLIReleaseNotes
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				return false
			}

			// Verify required fields exist
			if response.CLIName == "" {
				return false
			}
			if response.DisplayName == "" {
				return false
			}

			// Verify each release has required fields
			for _, rel := range response.Releases {
				if rel.Version == "" {
					return false
				}
				// ReleaseDate is a time.Time, it will always be present (zero value is valid)
				// Changelog can be empty, but field must exist (it does by struct definition)
			}

			return true
		},
		gen.SliceOfN(3, genReleaseNoteForHandler()),
	))

	properties.TestingRun(t)
}

// **Feature: cli-release-notes, Property 2: Invalid CLI name returns 400 error**
// **Validates: Requirements 1.3**
// For any string that is not a valid CLI name (claude, codex, cursor, gemini, qwen),
// a GET request to /release-notes/{invalid_name} SHALL return a 400 status code.
func TestProperty_InvalidCLINameReturns400Error(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	validCLIs := map[string]bool{
		"claude": true,
		"codex":  true,
		"cursor": true,
		"gemini": true,
		"qwen":   true,
		"view":   true, // Special case for HTML viewer
	}

	properties.Property("invalid CLI name returns 400 error", prop.ForAll(
		func(invalidName string) bool {
			// Skip if it happens to be a valid CLI name
			if validCLIs[invalidName] {
				return true
			}

			// Create service with no fetchers (doesn't matter for this test)
			service := createTestService(map[string]release_notes.ReleaseNoteFetcher{})
			handler := NewReleaseNotesHandler(service)

			req := httptest.NewRequest(http.MethodGet, "/release-notes/"+invalidName, nil)
			w := httptest.NewRecorder()
			handler.HandleGetCLIReleaseNotes(w, req)

			// Should return 400 Bad Request
			if w.Code != http.StatusBadRequest {
				return false
			}

			// Verify error response contains supported CLIs list
			var errResp ErrorResponse
			if err := json.Unmarshal(w.Body.Bytes(), &errResp); err != nil {
				return false
			}

			// Should have supported_clis in response
			return len(errResp.SupportedCLIs) > 0
		},
		gen.AlphaString().SuchThat(func(s string) bool {
			// Generate strings that are not valid CLI names
			return len(s) > 0 && !validCLIs[s]
		}),
	))

	properties.TestingRun(t)
}


// **Feature: cli-release-notes, Property 3: Unavailable source returns 503 error**
// **Validates: Requirements 2.6**
// For any CLI fetcher, when the external source is unavailable,
// the system SHALL return a 503 status code with an error message.
func TestProperty_UnavailableSourceReturns503Error(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	properties.Property("unavailable source returns 503 error", prop.ForAll(
		func(cliName string) bool {
			// Only test with valid CLI names
			validCLIs := []string{"claude", "codex", "cursor", "gemini", "qwen"}
			isValid := false
			for _, v := range validCLIs {
				if v == cliName {
					isValid = true
					break
				}
			}
			if !isValid {
				return true // Skip invalid CLI names
			}

			// Create service with failing fetcher
			fetchers := map[string]release_notes.ReleaseNoteFetcher{
				cliName: NewFailingFetcherForHandler(cliName),
			}
			service := createTestService(fetchers)
			handler := NewReleaseNotesHandler(service)

			// Request with force_refresh to ensure we try to fetch
			req := httptest.NewRequest(http.MethodGet, "/release-notes/"+cliName+"?force_refresh=true", nil)
			w := httptest.NewRecorder()
			handler.HandleGetCLIReleaseNotes(w, req)

			// Should return 503 Service Unavailable
			return w.Code == http.StatusServiceUnavailable
		},
		gen.OneConstOf("claude", "codex", "cursor", "gemini", "qwen"),
	))

	properties.TestingRun(t)
}

// **Feature: cli-release-notes, Property 10: Releases sorted by date descending**
// **Validates: Requirements 7.1**
// For any list of releases returned by the API, the releases SHALL be sorted
// in reverse chronological order (newest first).
func TestProperty_ReleasesSortedByDateDescending(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	properties.Property("releases are sorted by date descending", prop.ForAll(
		func(releases []release_notes.ReleaseNote) bool {
			if len(releases) < 2 {
				return true // Need at least 2 releases to verify sorting
			}

			// Sort releases by date descending before passing to fetcher
			// (simulating what the fetcher should do)
			sort.Slice(releases, func(i, j int) bool {
				return releases[i].ReleaseDate.After(releases[j].ReleaseDate)
			})

			// Create service with mock fetcher
			fetchers := map[string]release_notes.ReleaseNoteFetcher{
				"claude": NewMockFetcherForHandler("claude", releases),
			}
			service := createTestService(fetchers)
			ctx := context.Background()
			service.Refresh(ctx, true)

			handler := NewReleaseNotesHandler(service)

			req := httptest.NewRequest(http.MethodGet, "/release-notes/claude", nil)
			w := httptest.NewRecorder()
			handler.HandleGetCLIReleaseNotes(w, req)

			if w.Code != http.StatusOK {
				return false
			}

			var response release_notes.CLIReleaseNotes
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				return false
			}

			// Verify releases are sorted by date descending
			for i := 0; i < len(response.Releases)-1; i++ {
				current := response.Releases[i].ReleaseDate
				next := response.Releases[i+1].ReleaseDate
				// Current should be >= next (descending order)
				if current.Before(next) {
					return false
				}
			}

			return true
		},
		gen.SliceOfN(5, genReleaseNoteForHandler()),
	))

	properties.TestingRun(t)
}


// Unit tests for additional handler behavior

func TestHandler_GetAllReleaseNotes_Success(t *testing.T) {
	releases := []release_notes.ReleaseNote{
		{Version: "1.0.0", ReleaseDate: time.Now(), Changelog: "Initial release"},
	}
	fetchers := map[string]release_notes.ReleaseNoteFetcher{
		"claude": NewMockFetcherForHandler("claude", releases),
		"codex":  NewMockFetcherForHandler("codex", releases),
	}
	service := createTestService(fetchers)
	ctx := context.Background()
	service.Refresh(ctx, true)

	handler := NewReleaseNotesHandler(service)

	req := httptest.NewRequest(http.MethodGet, "/release-notes", nil)
	w := httptest.NewRecorder()
	handler.HandleGetAllReleaseNotes(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response release_notes.AllReleaseNotes
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if len(response.CLIs) != 2 {
		t.Errorf("Expected 2 CLIs, got %d", len(response.CLIs))
	}
}

func TestHandler_GetCLIReleaseNotes_MethodNotAllowed(t *testing.T) {
	service := createTestService(map[string]release_notes.ReleaseNoteFetcher{})
	handler := NewReleaseNotesHandler(service)

	req := httptest.NewRequest(http.MethodPost, "/release-notes/claude", nil)
	w := httptest.NewRecorder()
	handler.HandleGetCLIReleaseNotes(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", w.Code)
	}
}

func TestHandler_GetAllReleaseNotes_MethodNotAllowed(t *testing.T) {
	service := createTestService(map[string]release_notes.ReleaseNoteFetcher{})
	handler := NewReleaseNotesHandler(service)

	req := httptest.NewRequest(http.MethodPost, "/release-notes", nil)
	w := httptest.NewRecorder()
	handler.HandleGetAllReleaseNotes(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", w.Code)
	}
}

func TestHandler_GetCLIReleaseNotes_WithQueryParams(t *testing.T) {
	releases := []release_notes.ReleaseNote{
		{Version: "2.0.0", ReleaseDate: time.Now(), Changelog: "New release"},
	}
	fetchers := map[string]release_notes.ReleaseNoteFetcher{
		"claude": NewMockFetcherForHandler("claude", releases),
	}
	service := createTestService(fetchers)
	ctx := context.Background()
	service.Refresh(ctx, true)

	handler := NewReleaseNotesHandler(service)

	// Test with include_local=true
	req := httptest.NewRequest(http.MethodGet, "/release-notes/claude?include_local=true", nil)
	w := httptest.NewRecorder()
	handler.HandleGetCLIReleaseNotes(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response release_notes.CLIReleaseNotes
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	// LocalVersion should be empty since mock returns error for GetLocalVersion
	if response.LocalVersion != "" {
		t.Errorf("Expected empty local version, got %s", response.LocalVersion)
	}
}

func TestHandler_InvalidCLIName_ReturnsListOfSupported(t *testing.T) {
	service := createTestService(map[string]release_notes.ReleaseNoteFetcher{})
	handler := NewReleaseNotesHandler(service)

	req := httptest.NewRequest(http.MethodGet, "/release-notes/invalid-cli", nil)
	w := httptest.NewRecorder()
	handler.HandleGetCLIReleaseNotes(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}

	var errResp ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("Failed to unmarshal error response: %v", err)
	}

	if len(errResp.SupportedCLIs) == 0 {
		t.Error("Expected supported CLIs list in error response")
	}

	// Verify all supported CLIs are listed
	expectedCLIs := []string{"claude", "codex", "cursor", "gemini", "qwen"}
	for _, expected := range expectedCLIs {
		found := false
		for _, cli := range errResp.SupportedCLIs {
			if cli == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected %s in supported CLIs list", expected)
		}
	}
}
