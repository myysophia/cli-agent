package release_notes

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// FailingFetcher is a test fetcher that always fails
type FailingFetcher struct {
	cliName     string
	displayName string
	fetchCount  atomic.Int32
}

func NewFailingFetcher(cliName string) *FailingFetcher {
	return &FailingFetcher{
		cliName:     cliName,
		displayName: cliName + " CLI",
	}
}

func (f *FailingFetcher) CLIName() string     { return f.cliName }
func (f *FailingFetcher) DisplayName() string { return f.displayName }

func (f *FailingFetcher) Fetch(ctx context.Context) (*CLIReleaseNotes, error) {
	f.fetchCount.Add(1)
	return nil, fmt.Errorf("simulated fetch failure")
}

func (f *FailingFetcher) GetLocalVersion() (string, error) {
	return "", fmt.Errorf("CLI not installed")
}

func (f *FailingFetcher) FetchCount() int {
	return int(f.fetchCount.Load())
}

// ConfigurableMockFetcher allows configuring version responses
type ConfigurableMockFetcher struct {
	cliName       string
	displayName   string
	latestVersion string
	localVersion  string
	localErr      error
	fetchCount    atomic.Int32
}

func NewConfigurableMockFetcher(cliName, latestVersion, localVersion string) *ConfigurableMockFetcher {
	return &ConfigurableMockFetcher{
		cliName:       cliName,
		displayName:   cliName + " CLI",
		latestVersion: latestVersion,
		localVersion:  localVersion,
	}
}

func (f *ConfigurableMockFetcher) CLIName() string     { return f.cliName }
func (f *ConfigurableMockFetcher) DisplayName() string { return f.displayName }

func (f *ConfigurableMockFetcher) Fetch(ctx context.Context) (*CLIReleaseNotes, error) {
	f.fetchCount.Add(1)
	return &CLIReleaseNotes{
		CLIName:       f.cliName,
		DisplayName:   f.displayName,
		LatestVersion: f.latestVersion,
		LastUpdated:   time.Now(),
		Releases: []ReleaseNote{
			{Version: f.latestVersion, ReleaseDate: time.Now(), Changelog: "Release"},
		},
	}, nil
}

func (f *ConfigurableMockFetcher) GetLocalVersion() (string, error) {
	if f.localErr != nil {
		return "", f.localErr
	}
	return f.localVersion, nil
}

func (f *ConfigurableMockFetcher) SetLocalError(err error) {
	f.localErr = err
}


// **Feature: cli-release-notes, Property 7: Version comparison correctness**
// **Validates: Requirements 4.1, 4.2**
// For any two version strings where local_version differs from latest_version,
// the update_available field SHALL be true.
func TestProperty_VersionComparisonCorrectness(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	properties.Property("update_available is true when versions differ", prop.ForAll(
		func(localVersion, latestVersion string) bool {
			// Skip empty versions as they are edge cases handled separately
			if localVersion == "" || latestVersion == "" {
				return true
			}

			result := CompareVersions(localVersion, latestVersion)

			// If versions are different, update should be available
			if localVersion != latestVersion {
				return result == true
			}
			// If versions are the same, no update available
			return result == false
		},
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 }),
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 }),
	))

	properties.TestingRun(t)
}

// **Feature: cli-release-notes, Property 7: Version comparison with empty versions**
// **Validates: Requirements 4.3**
// When local CLI is not installed (empty version), update_available SHALL be false.
func TestProperty_VersionComparisonEmptyLocal(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	properties.Property("empty local version means no update available", prop.ForAll(
		func(latestVersion string) bool {
			result := CompareVersions("", latestVersion)
			return result == false
		},
		gen.AlphaString(),
	))

	properties.TestingRun(t)
}

// **Feature: cli-release-notes, Property 11: Scheduled refresh executes at interval**
// **Validates: Requirements 8.2**
// For any configured refresh interval, the system SHALL automatically fetch
// new data after the interval passes.
func TestProperty_ScheduledRefreshExecutesAtInterval(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 20 // Fewer tests due to timing sensitivity
	properties := gopter.NewProperties(parameters)

	properties.Property("scheduled refresh executes after interval", prop.ForAll(
		func(intervalMs int) bool {
			interval := time.Duration(intervalMs) * time.Millisecond

			// Create service with short refresh interval
			// Cache TTL should be shorter than refresh interval so cache expires before refresh
			config := ServiceConfig{
				CacheTTL:        interval / 2, // Cache expires before refresh interval
				RefreshInterval: interval,
				StoragePath:     fmt.Sprintf("/tmp/test_service_%d.json", time.Now().UnixNano()),
			}
			service := NewReleaseNotesService(config)

			// Register mock fetcher
			fetcher := NewMockFetcher("test-cli")
			service.RegisterFetcher("test-cli", fetcher)

			// Start service (this triggers initial fetch)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			err := service.Start(ctx)
			if err != nil {
				return false
			}
			defer service.Stop()

			// Record count after initial startup fetch
			// Give a small delay to ensure startup fetch completes
			time.Sleep(20 * time.Millisecond)
			countAfterStart := fetcher.FetchCount()

			// Wait for at least one scheduled refresh (interval + buffer)
			// The cache will expire before the refresh interval, so the scheduler
			// will trigger a new fetch
			time.Sleep(interval + 150*time.Millisecond)

			// Should have at least one more fetch from scheduler
			countAfterWait := fetcher.FetchCount()
			return countAfterWait > countAfterStart
		},
		gen.IntRange(100, 200), // Interval between 100-200ms for more reliable timing
	))

	properties.TestingRun(t)
}

// **Feature: cli-release-notes, Property 12: Failed refresh preserves cache**
// **Validates: Requirements 8.3**
// For any automatic refresh that fails, the system SHALL retain the
// previously cached data without modification.
func TestProperty_FailedRefreshPreservesCache(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	properties.Property("failed refresh preserves existing cache", prop.ForAll(
		func(version string) bool {
			if version == "" {
				return true // Skip empty versions
			}

			// Create service
			config := ServiceConfig{
				CacheTTL:        time.Hour,
				RefreshInterval: time.Hour,
				StoragePath:     fmt.Sprintf("/tmp/test_preserve_%d.json", time.Now().UnixNano()),
			}
			service := NewReleaseNotesService(config)

			// First, populate cache with a working fetcher
			workingFetcher := NewConfigurableMockFetcher("test-cli", version, "")
			service.RegisterFetcher("test-cli", workingFetcher)

			ctx := context.Background()

			// Fetch to populate cache
			_, err := service.GetByCLI(ctx, "test-cli", false, true)
			if err != nil {
				return false
			}

			// Verify cache has data
			cached := service.GetCache().Get("test-cli")
			if cached == nil || cached.LatestVersion != version {
				return false
			}

			// Now replace with failing fetcher
			failingFetcher := NewFailingFetcher("test-cli")
			service.RegisterFetcher("test-cli", failingFetcher)

			// Try to refresh - this should fail
			_ = service.Refresh(ctx, true)

			// Cache should still have the original data
			cachedAfterFail := service.GetCache().Get("test-cli")
			if cachedAfterFail == nil {
				return false
			}

			return cachedAfterFail.LatestVersion == version
		},
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 }),
	))

	properties.TestingRun(t)
}


// Unit tests for service behavior

func TestService_GetByCLI_InvalidCLI(t *testing.T) {
	config := DefaultServiceConfig()
	config.StoragePath = fmt.Sprintf("/tmp/test_invalid_%d.json", time.Now().UnixNano())
	service := NewReleaseNotesService(config)

	_, err := service.GetByCLI(context.Background(), "invalid-cli", false, false)
	if err == nil {
		t.Error("Expected error for invalid CLI name")
	}
}

func TestService_GetByCLI_WithLocalVersion(t *testing.T) {
	config := DefaultServiceConfig()
	config.StoragePath = fmt.Sprintf("/tmp/test_local_%d.json", time.Now().UnixNano())
	service := NewReleaseNotesService(config)

	fetcher := NewConfigurableMockFetcher("test-cli", "2.0.0", "1.0.0")
	service.RegisterFetcher("test-cli", fetcher)

	result, err := service.GetByCLI(context.Background(), "test-cli", true, true)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result.LocalVersion != "1.0.0" {
		t.Errorf("Expected local version '1.0.0', got '%s'", result.LocalVersion)
	}

	if result.LatestVersion != "2.0.0" {
		t.Errorf("Expected latest version '2.0.0', got '%s'", result.LatestVersion)
	}

	if !result.UpdateAvailable {
		t.Error("Expected UpdateAvailable to be true when versions differ")
	}
}

func TestService_GetByCLI_CLINotInstalled(t *testing.T) {
	config := DefaultServiceConfig()
	config.StoragePath = fmt.Sprintf("/tmp/test_notinstalled_%d.json", time.Now().UnixNano())
	service := NewReleaseNotesService(config)

	fetcher := NewConfigurableMockFetcher("test-cli", "2.0.0", "")
	fetcher.SetLocalError(fmt.Errorf("CLI not installed"))
	service.RegisterFetcher("test-cli", fetcher)

	result, err := service.GetByCLI(context.Background(), "test-cli", true, true)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result.LocalVersion != "" {
		t.Errorf("Expected empty local version, got '%s'", result.LocalVersion)
	}

	if result.UpdateAvailable {
		t.Error("Expected UpdateAvailable to be false when CLI not installed")
	}
}

func TestService_StartStop(t *testing.T) {
	config := ServiceConfig{
		CacheTTL:        time.Hour,
		RefreshInterval: time.Hour,
		StoragePath:     fmt.Sprintf("/tmp/test_startstop_%d.json", time.Now().UnixNano()),
	}
	service := NewReleaseNotesService(config)

	// Register a mock fetcher
	fetcher := NewMockFetcher("test-cli")
	service.RegisterFetcher("test-cli", fetcher)

	ctx := context.Background()

	// Start service
	err := service.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start service: %v", err)
	}

	if !service.IsRunning() {
		t.Error("Service should be running after Start")
	}

	// Stop service
	err = service.Stop()
	if err != nil {
		t.Fatalf("Failed to stop service: %v", err)
	}

	if service.IsRunning() {
		t.Error("Service should not be running after Stop")
	}
}

func TestService_GetAll(t *testing.T) {
	config := DefaultServiceConfig()
	config.StoragePath = fmt.Sprintf("/tmp/test_getall_%d.json", time.Now().UnixNano())
	service := NewReleaseNotesService(config)

	// Register multiple fetchers
	service.RegisterFetcher("cli1", NewConfigurableMockFetcher("cli1", "1.0.0", "0.9.0"))
	service.RegisterFetcher("cli2", NewConfigurableMockFetcher("cli2", "2.0.0", "1.9.0"))

	ctx := context.Background()

	// Refresh to populate cache
	err := service.Refresh(ctx, true)
	if err != nil {
		t.Fatalf("Refresh failed: %v", err)
	}

	// Get all
	result, err := service.GetAll(ctx, true)
	if err != nil {
		t.Fatalf("GetAll failed: %v", err)
	}

	if len(result.CLIs) != 2 {
		t.Errorf("Expected 2 CLIs, got %d", len(result.CLIs))
	}

	if result.CLIs["cli1"] == nil {
		t.Error("Expected cli1 in results")
	}

	if result.CLIs["cli2"] == nil {
		t.Error("Expected cli2 in results")
	}
}

func TestIsValidCLI(t *testing.T) {
	validCLIs := []string{"claude", "codex", "cursor", "gemini", "qwen"}
	for _, cli := range validCLIs {
		if !IsValidCLI(cli) {
			t.Errorf("Expected '%s' to be valid", cli)
		}
	}

	invalidCLIs := []string{"invalid", "unknown", "test", ""}
	for _, cli := range invalidCLIs {
		if IsValidCLI(cli) {
			t.Errorf("Expected '%s' to be invalid", cli)
		}
	}
}

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		local    string
		latest   string
		expected bool
	}{
		{"1.0.0", "2.0.0", true},  // Different versions
		{"2.0.0", "2.0.0", false}, // Same versions
		{"", "2.0.0", false},      // Empty local
		{"1.0.0", "", false},      // Empty latest
		{"", "", false},           // Both empty
	}

	for _, tt := range tests {
		result := CompareVersions(tt.local, tt.latest)
		if result != tt.expected {
			t.Errorf("CompareVersions(%q, %q) = %v, expected %v",
				tt.local, tt.latest, result, tt.expected)
		}
	}
}
