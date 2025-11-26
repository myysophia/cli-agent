package release_notes

import (
	"context"
)

// CachedFetcher wraps a ReleaseNoteFetcher with caching support
type CachedFetcher struct {
	fetcher ReleaseNoteFetcher
	cache   *Cache
}

// NewCachedFetcher creates a new cached fetcher wrapper
func NewCachedFetcher(fetcher ReleaseNoteFetcher, cache *Cache) *CachedFetcher {
	return &CachedFetcher{
		fetcher: fetcher,
		cache:   cache,
	}
}

// FetchOptions contains options for fetching release notes
type FetchOptions struct {
	ForceRefresh bool // Bypass cache and fetch fresh data
	IncludeLocal bool // Include local version information
}

// Fetch retrieves release notes, using cache unless force refresh is requested
func (cf *CachedFetcher) Fetch(ctx context.Context, opts FetchOptions) (*CLIReleaseNotes, error) {
	cliName := cf.fetcher.CLIName()

	// Check cache first (unless force refresh is requested)
	if !opts.ForceRefresh {
		if cached := cf.cache.Get(cliName); cached != nil {
			// If include_local is requested, update local version
			if opts.IncludeLocal {
				return cf.withLocalVersion(cached)
			}
			return cached, nil
		}
	}

	// Fetch from external source
	data, err := cf.fetcher.Fetch(ctx)
	if err != nil {
		return nil, err
	}

	// Update cache
	cf.cache.Set(cliName, data)

	// Add local version if requested
	if opts.IncludeLocal {
		return cf.withLocalVersion(data)
	}

	return data, nil
}

// withLocalVersion adds local version information to the release notes
func (cf *CachedFetcher) withLocalVersion(data *CLIReleaseNotes) (*CLIReleaseNotes, error) {
	// Create a copy to avoid modifying cached data
	result := *data
	
	localVersion, err := cf.fetcher.GetLocalVersion()
	if err != nil {
		// CLI not installed or error getting version
		result.LocalVersion = ""
		result.UpdateAvailable = false
	} else {
		result.LocalVersion = localVersion
		result.UpdateAvailable = localVersion != "" && localVersion != data.LatestVersion
	}

	return &result, nil
}

// CLIName returns the underlying fetcher's CLI name
func (cf *CachedFetcher) CLIName() string {
	return cf.fetcher.CLIName()
}

// DisplayName returns the underlying fetcher's display name
func (cf *CachedFetcher) DisplayName() string {
	return cf.fetcher.DisplayName()
}

// IsCached returns true if data for this CLI is cached and not expired
func (cf *CachedFetcher) IsCached() bool {
	return cf.cache.Get(cf.fetcher.CLIName()) != nil
}

// Invalidate removes the cached data for this CLI
func (cf *CachedFetcher) Invalidate() {
	cf.cache.Delete(cf.fetcher.CLIName())
}
