package release_notes

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// genNPMPackageInfo generates arbitrary NPMPackageInfo values for testing
func genNPMPackageInfo() gopter.Gen {
	return gen.IntRange(1, 10).FlatMap(func(numVersions interface{}) gopter.Gen {
		n := numVersions.(int)

		return gen.SliceOfN(n, gen.AlphaString()).Map(func(versions []string) NPMPackageInfo {
			distTags := make(map[string]string)
			timeMap := make(map[string]string)
			versionsMap := make(map[string]NPMVersionInfo)

			baseTime := time.Now()
			for i, v := range versions {
				version := "1.0." + v
				if i == 0 {
					distTags["latest"] = version
				}
				// Create timestamps going back in time
				releaseTime := baseTime.Add(-time.Duration(i) * 24 * time.Hour)
				timeMap[version] = releaseTime.Format(time.RFC3339)
				versionsMap[version] = NPMVersionInfo{
					Version:     version,
					Description: "Test package",
					Homepage:    "https://example.com",
				}
			}

			// Add special time entries
			timeMap["created"] = baseTime.Add(-365 * 24 * time.Hour).Format(time.RFC3339)
			timeMap["modified"] = baseTime.Format(time.RFC3339)

			return NPMPackageInfo{
				Name:     "test-package",
				DistTags: distTags,
				Time:     timeMap,
				Versions: versionsMap,
			}
		})
	}, nil)
}


// **Feature: cli-release-notes, Property 8: Consistent JSON schema across CLIs (NPM)**
// **Validates: Requirements 5.1**
// For any NPM package info, parsePackageInfo SHALL produce a CLIReleaseNotes
// with the consistent JSON schema.
func TestProperty_NPMConsistentJSONSchema(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	fetcher := NewClaudeFetcher()

	properties.Property("NPM fetcher produces CLIReleaseNotes with consistent JSON schema", prop.ForAll(
		func(packageInfo NPMPackageInfo) bool {
			result := fetcher.parsePackageInfo(&packageInfo)

			// Verify the result can be serialized to JSON
			jsonBytes, err := json.Marshal(result)
			if err != nil {
				t.Logf("Failed to marshal result: %v", err)
				return false
			}

			// Verify the JSON has all required fields by unmarshaling to a map
			var jsonMap map[string]interface{}
			if err := json.Unmarshal(jsonBytes, &jsonMap); err != nil {
				t.Logf("Failed to unmarshal to map: %v", err)
				return false
			}

			// Check required fields exist
			requiredFields := []string{
				"cli_name",
				"display_name",
				"latest_version",
				"local_version",
				"update_available",
				"last_updated",
				"releases",
			}

			for _, field := range requiredFields {
				if _, exists := jsonMap[field]; !exists {
					t.Logf("Missing required field '%s'", field)
					return false
				}
			}

			// Verify releases is an array
			releasesField, ok := jsonMap["releases"].([]interface{})
			if !ok {
				t.Logf("'releases' field is not an array")
				return false
			}

			// Verify each release has required fields
			for i, release := range releasesField {
				releaseMap, ok := release.(map[string]interface{})
				if !ok {
					t.Logf("Release %d is not an object", i)
					return false
				}

				releaseRequiredFields := []string{"version", "release_date", "changelog", "url"}
				for _, field := range releaseRequiredFields {
					if _, exists := releaseMap[field]; !exists {
						t.Logf("Missing required field '%s' in release %d", field, i)
						return false
					}
				}
			}

			return true
		},
		genNPMPackageInfo(),
	))

	properties.TestingRun(t)
}

// **Feature: cli-release-notes, Property 8: Consistent JSON schema across CLIs (NPM config)**
// **Validates: Requirements 5.1**
// For any NPM package info, parsePackageInfo SHALL preserve the fetcher configuration.
func TestProperty_NPMParsePreservesConfig(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	fetcher := NewClaudeFetcher()

	properties.Property("parsePackageInfo preserves fetcher configuration", prop.ForAll(
		func(packageInfo NPMPackageInfo) bool {
			result := fetcher.parsePackageInfo(&packageInfo)

			if result.CLIName != "claude" {
				t.Logf("Expected CLIName 'claude', got '%s'", result.CLIName)
				return false
			}

			if result.DisplayName != "Claude CLI" {
				t.Logf("Expected DisplayName 'Claude CLI', got '%s'", result.DisplayName)
				return false
			}

			return true
		},
		genNPMPackageInfo(),
	))

	properties.TestingRun(t)
}


// **Feature: cli-release-notes, Property 10: Releases sorted by date descending (NPM)**
// **Validates: Requirements 7.1**
// For any NPM package info, parsePackageInfo SHALL return releases sorted by date descending.
func TestProperty_NPMReleasesSortedByDateDescending(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	fetcher := NewClaudeFetcher()

	properties.Property("parsePackageInfo returns releases sorted by date descending", prop.ForAll(
		func(packageInfo NPMPackageInfo) bool {
			result := fetcher.parsePackageInfo(&packageInfo)

			// Check that releases are sorted by date descending
			for i := 1; i < len(result.Releases); i++ {
				if result.Releases[i-1].ReleaseDate.Before(result.Releases[i].ReleaseDate) {
					t.Logf("Releases not sorted: %v before %v",
						result.Releases[i-1].ReleaseDate, result.Releases[i].ReleaseDate)
					return false
				}
			}

			return true
		},
		genNPMPackageInfo(),
	))

	properties.TestingRun(t)
}

// Unit test for parsePackageInfo with empty input
func TestNPMParsePackageInfo_Empty(t *testing.T) {
	fetcher := NewClaudeFetcher()
	packageInfo := &NPMPackageInfo{
		Name:     "@anthropic-ai/claude-code",
		DistTags: make(map[string]string),
		Time:     make(map[string]string),
		Versions: make(map[string]NPMVersionInfo),
	}

	result := fetcher.parsePackageInfo(packageInfo)

	if result.CLIName != "claude" {
		t.Errorf("Expected CLIName 'claude', got '%s'", result.CLIName)
	}

	if result.LatestVersion != "" {
		t.Errorf("Expected empty LatestVersion, got '%s'", result.LatestVersion)
	}

	if len(result.Releases) != 0 {
		t.Errorf("Expected 0 releases, got %d", len(result.Releases))
	}
}

// Unit test for parsePackageInfo with real-like data
func TestNPMParsePackageInfo_RealData(t *testing.T) {
	fetcher := NewClaudeFetcher()

	now := time.Now()
	packageInfo := &NPMPackageInfo{
		Name: "@anthropic-ai/claude-code",
		DistTags: map[string]string{
			"latest": "2.0.53",
		},
		Time: map[string]string{
			"created":  now.Add(-365 * 24 * time.Hour).Format(time.RFC3339),
			"modified": now.Format(time.RFC3339),
			"2.0.53":   now.Format(time.RFC3339),
			"2.0.52":   now.Add(-24 * time.Hour).Format(time.RFC3339),
			"2.0.51":   now.Add(-48 * time.Hour).Format(time.RFC3339),
		},
		Versions: map[string]NPMVersionInfo{
			"2.0.53": {Version: "2.0.53", Description: "Claude CLI", Homepage: "https://github.com/anthropics/claude-code"},
			"2.0.52": {Version: "2.0.52", Description: "Claude CLI", Homepage: "https://github.com/anthropics/claude-code"},
			"2.0.51": {Version: "2.0.51", Description: "Claude CLI", Homepage: "https://github.com/anthropics/claude-code"},
		},
	}

	result := fetcher.parsePackageInfo(packageInfo)

	if result.CLIName != "claude" {
		t.Errorf("Expected CLIName 'claude', got '%s'", result.CLIName)
	}

	if result.DisplayName != "Claude CLI" {
		t.Errorf("Expected DisplayName 'Claude CLI', got '%s'", result.DisplayName)
	}

	if result.LatestVersion != "2.0.53" {
		t.Errorf("Expected LatestVersion '2.0.53', got '%s'", result.LatestVersion)
	}

	// Should have 3 releases (excluding created and modified)
	if len(result.Releases) != 3 {
		t.Errorf("Expected 3 releases, got %d", len(result.Releases))
	}

	// First release should be the newest (2.0.53)
	if len(result.Releases) > 0 && result.Releases[0].Version != "2.0.53" {
		t.Errorf("Expected first release to be '2.0.53', got '%s'", result.Releases[0].Version)
	}
}

// Unit test for parsePackageInfo skipping special time entries
func TestNPMParsePackageInfo_SkipsSpecialTimeEntries(t *testing.T) {
	fetcher := NewClaudeFetcher()

	now := time.Now()
	packageInfo := &NPMPackageInfo{
		Name: "@anthropic-ai/claude-code",
		DistTags: map[string]string{
			"latest": "1.0.0",
		},
		Time: map[string]string{
			"created":  now.Add(-365 * 24 * time.Hour).Format(time.RFC3339),
			"modified": now.Format(time.RFC3339),
			"1.0.0":    now.Format(time.RFC3339),
		},
		Versions: map[string]NPMVersionInfo{
			"1.0.0": {Version: "1.0.0", Description: "Test", Homepage: "https://example.com"},
		},
	}

	result := fetcher.parsePackageInfo(packageInfo)

	// Should only have 1 release (1.0.0), not "created" or "modified"
	if len(result.Releases) != 1 {
		t.Errorf("Expected 1 release, got %d", len(result.Releases))
	}

	if len(result.Releases) > 0 && result.Releases[0].Version != "1.0.0" {
		t.Errorf("Expected release version '1.0.0', got '%s'", result.Releases[0].Version)
	}
}
