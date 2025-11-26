package release_notes

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// MockFetcher is a test fetcher that tracks fetch calls
type MockFetcher struct {
	cliName     string
	displayName string
	fetchCount  atomic.Int32
	data        *CLIReleaseNotes
	fetchErr    error
}

func NewMockFetcher(cliName string) *MockFetcher {
	return &MockFetcher{
		cliName:     cliName,
		displayName: cliName + " CLI",
		data: &CLIReleaseNotes{
			CLIName:       cliName,
			DisplayName:   cliName + " CLI",
			LatestVersion: "1.0.0",
			LastUpdated:   time.Now(),
			Releases: []ReleaseNote{
				{Version: "1.0.0", ReleaseDate: time.Now(), Changelog: "Initial release"},
			},
		},
	}
}

func (m *MockFetcher) CLIName() string     { return m.cliName }
func (m *MockFetcher) DisplayName() string { return m.displayName }

func (m *MockFetcher) Fetch(ctx context.Context) (*CLIReleaseNotes, error) {
	m.fetchCount.Add(1)
	if m.fetchErr != nil {
		return nil, m.fetchErr
	}
	return m.data, nil
}

func (m *MockFetcher) GetLocalVersion() (string, error) {
	return "0.9.0", nil
}

func (m *MockFetcher) FetchCount() int {
	return int(m.fetchCount.Load())
}

func (m *MockFetcher) Reset() {
	m.fetchCount.Store(0)
}


// **Feature: cli-release-notes, Property 4: Cache prevents redundant fetches**
// **Validates: Requirements 3.1, 3.2**
// For any sequence of requests within the cache TTL, only the first request
// SHALL trigger an external fetch; subsequent requests SHALL use cached data.
func TestProperty_CachePreventsRedundantFetches(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	properties.Property("cache prevents redundant fetches within TTL", prop.ForAll(
		func(numRequests int) bool {
			// Create cache with long TTL (won't expire during test)
			cache := NewCache(time.Hour)
			fetcher := NewMockFetcher("test-cli")
			cachedFetcher := NewCachedFetcher(fetcher, cache)

			// Make multiple requests
			for i := 0; i < numRequests; i++ {
				_, err := cachedFetcher.Fetch(context.Background(), FetchOptions{
					ForceRefresh: false,
				})
				if err != nil {
					return false
				}
			}

			// Only the first request should trigger a fetch
			return fetcher.FetchCount() == 1
		},
		gen.IntRange(1, 20), // 1 to 20 requests
	))

	properties.TestingRun(t)
}

// **Feature: cli-release-notes, Property 5: Cache expiration triggers refresh**
// **Validates: Requirements 3.3**
// For any cached data older than the configured TTL, a new request
// SHALL trigger a fresh fetch from external sources.
func TestProperty_CacheExpirationTriggersRefresh(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	properties.Property("expired cache triggers new fetch", prop.ForAll(
		func(ttlMs int) bool {
			// Create cache with very short TTL
			ttl := time.Duration(ttlMs) * time.Millisecond
			cache := NewCache(ttl)
			fetcher := NewMockFetcher("test-cli")
			cachedFetcher := NewCachedFetcher(fetcher, cache)

			// First fetch - should call external source
			_, err := cachedFetcher.Fetch(context.Background(), FetchOptions{})
			if err != nil {
				return false
			}
			if fetcher.FetchCount() != 1 {
				return false
			}

			// Wait for cache to expire
			time.Sleep(ttl + 10*time.Millisecond)

			// Second fetch after expiration - should call external source again
			_, err = cachedFetcher.Fetch(context.Background(), FetchOptions{})
			if err != nil {
				return false
			}

			// Should have fetched twice (once initially, once after expiration)
			return fetcher.FetchCount() == 2
		},
		gen.IntRange(10, 50), // TTL between 10-50ms for fast tests
	))

	properties.TestingRun(t)
}


// **Feature: cli-release-notes, Property 6: Force refresh bypasses cache**
// **Validates: Requirements 3.4**
// For any request with force_refresh=true, the system SHALL fetch from
// external sources regardless of cache state.
func TestProperty_ForceRefreshBypassesCache(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	properties.Property("force refresh always fetches from source", prop.ForAll(
		func(numForceRefreshes int) bool {
			// Create cache with long TTL
			cache := NewCache(time.Hour)
			fetcher := NewMockFetcher("test-cli")
			cachedFetcher := NewCachedFetcher(fetcher, cache)

			// First normal fetch to populate cache
			_, err := cachedFetcher.Fetch(context.Background(), FetchOptions{
				ForceRefresh: false,
			})
			if err != nil {
				return false
			}

			// Multiple force refresh requests
			for i := 0; i < numForceRefreshes; i++ {
				_, err := cachedFetcher.Fetch(context.Background(), FetchOptions{
					ForceRefresh: true,
				})
				if err != nil {
					return false
				}
			}

			// Should have fetched 1 (initial) + numForceRefreshes times
			expectedFetches := 1 + numForceRefreshes
			return fetcher.FetchCount() == expectedFetches
		},
		gen.IntRange(1, 10), // 1 to 10 force refreshes
	))

	properties.TestingRun(t)
}

// Additional unit tests for cache behavior

func TestCache_GetSetBasic(t *testing.T) {
	cache := NewCache(time.Hour)
	
	data := &CLIReleaseNotes{
		CLIName:       "test",
		LatestVersion: "1.0.0",
	}
	
	// Initially empty
	if got := cache.Get("test"); got != nil {
		t.Errorf("Expected nil for non-existent key, got %v", got)
	}
	
	// Set and get
	cache.Set("test", data)
	got := cache.Get("test")
	if got == nil {
		t.Error("Expected data after Set, got nil")
	}
	if got.CLIName != "test" {
		t.Errorf("Expected CLIName 'test', got '%s'", got.CLIName)
	}
}

func TestCache_IsExpired(t *testing.T) {
	cache := NewCache(50 * time.Millisecond)
	
	// Non-existent entry is considered expired
	if !cache.IsExpired("nonexistent") {
		t.Error("Non-existent entry should be considered expired")
	}
	
	data := &CLIReleaseNotes{CLIName: "test"}
	cache.Set("test", data)
	
	// Fresh entry is not expired
	if cache.IsExpired("test") {
		t.Error("Fresh entry should not be expired")
	}
	
	// Wait for expiration
	time.Sleep(60 * time.Millisecond)
	
	// Now it should be expired
	if !cache.IsExpired("test") {
		t.Error("Entry should be expired after TTL")
	}
}

func TestCache_Clear(t *testing.T) {
	cache := NewCache(time.Hour)
	
	cache.Set("cli1", &CLIReleaseNotes{CLIName: "cli1"})
	cache.Set("cli2", &CLIReleaseNotes{CLIName: "cli2"})
	
	if cache.Size() != 2 {
		t.Errorf("Expected size 2, got %d", cache.Size())
	}
	
	cache.Clear()
	
	if cache.Size() != 0 {
		t.Errorf("Expected size 0 after clear, got %d", cache.Size())
	}
}

func TestCache_Delete(t *testing.T) {
	cache := NewCache(time.Hour)
	
	cache.Set("test", &CLIReleaseNotes{CLIName: "test"})
	
	if cache.Get("test") == nil {
		t.Error("Expected data after Set")
	}
	
	cache.Delete("test")
	
	if cache.Get("test") != nil {
		t.Error("Expected nil after Delete")
	}
}
