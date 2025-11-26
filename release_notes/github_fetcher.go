package release_notes

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"
	"time"
)

// GitHubRelease represents a release from GitHub API
type GitHubRelease struct {
	TagName     string    `json:"tag_name"`
	Name        string    `json:"name"`
	Body        string    `json:"body"`
	PublishedAt time.Time `json:"published_at"`
	HTMLURL     string    `json:"html_url"`
	Draft       bool      `json:"draft"`
	Prerelease  bool      `json:"prerelease"`
}

// GitHubFetcherConfig holds configuration for GitHub fetcher
type GitHubFetcherConfig struct {
	Owner          string        // Repository owner
	Repo           string        // Repository name
	CLIName        string        // CLI tool name
	DisplayName    string        // Display name
	VersionCommand string        // Command to get local version (e.g., "codex --version")
	Timeout        time.Duration // HTTP timeout
	MaxRetries     int           // Maximum retry attempts
}

// GitHubFetcher fetches release notes from GitHub Releases API
type GitHubFetcher struct {
	config     GitHubFetcherConfig
	httpClient *http.Client
}

// NewGitHubFetcher creates a new GitHub fetcher with the given configuration
func NewGitHubFetcher(config GitHubFetcherConfig) *GitHubFetcher {
	if config.Timeout <= 0 {
		config.Timeout = 30 * time.Second
	}
	if config.MaxRetries <= 0 {
		config.MaxRetries = 3
	}

	return &GitHubFetcher{
		config: config,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
	}
}


// CLIName returns the CLI tool name
func (f *GitHubFetcher) CLIName() string {
	return f.config.CLIName
}

// DisplayName returns the display name
func (f *GitHubFetcher) DisplayName() string {
	return f.config.DisplayName
}

// Fetch fetches release notes from GitHub Releases API
func (f *GitHubFetcher) Fetch(ctx context.Context) (*CLIReleaseNotes, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases", f.config.Owner, f.config.Repo)

	var releases []GitHubRelease
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

		releases, lastErr = f.fetchReleases(ctx, url)
		if lastErr == nil {
			break
		}
	}

	if lastErr != nil {
		return nil, fmt.Errorf("failed to fetch releases after %d attempts: %w", f.config.MaxRetries, lastErr)
	}

	return f.parseReleases(releases), nil
}

// fetchReleases performs a single HTTP request to fetch releases
func (f *GitHubFetcher) fetchReleases(ctx context.Context, url string) ([]GitHubRelease, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "CLI-Gateway-Release-Notes-Fetcher")

	resp, err := f.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API returned status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var releases []GitHubRelease
	if err := json.Unmarshal(body, &releases); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	return releases, nil
}

// parseReleases converts GitHub releases to CLIReleaseNotes
func (f *GitHubFetcher) parseReleases(releases []GitHubRelease) *CLIReleaseNotes {
	// Initialize as empty slice (not nil) to ensure JSON marshals to [] instead of null
	releaseNotes := make([]ReleaseNote, 0)
	var latestVersion string

	for _, release := range releases {
		// Skip drafts and prereleases
		if release.Draft || release.Prerelease {
			continue
		}

		version := normalizeVersion(release.TagName)

		// Skip non-stable versions (nightly, preview, alpha, beta, rc, etc.)
		// These are often not marked as prerelease in GitHub but should be filtered
		lowerVersion := strings.ToLower(version)
		if strings.Contains(lowerVersion, "nightly") ||
			strings.Contains(lowerVersion, "preview") ||
			strings.Contains(lowerVersion, "alpha") ||
			strings.Contains(lowerVersion, "beta") ||
			strings.Contains(lowerVersion, "rc") {
			continue
		}

		// Set latest version from the first stable release
		if latestVersion == "" {
			latestVersion = version
		}

		releaseNotes = append(releaseNotes, ReleaseNote{
			Version:     version,
			ReleaseDate: release.PublishedAt,
			Changelog:   release.Body,
			URL:         release.HTMLURL,
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
func (f *GitHubFetcher) GetLocalVersion() (string, error) {
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

// normalizeVersion removes common prefixes like "v", "rust-v" from version strings
func normalizeVersion(version string) string {
	version = strings.TrimSpace(version)
	// Remove common prefixes
	version = strings.TrimPrefix(version, "rust-v")
	version = strings.TrimPrefix(version, "v")
	return version
}
