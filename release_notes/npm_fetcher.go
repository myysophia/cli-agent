package release_notes

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"sort"
	"strings"
	"time"
)

// NPMPackageInfo represents the NPM registry API response
type NPMPackageInfo struct {
	Name     string                       `json:"name"`
	DistTags map[string]string            `json:"dist-tags"`
	Time     map[string]string            `json:"time"`
	Versions map[string]NPMVersionInfo    `json:"versions"`
}

// NPMVersionInfo represents version-specific information from NPM
type NPMVersionInfo struct {
	Version     string `json:"version"`
	Description string `json:"description"`
	Homepage    string `json:"homepage"`
}

// NPMFetcherConfig holds configuration for NPM fetcher
type NPMFetcherConfig struct {
	PackageName    string        // NPM package name (e.g., "@anthropic-ai/claude-code")
	CLIName        string        // CLI tool name
	DisplayName    string        // Display name
	VersionCommand string        // Command to get local version (e.g., "claude --version")
	Timeout        time.Duration // HTTP timeout
	MaxRetries     int           // Maximum retry attempts
}

// NPMFetcher fetches release notes from NPM Registry API
type NPMFetcher struct {
	config     NPMFetcherConfig
	httpClient *http.Client
}

// NewNPMFetcher creates a new NPM fetcher with the given configuration
func NewNPMFetcher(config NPMFetcherConfig) *NPMFetcher {
	if config.Timeout <= 0 {
		config.Timeout = 30 * time.Second
	}
	if config.MaxRetries <= 0 {
		config.MaxRetries = 3
	}

	return &NPMFetcher{
		config: config,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
	}
}


// CLIName returns the CLI tool name
func (f *NPMFetcher) CLIName() string {
	return f.config.CLIName
}

// DisplayName returns the display name
func (f *NPMFetcher) DisplayName() string {
	return f.config.DisplayName
}

// Fetch fetches release notes from NPM Registry API
func (f *NPMFetcher) Fetch(ctx context.Context) (*CLIReleaseNotes, error) {
	url := fmt.Sprintf("https://registry.npmjs.org/%s", f.config.PackageName)

	var packageInfo *NPMPackageInfo
	var lastErr error

	// Retry logic with exponential backoff
	for attempt := 0; attempt < f.config.MaxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff: 1s, 2s, 4s...
			backoff := time.Duration(1<<uint(attempt-1)) * time.Second
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
			}
		}

		packageInfo, lastErr = f.fetchPackageInfo(ctx, url)
		if lastErr == nil {
			break
		}
	}

	if lastErr != nil {
		return nil, fmt.Errorf("failed to fetch package info after %d attempts: %w", f.config.MaxRetries, lastErr)
	}

	return f.parsePackageInfo(packageInfo), nil
}

// fetchPackageInfo performs a single HTTP request to fetch package info
func (f *NPMFetcher) fetchPackageInfo(ctx context.Context, url string) (*NPMPackageInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "CLI-Gateway-Release-Notes-Fetcher")

	resp, err := f.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("NPM registry returned status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var packageInfo NPMPackageInfo
	if err := json.Unmarshal(body, &packageInfo); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	return &packageInfo, nil
}


// parsePackageInfo converts NPM package info to CLIReleaseNotes
func (f *NPMFetcher) parsePackageInfo(packageInfo *NPMPackageInfo) *CLIReleaseNotes {
	// Initialize as empty slice (not nil) to ensure JSON marshals to [] instead of null
	releaseNotes := make([]ReleaseNote, 0)

	// Get latest version from dist-tags
	latestVersion := ""
	if latest, ok := packageInfo.DistTags["latest"]; ok {
		latestVersion = latest
	}

	// Collect all versions with their timestamps
	type versionTime struct {
		version string
		time    time.Time
	}
	var versions []versionTime

	for version, timeStr := range packageInfo.Time {
		// Skip special keys like "created" and "modified"
		if version == "created" || version == "modified" {
			continue
		}

		t, err := time.Parse(time.RFC3339, timeStr)
		if err != nil {
			continue
		}

		versions = append(versions, versionTime{
			version: version,
			time:    t,
		})
	}

	// Sort by time descending (newest first)
	sort.Slice(versions, func(i, j int) bool {
		return versions[i].time.After(versions[j].time)
	})

	// Get homepage URL from the latest version info
	homepage := ""
	if versionInfo, ok := packageInfo.Versions[latestVersion]; ok {
		homepage = versionInfo.Homepage
	}

	// Convert to ReleaseNote slice
	for _, v := range versions {
		// NPM doesn't provide changelog content per version
		// We'll use a placeholder or empty string
		changelog := ""

		releaseURL := homepage
		if releaseURL == "" {
			releaseURL = fmt.Sprintf("https://www.npmjs.com/package/%s/v/%s", f.config.PackageName, v.version)
		}

		releaseNotes = append(releaseNotes, ReleaseNote{
			Version:     v.version,
			ReleaseDate: v.time,
			Changelog:   changelog,
			URL:         releaseURL,
		})
	}

	return &CLIReleaseNotes{
		CLIName:       f.config.CLIName,
		DisplayName:   f.config.DisplayName,
		LatestVersion: latestVersion,
		LastUpdated:   time.Now(),
		Releases:      releaseNotes,
	}
}

// GetLocalVersion gets the locally installed version by running the version command
func (f *NPMFetcher) GetLocalVersion() (string, error) {
	if f.config.VersionCommand == "" {
		return "", fmt.Errorf("version command not configured")
	}

	parts := strings.Fields(f.config.VersionCommand)
	if len(parts) == 0 {
		return "", fmt.Errorf("invalid version command")
	}

	cmd := exec.Command(parts[0], parts[1:]...)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to execute version command: %w", err)
	}

	version := strings.TrimSpace(string(output))
	return normalizeVersion(version), nil
}
