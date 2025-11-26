package release_notes

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// genGitHubRelease generates arbitrary GitHubRelease values for testing
func genGitHubRelease() gopter.Gen {
	return gopter.CombineGens(
		gen.AlphaString(),                                                    // TagName
		gen.AlphaString(),                                                    // Name
		gen.AlphaString(),                                                    // Body
		gen.TimeRange(time.Now().Add(-365*24*time.Hour), 365*24*time.Hour),   // PublishedAt
		gen.AlphaString(),                                                    // HTMLURL
		gen.Bool(),                                                           // Draft
		gen.Bool(),                                                           // Prerelease
	).Map(func(values []interface{}) GitHubRelease {
		return GitHubRelease{
			TagName:     values[0].(string),
			Name:        values[1].(string),
			Body:        values[2].(string),
			PublishedAt: values[3].(time.Time),
			HTMLURL:     values[4].(string),
			Draft:       values[5].(bool),
			Prerelease:  values[6].(bool),
		}
	})
}

// genNonDraftRelease generates GitHubRelease values that are not drafts or prereleases
func genNonDraftRelease() gopter.Gen {
	return gopter.CombineGens(
		gen.AlphaString(),                                                    // TagName
		gen.AlphaString(),                                                    // Name
		gen.AlphaString(),                                                    // Body
		gen.TimeRange(time.Now().Add(-365*24*time.Hour), 365*24*time.Hour),   // PublishedAt
		gen.AlphaString(),                                                    // HTMLURL
	).Map(func(values []interface{}) GitHubRelease {
		return GitHubRelease{
			TagName:     values[0].(string),
			Name:        values[1].(string),
			Body:        values[2].(string),
			PublishedAt: values[3].(time.Time),
			HTMLURL:     values[4].(string),
			Draft:       false,
			Prerelease:  false,
		}
	})
}


// **Feature: cli-release-notes, Property 8: Consistent JSON schema across CLIs**
// **Validates: Requirements 5.1**
// For any CLI tool, the response JSON SHALL have the same structure with fields:
// cli_name, display_name, latest_version, local_version, update_available, last_updated, releases.
func TestProperty_ConsistentJSONSchemaAcrossCLIs(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	// Test that all GitHub-based fetchers produce consistent JSON schema
	fetchers := []*GitHubFetcher{
		NewCodexFetcher().GitHubFetcher,
		NewGeminiFetcher().GitHubFetcher,
		NewQwenFetcher().GitHubFetcher,
	}

	// Generate a list of non-draft releases
	genReleases := gen.SliceOf(genNonDraftRelease())

	properties.Property("All GitHub fetchers produce CLIReleaseNotes with consistent JSON schema", prop.ForAll(
		func(releases []GitHubRelease) bool {
			for _, fetcher := range fetchers {
				result := fetcher.parseReleases(releases)

				// Verify the result can be serialized to JSON
				jsonBytes, err := json.Marshal(result)
				if err != nil {
					t.Logf("Failed to marshal result for %s: %v", fetcher.CLIName(), err)
					return false
				}

				// Verify the JSON has all required fields by unmarshaling to a map
				var jsonMap map[string]interface{}
				if err := json.Unmarshal(jsonBytes, &jsonMap); err != nil {
					t.Logf("Failed to unmarshal to map for %s: %v", fetcher.CLIName(), err)
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
						t.Logf("Missing required field '%s' for %s", field, fetcher.CLIName())
						return false
					}
				}

				// Verify releases is an array
				releasesField, ok := jsonMap["releases"].([]interface{})
				if !ok {
					t.Logf("'releases' field is not an array for %s", fetcher.CLIName())
					return false
				}

				// Verify each release has required fields
				for i, release := range releasesField {
					releaseMap, ok := release.(map[string]interface{})
					if !ok {
						t.Logf("Release %d is not an object for %s", i, fetcher.CLIName())
						return false
					}

					releaseRequiredFields := []string{"version", "release_date", "changelog", "url"}
					for _, field := range releaseRequiredFields {
						if _, exists := releaseMap[field]; !exists {
							t.Logf("Missing required field '%s' in release %d for %s", field, i, fetcher.CLIName())
							return false
						}
					}
				}
			}
			return true
		},
		genReleases,
	))

	properties.TestingRun(t)
}

// **Feature: cli-release-notes, Property 8: Consistent JSON schema across CLIs (parseReleases)**
// **Validates: Requirements 5.1**
// For any list of GitHub releases, parseReleases SHALL produce a CLIReleaseNotes
// with the correct CLI name and display name from the fetcher configuration.
func TestProperty_ParseReleasesPreservesConfig(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	genReleases := gen.SliceOf(genGitHubRelease())

	properties.Property("parseReleases preserves fetcher configuration", prop.ForAll(
		func(releases []GitHubRelease) bool {
			testCases := []struct {
				fetcher     *GitHubFetcher
				cliName     string
				displayName string
			}{
				{NewCodexFetcher().GitHubFetcher, "codex", "Codex CLI"},
				{NewGeminiFetcher().GitHubFetcher, "gemini", "Gemini CLI"},
				{NewQwenFetcher().GitHubFetcher, "qwen", "Qwen CLI"},
			}

			for _, tc := range testCases {
				result := tc.fetcher.parseReleases(releases)

				if result.CLIName != tc.cliName {
					t.Logf("Expected CLIName '%s', got '%s'", tc.cliName, result.CLIName)
					return false
				}

				if result.DisplayName != tc.displayName {
					t.Logf("Expected DisplayName '%s', got '%s'", tc.displayName, result.DisplayName)
					return false
				}
			}
			return true
		},
		genReleases,
	))

	properties.TestingRun(t)
}


// **Feature: cli-release-notes, Property 8: Consistent JSON schema across CLIs (draft filtering)**
// **Validates: Requirements 5.1**
// For any list of GitHub releases, parseReleases SHALL filter out drafts and prereleases.
func TestProperty_ParseReleasesFiltersDraftsAndPrereleases(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	genReleases := gen.SliceOf(genGitHubRelease())

	properties.Property("parseReleases filters out drafts and prereleases", prop.ForAll(
		func(releases []GitHubRelease) bool {
			fetcher := NewCodexFetcher()
			result := fetcher.parseReleases(releases)

			// Count expected releases (non-draft, non-prerelease)
			expectedCount := 0
			for _, r := range releases {
				if !r.Draft && !r.Prerelease {
					expectedCount++
				}
			}

			if len(result.Releases) != expectedCount {
				t.Logf("Expected %d releases, got %d", expectedCount, len(result.Releases))
				return false
			}

			return true
		},
		genReleases,
	))

	properties.TestingRun(t)
}

// Unit test for normalizeVersion
func TestNormalizeVersion(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"v1.0.0", "1.0.0"},
		{"1.0.0", "1.0.0"},
		{"v2.3.4", "2.3.4"},
		{"  v1.0.0  ", "1.0.0"},
		{"", ""},
		{"v", ""},
	}

	for _, tc := range testCases {
		result := normalizeVersion(tc.input)
		if result != tc.expected {
			t.Errorf("normalizeVersion(%q) = %q, expected %q", tc.input, result, tc.expected)
		}
	}
}

// Unit test for parseReleases with empty input
func TestParseReleases_Empty(t *testing.T) {
	fetcher := NewCodexFetcher()
	result := fetcher.parseReleases([]GitHubRelease{})

	if result.CLIName != "codex" {
		t.Errorf("Expected CLIName 'codex', got '%s'", result.CLIName)
	}

	if result.LatestVersion != "" {
		t.Errorf("Expected empty LatestVersion, got '%s'", result.LatestVersion)
	}

	if len(result.Releases) != 0 {
		t.Errorf("Expected 0 releases, got %d", len(result.Releases))
	}
}

// Unit test for parseReleases with mixed releases
func TestParseReleases_MixedReleases(t *testing.T) {
	fetcher := NewGeminiFetcher()
	releases := []GitHubRelease{
		{TagName: "v1.0.0", Body: "First release", PublishedAt: time.Now(), HTMLURL: "https://example.com/1", Draft: false, Prerelease: false},
		{TagName: "v1.1.0-beta", Body: "Beta release", PublishedAt: time.Now(), HTMLURL: "https://example.com/2", Draft: false, Prerelease: true},
		{TagName: "v1.2.0", Body: "Second release", PublishedAt: time.Now(), HTMLURL: "https://example.com/3", Draft: true, Prerelease: false},
		{TagName: "v2.0.0", Body: "Major release", PublishedAt: time.Now(), HTMLURL: "https://example.com/4", Draft: false, Prerelease: false},
	}

	result := fetcher.parseReleases(releases)

	// Should only include non-draft, non-prerelease (v1.0.0 and v2.0.0)
	if len(result.Releases) != 2 {
		t.Errorf("Expected 2 releases, got %d", len(result.Releases))
	}

	// First release should be v1.0.0 (first in input order)
	if result.LatestVersion != "1.0.0" {
		t.Errorf("Expected LatestVersion '1.0.0', got '%s'", result.LatestVersion)
	}
}
