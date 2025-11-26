package release_notes

import (
	"context"
	"time"
)

// ReleaseNote represents a single version release note
type ReleaseNote struct {
	Version     string    `json:"version"`      // Version number
	ReleaseDate time.Time `json:"release_date"` // Release date
	Changelog   string    `json:"changelog"`    // Changelog content (Markdown format)
	URL         string    `json:"url"`          // Release page URL
}

// CLIReleaseNotes represents all release notes for a CLI tool
type CLIReleaseNotes struct {
	CLIName         string        `json:"cli_name"`         // CLI name
	DisplayName     string        `json:"display_name"`     // Display name
	LatestVersion   string        `json:"latest_version"`   // Latest version
	LocalVersion    string        `json:"local_version"`    // Locally installed version
	UpdateAvailable bool          `json:"update_available"` // Whether update is available
	LastUpdated     time.Time     `json:"last_updated"`     // Last updated time
	Releases        []ReleaseNote `json:"releases"`         // Release history
}

// AllReleaseNotes represents release notes for all CLI tools
type AllReleaseNotes struct {
	CLIs        map[string]*CLIReleaseNotes `json:"clis"`
	LastUpdated time.Time                   `json:"last_updated"`
}

// ReleaseNoteFetcher defines the interface for fetching release notes
type ReleaseNoteFetcher interface {
	// CLIName returns the CLI tool name
	CLIName() string

	// DisplayName returns the display name
	DisplayName() string

	// Fetch fetches release notes from external source
	Fetch(ctx context.Context) (*CLIReleaseNotes, error)

	// GetLocalVersion gets the locally installed version
	GetLocalVersion() (string, error)
}

// CacheConfig holds cache configuration
type CacheConfig struct {
	TTL time.Duration // Cache time-to-live, default 1 hour
}

// PersistenceConfig holds persistence configuration
type PersistenceConfig struct {
	StoragePath   string // Storage file path, default "data/release_notes.json"
	WriteOnUpdate bool   // Write to file on each update, default true
	BackupEnabled bool   // Enable backup, default true
	BackupCount   int    // Number of backups to keep, default 3
}
