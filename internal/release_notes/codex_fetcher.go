package release_notes

import "time"

// CodexFetcher fetches release notes for Codex CLI from GitHub
type CodexFetcher struct {
	*GitHubFetcher
}

// NewCodexFetcher creates a new Codex CLI fetcher
func NewCodexFetcher() *CodexFetcher {
	return &CodexFetcher{
		GitHubFetcher: NewGitHubFetcher(GitHubFetcherConfig{
			Owner:          "openai",
			Repo:           "codex",
			CLIName:        "codex",
			DisplayName:    "Codex CLI",
			VersionCommand: "codex --version",
			Timeout:        30 * time.Second,
			MaxRetries:     3,
		}),
	}
}
