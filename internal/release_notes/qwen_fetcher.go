package release_notes

import "time"

// QwenFetcher fetches release notes for Qwen CLI from GitHub
type QwenFetcher struct {
	*GitHubFetcher
}

// NewQwenFetcher creates a new Qwen CLI fetcher
func NewQwenFetcher() *QwenFetcher {
	return &QwenFetcher{
		GitHubFetcher: NewGitHubFetcher(GitHubFetcherConfig{
			Owner:          "QwenLM",
			Repo:           "qwen-code",
			CLIName:        "qwen",
			DisplayName:    "Qwen CLI",
			VersionCommand: "qwen --version",
			Timeout:        30 * time.Second,
			MaxRetries:     3,
		}),
	}
}
