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

// CursorFetcherConfig holds configuration for Cursor fetcher
type CursorFetcherConfig struct {
	ChangelogURL   string        // Changelog page URL
	CLIName        string        // CLI tool name
	DisplayName    string        // Display name
	VersionCommand string        // Command to get local version (e.g., "cursor-agent --version")
	Timeout        time.Duration // HTTP timeout
	MaxRetries     int           // Maximum retry attempts
}

// CursorFetcher fetches release notes from Cursor changelog page
type CursorFetcher struct {
	config     CursorFetcherConfig
	httpClient *http.Client
}

// NewCursorFetcher creates a new Cursor fetcher with the given configuration
func NewCursorFetcher(config CursorFetcherConfig) *CursorFetcher {
	if config.ChangelogURL == "" {
		config.ChangelogURL = "https://www.cursor.com/changelog"
	}
	if config.CLIName == "" {
		config.CLIName = "cursor"
	}
	if config.DisplayName == "" {
		config.DisplayName = "Cursor"
	}
	if config.VersionCommand == "" {
		config.VersionCommand = "cursor-agent --version"
	}
	if config.Timeout <= 0 {
		config.Timeout = 30 * time.Second
	}
	if config.MaxRetries <= 0 {
		config.MaxRetries = 3
	}

	return &CursorFetcher{
		config: config,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

// CLIName returns the CLI tool name
func (f *CursorFetcher) CLIName() string {
	return f.config.CLIName
}

// DisplayName returns the display name
func (f *CursorFetcher) DisplayName() string {
	return f.config.DisplayName
}


// Fetch fetches release notes from Cursor changelog page
func (f *CursorFetcher) Fetch(ctx context.Context) (*CLIReleaseNotes, error) {
	var htmlContent string
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

		htmlContent, lastErr = f.fetchHTML(ctx, f.config.ChangelogURL)
		if lastErr == nil {
			break
		}
	}

	if lastErr != nil {
		return nil, fmt.Errorf("failed to fetch changelog after %d attempts: %w", f.config.MaxRetries, lastErr)
	}

	return f.parseHTML(htmlContent), nil
}

// fetchHTML performs a single HTTP request to fetch the changelog page
func (f *CursorFetcher) fetchHTML(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("User-Agent", "CLI-Gateway-Release-Notes-Fetcher")

	resp, err := f.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("Cursor changelog returned status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	return string(body), nil
}


// parseHTML parses the Cursor changelog HTML page and extracts release notes
// The page uses Next.js with embedded JSON data in script tags
func (f *CursorFetcher) parseHTML(html string) *CLIReleaseNotes {
	// Initialize as empty slice (not nil) to ensure JSON marshals to [] instead of null
	releaseNotes := make([]ReleaseNote, 0)
	var latestVersion string

	// Pattern 1: Match href="/changelog/X-Y" patterns
	// This is simpler and more reliable for the current page structure
	hrefPattern := regexp.MustCompile(`/changelog/(\d+-\d+(?:-\d+)?)`)
	matches := hrefPattern.FindAllStringSubmatch(html, -1)
	
	versionMap := make(map[string]bool)
	
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		
		// Convert URL slug to version (e.g., "2-1" -> "2.1")
		version := strings.ReplaceAll(match[1], "-", ".")
		
		// Skip duplicates
		if versionMap[version] {
			continue
		}
		versionMap[version] = true
		
		// Set latest version from the first entry
		if latestVersion == "" {
			latestVersion = version
		}
		
		releaseNotes = append(releaseNotes, ReleaseNote{
			Version:     version,
			ReleaseDate: time.Now(), // We don't have reliable date parsing, use current time
			Changelog:   "",         // Changelog content would require fetching individual pages
			URL:         fmt.Sprintf("https://www.cursor.com/changelog/%s", match[1]),
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


// parseHTMLAlternative tries an alternative parsing approach for the changelog
func (f *CursorFetcher) parseHTMLAlternative(html string) ([]ReleaseNote, string) {
	releaseNotes := make([]ReleaseNote, 0)
	var latestVersion string

	// Look for article patterns with version and date
	// Pattern: href="/changelog/X-Y" with nearby dateTime
	articlePattern := regexp.MustCompile(`href\s*[=:]\s*["']?/changelog/(\d+-\d+(?:-\d+)?)["']?[^}]*"dateTime"\s*[=:]\s*["']?(\d{4}-\d{2}-\d{2}T[^"']+)["']?`)

	matches := articlePattern.FindAllStringSubmatch(html, -1)
	versionMap := make(map[string]bool)

	for _, match := range matches {
		if len(match) < 3 {
			continue
		}

		// Convert URL slug to version (e.g., "2-1" -> "2.1")
		version := strings.ReplaceAll(match[1], "-", ".")

		// Skip duplicates
		if versionMap[version] {
			continue
		}
		versionMap[version] = true

		// Parse date
		releaseDate, err := time.Parse(time.RFC3339, match[2])
		if err != nil {
			releaseDate = time.Time{}
		}

		if latestVersion == "" {
			latestVersion = version
		}

		releaseNotes = append(releaseNotes, ReleaseNote{
			Version:     version,
			ReleaseDate: releaseDate,
			Changelog:   "",
			URL:         fmt.Sprintf("https://www.cursor.com/changelog/%s", match[1]),
		})
	}

	// If still no results, try a more lenient pattern
	if len(releaseNotes) == 0 {
		// Look for version patterns in the latestVersion field from Next.js data
		latestVersionPattern := regexp.MustCompile(`"versionNumber"\s*:\s*"([^"]+)"`)
		if match := latestVersionPattern.FindStringSubmatch(html); len(match) >= 2 {
			latestVersion = match[1]
			releaseNotes = append(releaseNotes, ReleaseNote{
				Version:     latestVersion,
				ReleaseDate: time.Now(),
				Changelog:   "",
				URL:         fmt.Sprintf("https://www.cursor.com/changelog/%s", strings.ReplaceAll(latestVersion, ".", "-")),
			})
		}
	}

	return releaseNotes, latestVersion
}

// GetLocalVersion gets the locally installed version by running the version command
func (f *CursorFetcher) GetLocalVersion() (string, error) {
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
