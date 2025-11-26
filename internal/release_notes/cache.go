package release_notes

import (
	"sync"
	"time"
)

// CacheEntry represents a cached item with expiration
type CacheEntry struct {
	Data      *CLIReleaseNotes
	CachedAt  time.Time
	ExpiresAt time.Time
}

// Cache provides thread-safe in-memory caching for release notes
type Cache struct {
	mu      sync.RWMutex
	entries map[string]*CacheEntry
	ttl     time.Duration
}

// NewCache creates a new cache with the specified TTL
func NewCache(ttl time.Duration) *Cache {
	if ttl <= 0 {
		ttl = time.Hour // Default 1 hour
	}
	return &Cache{
		entries: make(map[string]*CacheEntry),
		ttl:     ttl,
	}
}

// Get retrieves a cached entry by CLI name
// Returns nil if not found or expired
func (c *Cache) Get(cliName string) *CLIReleaseNotes {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.entries[cliName]
	if !exists {
		return nil
	}

	if c.isExpired(entry) {
		return nil
	}

	return entry.Data
}

// Set stores a release notes entry in the cache
func (c *Cache) Set(cliName string, data *CLIReleaseNotes) {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	c.entries[cliName] = &CacheEntry{
		Data:      data,
		CachedAt:  now,
		ExpiresAt: now.Add(c.ttl),
	}
}


// IsExpired checks if a cached entry for the given CLI name is expired
func (c *Cache) IsExpired(cliName string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.entries[cliName]
	if !exists {
		return true // Non-existent entries are considered expired
	}

	return c.isExpired(entry)
}

// isExpired is an internal helper to check expiration (caller must hold lock)
func (c *Cache) isExpired(entry *CacheEntry) bool {
	return time.Now().After(entry.ExpiresAt)
}

// Delete removes a cached entry
func (c *Cache) Delete(cliName string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.entries, cliName)
}

// Clear removes all cached entries
func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries = make(map[string]*CacheEntry)
}

// GetAll returns all non-expired cached entries
func (c *Cache) GetAll() map[string]*CLIReleaseNotes {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make(map[string]*CLIReleaseNotes)
	for name, entry := range c.entries {
		if !c.isExpired(entry) {
			result[name] = entry.Data
		}
	}
	return result
}

// GetAllReleaseNotes returns all cached entries as AllReleaseNotes struct
func (c *Cache) GetAllReleaseNotes() *AllReleaseNotes {
	clis := c.GetAll()
	if len(clis) == 0 {
		return nil
	}

	// Find the most recent update time
	var lastUpdated time.Time
	for _, cli := range clis {
		if cli.LastUpdated.After(lastUpdated) {
			lastUpdated = cli.LastUpdated
		}
	}

	return &AllReleaseNotes{
		CLIs:        clis,
		LastUpdated: lastUpdated,
	}
}

// SetAll stores multiple release notes entries in the cache
func (c *Cache) SetAll(data *AllReleaseNotes) {
	if data == nil || data.CLIs == nil {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for name, cli := range data.CLIs {
		c.entries[name] = &CacheEntry{
			Data:      cli,
			CachedAt:  now,
			ExpiresAt: now.Add(c.ttl),
		}
	}
}

// GetEntry returns the cache entry with metadata (for testing/debugging)
func (c *Cache) GetEntry(cliName string) *CacheEntry {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.entries[cliName]
}

// TTL returns the cache TTL duration
func (c *Cache) TTL() time.Duration {
	return c.ttl
}

// SetTTL updates the cache TTL (does not affect existing entries)
func (c *Cache) SetTTL(ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ttl = ttl
}

// Size returns the number of entries in the cache
func (c *Cache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.entries)
}
