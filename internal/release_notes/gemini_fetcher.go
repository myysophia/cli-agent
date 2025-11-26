package release_notes

import "time"

// GeminiFetcher fetches release notes for Gemini CLI from GitHub
type GeminiFetcher struct {
	*GitHubFetcher
}

// NewGeminiFetcher creates a new Gemini CLI fetcher
func NewGeminiFetcher() *GeminiFetcher {
	return &GeminiFetcher{
		GitHubFetcher: NewGitHubFetcher(GitHubFetcherConfig{
			Owner:          "google-gemini",
			Repo:           "gemini-cli",
			CLIName:        "gemini",
			DisplayName:    "Gemini CLI",
			VersionCommand: "gemini --version",
			Timeout:        30 * time.Second,
			MaxRetries:     3,
		}),
	}
}
