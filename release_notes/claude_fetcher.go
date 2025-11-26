package release_notes

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

// ClaudeFetcherConfig holds configuration for Claude fetcher
type ClaudeFetcherConfig struct {
	ChangelogURL   string        // GitHub CHANGELOG.md URL
	CLIName        string        // CLI tool name
	DisplayName    string        // Display name
	VersionCommand string        // Command to get local version
	Timeout        time.Duration // HTTP timeout
	MaxRetries     int           // Maximum retry attempts
}

// ClaudeFetcher fetches release notes from Claude Code GitHub CHANGELOG.md
type ClaudeFetcher struct {
	config     ClaudeFetcherConfig
	httpClient *http.Client
}

// NewClaudeFetcher creates a new Claude fetcher
func NewClaudeFetcher() *ClaudeFetcher {
	config := ClaudeFetcherConfig{
		ChangelogURL:   "https://raw.githubusercontent.com/anthropics/claude-code/main/CHANGELOG.md",
		CLIName:        "claude",
		DisplayName:    "Claude CLI",
		VersionCommand: "claude --version",
		Timeout:        30 * time.Second,
		MaxRetries:     3,
	}

	return &ClaudeFetcher{
		config: config,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

// CLIName returns the CLI tool name
func (f *ClaudeFetcher) CLIName() string {
	return f.config.CLIName
}

// DisplayName returns the display name
func (f *ClaudeFetcher) DisplayName() string {
	return f.config.DisplayName
}

// Fetch fetches release notes from GitHub CHANGELOG.md
func (f *ClaudeFetcher) Fetch(ctx context.Context) (*CLIReleaseNotes, error) {
	var changelog string
	var lastErr error

	// Retry logic with exponential backoff
	for attempt := 0; attempt < f.config.MaxRetries; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(1<<uint(attempt-1)) * time.Second
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
			}
		}

		changelog, lastErr = f.fetchChangelog(ctx)
		if lastErr == nil {
			break
		}
	}

	if lastErr != nil {
		return nil, fmt.Errorf("failed to fetch changelog after %d attempts: %w", f.config.MaxRetries, lastErr)
	}

	return f.parseChangelog(changelog), nil
}

// fetchChangelog fetches the CHANGELOG.md content
func (f *ClaudeFetcher) fetchChangelog(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, f.config.ChangelogURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "CLI-Gateway-Release-Notes-Fetcher")

	resp, err := f.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("GitHub returned status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	return string(body), nil
}

// parseChangelog parses the CHANGELOG.md content
// Expected format:
// ## 2.0.52
// - Feature description
func (f *ClaudeFetcher) parseChangelog(changelog string) *CLIReleaseNotes {
	releaseNotes := make([]ReleaseNote, 0)
	var latestVersion string

	// Pattern to match version headers like: ## 2.0.52 or ## [2.0.52]
	versionPattern := regexp.MustCompile(`(?m)^##\s+\[?([0-9]+\.[0-9]+\.[0-9]+)\]?`)
	
	matches := versionPattern.FindAllStringSubmatchIndex(changelog, -1)
	
	for i, match := range matches {
		if len(match) < 4 {
			continue
		}
		
		version := changelog[match[2]:match[3]]
		
		// Set latest version from first match
		if latestVersion == "" {
			latestVersion = version
		}
		
		// Extract changelog content between this version and the next
		changelogStart := match[1] // End of the version header line
		changelogEnd := len(changelog)
		if i+1 < len(matches) {
			changelogEnd = matches[i+1][0] // Start of next version header
		}
		
		changelogContent := strings.TrimSpace(changelog[changelogStart:changelogEnd])
		
		// Use current time as release date since CHANGELOG doesn't have dates
		releaseDate := time.Now()
		
		releaseNotes = append(releaseNotes, ReleaseNote{
			Version:     version,
			ReleaseDate: releaseDate,
			Changelog:   changelogContent,
			URL:         fmt.Sprintf("https://github.com/anthropics/claude-code/blob/main/CHANGELOG.md#%s", strings.ReplaceAll(version, ".", "")),
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

// GetLocalVersion gets the locally installed version
func (f *ClaudeFetcher) GetLocalVersion() (string, error) {
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
