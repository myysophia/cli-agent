package release_notes

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// ServiceConfig holds configuration for the ReleaseNotesService
type ServiceConfig struct {
	CacheTTL        time.Duration // Cache time-to-live, default 1 hour
	RefreshInterval time.Duration // Scheduled refresh interval, default 1 hour
	StoragePath     string        // Storage file path for persistence
}

// DefaultServiceConfig returns the default service configuration
func DefaultServiceConfig() ServiceConfig {
	return ServiceConfig{
		CacheTTL:        time.Hour,
		RefreshInterval: time.Hour,
		StoragePath:     "data/release_notes.json",
	}
}

// ReleaseNotesService manages release notes fetching, caching, and scheduled refresh
type ReleaseNotesService struct {
	config   ServiceConfig
	fetchers map[string]ReleaseNoteFetcher
	cache    *Cache
	storage  *Storage
	mu       sync.RWMutex

	// Scheduler control
	stopChan chan struct{}
	wg       sync.WaitGroup
	running  bool
}

// SupportedCLIs returns the list of supported CLI names
var SupportedCLIs = []string{"claude", "codex", "cursor", "gemini", "qwen"}

// NewReleaseNotesService creates a new release notes service
func NewReleaseNotesService(config ServiceConfig) *ReleaseNotesService {
	if config.CacheTTL <= 0 {
		config.CacheTTL = time.Hour
	}
	if config.RefreshInterval <= 0 {
		config.RefreshInterval = time.Hour
	}
	if config.StoragePath == "" {
		config.StoragePath = "data/release_notes.json"
	}

	// Create storage with default persistence config
	storageConfig := DefaultPersistenceConfig()
	storageConfig.StoragePath = config.StoragePath

	return &ReleaseNotesService{
		config:   config,
		fetchers: make(map[string]ReleaseNoteFetcher),
		cache:    NewCache(config.CacheTTL),
		storage:  NewStorage(storageConfig),
		stopChan: make(chan struct{}),
	}
}

// InitializeFetchers initializes all CLI fetchers
func (s *ReleaseNotesService) InitializeFetchers() {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Initialize all fetchers
	s.fetchers["claude"] = NewClaudeFetcher() // Fetches from GitHub CHANGELOG.md
	s.fetchers["codex"] = NewCodexFetcher()   // Fetches from GitHub Releases (non-prerelease only)
	s.fetchers["cursor"] = NewCursorFetcher(CursorFetcherConfig{})
	s.fetchers["gemini"] = NewGeminiFetcher()
	s.fetchers["qwen"] = NewQwenFetcher()
}

// RegisterFetcher registers a custom fetcher (useful for testing)
func (s *ReleaseNotesService) RegisterFetcher(name string, fetcher ReleaseNoteFetcher) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.fetchers[name] = fetcher
}

// Start starts the service: loads from storage and starts scheduled refresh
func (s *ReleaseNotesService) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return fmt.Errorf("service is already running")
	}
	s.running = true
	s.stopChan = make(chan struct{})
	s.mu.Unlock()

	// Initialize fetchers if not already done
	if len(s.fetchers) == 0 {
		s.InitializeFetchers()
	}

	// Load from persistent storage
	if err := s.LoadFromStorage(); err != nil {
		log.Printf("Warning: failed to load from storage: %v", err)
	}

	// Fetch immediately on startup (Requirement 8.1)
	log.Println("Fetching release notes on startup...")
	if err := s.Refresh(ctx, false); err != nil {
		log.Printf("Warning: initial fetch failed: %v", err)
	}

	// Start scheduled refresh
	s.startScheduler(ctx)

	return nil
}

// Stop stops the service and saves to storage
func (s *ReleaseNotesService) Stop() error {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return nil
	}
	s.running = false
	close(s.stopChan)
	s.mu.Unlock()

	// Wait for scheduler to stop
	s.wg.Wait()

	// Save to storage on shutdown
	return s.SaveToStorage()
}

// GetAll returns release notes for all CLI tools
func (s *ReleaseNotesService) GetAll(ctx context.Context, includeLocal bool) (*AllReleaseNotes, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := &AllReleaseNotes{
		CLIs:        make(map[string]*CLIReleaseNotes),
		LastUpdated: time.Now(),
	}

	for name, fetcher := range s.fetchers {
		cached := s.cache.Get(name)
		if cached == nil {
			continue
		}

		// Create a copy to avoid modifying cached data
		cli := *cached

		// Add local version if requested
		if includeLocal {
			s.updateLocalVersion(&cli, fetcher)
		}

		result.CLIs[name] = &cli
	}

	// Update last updated time from cache
	if allNotes := s.cache.GetAllReleaseNotes(); allNotes != nil {
		result.LastUpdated = allNotes.LastUpdated
	}

	return result, nil
}

// GetByCLI returns release notes for a specific CLI tool
func (s *ReleaseNotesService) GetByCLI(ctx context.Context, cliName string, includeLocal bool, forceRefresh bool) (*CLIReleaseNotes, error) {
	s.mu.RLock()
	fetcher, exists := s.fetchers[cliName]
	s.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("unsupported CLI: %s", cliName)
	}

	// Force refresh if requested
	if forceRefresh {
		if err := s.refreshCLI(ctx, cliName, fetcher); err != nil {
			// If refresh fails, try to return cached data
			cached := s.cache.Get(cliName)
			if cached != nil {
				log.Printf("Warning: refresh failed for %s, using cached data: %v", cliName, err)
			} else {
				return nil, fmt.Errorf("failed to fetch release notes for %s: %w", cliName, err)
			}
		}
	}

	// Get from cache
	cached := s.cache.Get(cliName)
	if cached == nil {
		// Try to fetch if not in cache
		if err := s.refreshCLI(ctx, cliName, fetcher); err != nil {
			return nil, fmt.Errorf("failed to fetch release notes for %s: %w", cliName, err)
		}
		cached = s.cache.Get(cliName)
		if cached == nil {
			return nil, fmt.Errorf("no release notes available for %s", cliName)
		}
	}

	// Create a copy to avoid modifying cached data
	result := *cached

	// Add local version if requested
	if includeLocal {
		s.updateLocalVersion(&result, fetcher)
	}

	return &result, nil
}

// Refresh fetches fresh release notes for all CLI tools
func (s *ReleaseNotesService) Refresh(ctx context.Context, force bool) error {
	s.mu.RLock()
	fetchers := make(map[string]ReleaseNoteFetcher)
	for k, v := range s.fetchers {
		fetchers[k] = v
	}
	s.mu.RUnlock()

	var wg sync.WaitGroup
	errChan := make(chan error, len(fetchers))

	for name, fetcher := range fetchers {
		// Skip if cached and not forcing refresh
		if !force && s.cache.Get(name) != nil && !s.cache.IsExpired(name) {
			continue
		}

		wg.Add(1)
		go func(name string, fetcher ReleaseNoteFetcher) {
			defer wg.Done()
			if err := s.refreshCLI(ctx, name, fetcher); err != nil {
				errChan <- fmt.Errorf("%s: %w", name, err)
			}
		}(name, fetcher)
	}

	wg.Wait()
	close(errChan)

	// Collect errors
	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}

	// Save to storage after refresh
	if saveErr := s.SaveToStorage(); saveErr != nil {
		log.Printf("Warning: failed to save to storage: %v", saveErr)
	}

	if len(errors) > 0 {
		return fmt.Errorf("refresh failed for %d CLIs: %v", len(errors), errors)
	}

	return nil
}

// refreshCLI fetches release notes for a single CLI
func (s *ReleaseNotesService) refreshCLI(ctx context.Context, name string, fetcher ReleaseNoteFetcher) error {
	data, err := fetcher.Fetch(ctx)
	if err != nil {
		return err
	}

	s.cache.Set(name, data)
	log.Printf("Refreshed release notes for %s (latest: %s)", name, data.LatestVersion)
	return nil
}

// updateLocalVersion updates the local version and update_available flag
func (s *ReleaseNotesService) updateLocalVersion(cli *CLIReleaseNotes, fetcher ReleaseNoteFetcher) {
	localVersion, err := fetcher.GetLocalVersion()
	if err != nil {
		// CLI not installed or error getting version
		cli.LocalVersion = ""
		cli.UpdateAvailable = false
		return
	}

	cli.LocalVersion = localVersion
	cli.UpdateAvailable = CompareVersions(localVersion, cli.LatestVersion)
}

// CompareVersions returns true if localVersion differs from latestVersion
// indicating an update is available
func CompareVersions(localVersion, latestVersion string) bool {
	if localVersion == "" || latestVersion == "" {
		return false
	}
	return localVersion != latestVersion
}

// startScheduler starts the background refresh scheduler
func (s *ReleaseNotesService) startScheduler(ctx context.Context) {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()

		ticker := time.NewTicker(s.config.RefreshInterval)
		defer ticker.Stop()

		for {
			select {
			case <-s.stopChan:
				log.Println("Scheduler stopped")
				return
			case <-ctx.Done():
				log.Println("Scheduler context cancelled")
				return
			case <-ticker.C:
				log.Println("Scheduled refresh triggered")
				if err := s.Refresh(ctx, false); err != nil {
					// Log error but preserve cache (Requirement 8.3)
					log.Printf("Scheduled refresh failed: %v", err)
				}
			}
		}
	}()
}

// SaveToStorage saves the current cache to persistent storage
func (s *ReleaseNotesService) SaveToStorage() error {
	allNotes := s.cache.GetAllReleaseNotes()
	if allNotes == nil {
		return nil // Nothing to save
	}
	return s.storage.SaveToStorage(allNotes)
}

// LoadFromStorage loads release notes from persistent storage into cache
func (s *ReleaseNotesService) LoadFromStorage() error {
	data, err := s.storage.LoadFromStorage()
	if err != nil {
		return err
	}

	if data != nil && len(data.CLIs) > 0 {
		s.cache.SetAll(data)
		log.Printf("Loaded %d CLI release notes from storage", len(data.CLIs))
	}

	return nil
}

// IsValidCLI checks if the given CLI name is supported
func IsValidCLI(name string) bool {
	for _, cli := range SupportedCLIs {
		if cli == name {
			return true
		}
	}
	return false
}

// GetSupportedCLIs returns the list of supported CLI names
func (s *ReleaseNotesService) GetSupportedCLIs() []string {
	return SupportedCLIs
}

// IsRunning returns whether the service is currently running
func (s *ReleaseNotesService) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

// GetCache returns the cache (for testing purposes)
func (s *ReleaseNotesService) GetCache() *Cache {
	return s.cache
}

// GetStorage returns the storage (for testing purposes)
func (s *ReleaseNotesService) GetStorage() *Storage {
	return s.storage
}
