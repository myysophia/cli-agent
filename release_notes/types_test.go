package release_notes

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// genReleaseNote generates arbitrary ReleaseNote values
func genReleaseNote() gopter.Gen {
	return gopter.CombineGens(
		gen.AlphaString(),                    // Version
		gen.TimeRange(time.Now().Add(-365*24*time.Hour), 365*24*time.Hour), // ReleaseDate
		gen.AlphaString(),                    // Changelog
		gen.AlphaString(),                    // URL
	).Map(func(values []interface{}) ReleaseNote {
		return ReleaseNote{
			Version:     values[0].(string),
			ReleaseDate: values[1].(time.Time),
			Changelog:   values[2].(string),
			URL:         values[3].(string),
		}
	})
}

// genCLIReleaseNotes generates arbitrary CLIReleaseNotes values
func genCLIReleaseNotes() gopter.Gen {
	return gopter.CombineGens(
		gen.AlphaString(),                    // CLIName
		gen.AlphaString(),                    // DisplayName
		gen.AlphaString(),                    // LatestVersion
		gen.AlphaString(),                    // LocalVersion
		gen.Bool(),                           // UpdateAvailable
		gen.TimeRange(time.Now().Add(-365*24*time.Hour), 365*24*time.Hour), // LastUpdated
		gen.SliceOf(genReleaseNote()),        // Releases
	).Map(func(values []interface{}) CLIReleaseNotes {
		return CLIReleaseNotes{
			CLIName:         values[0].(string),
			DisplayName:     values[1].(string),
			LatestVersion:   values[2].(string),
			LocalVersion:    values[3].(string),
			UpdateAvailable: values[4].(bool),
			LastUpdated:     values[5].(time.Time),
			Releases:        values[6].([]ReleaseNote),
		}
	})
}

// **Feature: cli-release-notes, Property 9: JSON serialization round-trip**
// **Validates: Requirements 5.4, 5.5**
// For any valid CLIReleaseNotes struct, serializing to JSON and deserializing
// back SHALL produce an equivalent struct.
func TestProperty_JSONSerializationRoundTrip(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	properties.Property("JSON serialization round-trip preserves CLIReleaseNotes", prop.ForAll(
		func(original CLIReleaseNotes) bool {
			// Serialize to JSON
			jsonBytes, err := json.Marshal(original)
			if err != nil {
				return false
			}

			// Deserialize back
			var deserialized CLIReleaseNotes
			err = json.Unmarshal(jsonBytes, &deserialized)
			if err != nil {
				return false
			}

			// Compare - need to handle time comparison specially
			// JSON serialization may lose nanosecond precision
			return compareReleaseNotes(original, deserialized)
		},
		genCLIReleaseNotes(),
	))

	properties.TestingRun(t)
}

// compareReleaseNotes compares two CLIReleaseNotes accounting for time precision loss
func compareReleaseNotes(a, b CLIReleaseNotes) bool {
	if a.CLIName != b.CLIName ||
		a.DisplayName != b.DisplayName ||
		a.LatestVersion != b.LatestVersion ||
		a.LocalVersion != b.LocalVersion ||
		a.UpdateAvailable != b.UpdateAvailable {
		return false
	}

	// Compare times with second precision (JSON loses nanoseconds)
	if !timesEqualToSecond(a.LastUpdated, b.LastUpdated) {
		return false
	}

	// Compare releases
	if len(a.Releases) != len(b.Releases) {
		return false
	}

	for i := range a.Releases {
		if !compareReleaseNote(a.Releases[i], b.Releases[i]) {
			return false
		}
	}

	return true
}

// compareReleaseNote compares two ReleaseNote values
func compareReleaseNote(a, b ReleaseNote) bool {
	return a.Version == b.Version &&
		timesEqualToSecond(a.ReleaseDate, b.ReleaseDate) &&
		a.Changelog == b.Changelog &&
		a.URL == b.URL
}

// timesEqualToSecond compares two times with second precision
func timesEqualToSecond(a, b time.Time) bool {
	return a.Unix() == b.Unix()
}

// **Feature: cli-release-notes, Property 9: JSON serialization round-trip (ReleaseNote)**
// **Validates: Requirements 5.4, 5.5**
// For any valid ReleaseNote struct, serializing to JSON and deserializing
// back SHALL produce an equivalent struct.
func TestProperty_JSONSerializationRoundTrip_ReleaseNote(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	properties.Property("JSON serialization round-trip preserves ReleaseNote", prop.ForAll(
		func(original ReleaseNote) bool {
			// Serialize to JSON
			jsonBytes, err := json.Marshal(original)
			if err != nil {
				return false
			}

			// Deserialize back
			var deserialized ReleaseNote
			err = json.Unmarshal(jsonBytes, &deserialized)
			if err != nil {
				return false
			}

			return compareReleaseNote(original, deserialized)
		},
		genReleaseNote(),
	))

	properties.TestingRun(t)
}

// **Feature: cli-release-notes, Property 9: JSON serialization round-trip (AllReleaseNotes)**
// **Validates: Requirements 5.4, 5.5**
// For any valid AllReleaseNotes struct, serializing to JSON and deserializing
// back SHALL produce an equivalent struct.
func TestProperty_JSONSerializationRoundTrip_AllReleaseNotes(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	// Generate AllReleaseNotes with a map of CLIReleaseNotes
	genAllReleaseNotes := gopter.CombineGens(
		gen.MapOf(gen.AlphaString(), genCLIReleaseNotes().Map(func(v CLIReleaseNotes) *CLIReleaseNotes { return &v })),
		gen.TimeRange(time.Now().Add(-365*24*time.Hour), 365*24*time.Hour),
	).Map(func(values []interface{}) AllReleaseNotes {
		return AllReleaseNotes{
			CLIs:        values[0].(map[string]*CLIReleaseNotes),
			LastUpdated: values[1].(time.Time),
		}
	})

	properties.Property("JSON serialization round-trip preserves AllReleaseNotes", prop.ForAll(
		func(original AllReleaseNotes) bool {
			// Serialize to JSON
			jsonBytes, err := json.Marshal(original)
			if err != nil {
				return false
			}

			// Deserialize back
			var deserialized AllReleaseNotes
			err = json.Unmarshal(jsonBytes, &deserialized)
			if err != nil {
				return false
			}

			// Compare LastUpdated
			if !timesEqualToSecond(original.LastUpdated, deserialized.LastUpdated) {
				return false
			}

			// Compare CLIs map
			if len(original.CLIs) != len(deserialized.CLIs) {
				return false
			}

			for key, origVal := range original.CLIs {
				deserVal, exists := deserialized.CLIs[key]
				if !exists {
					return false
				}
				if origVal == nil && deserVal == nil {
					continue
				}
				if origVal == nil || deserVal == nil {
					return false
				}
				if !compareReleaseNotes(*origVal, *deserVal) {
					return false
				}
			}

			return true
		},
		genAllReleaseNotes,
	))

	properties.TestingRun(t)
}

// Unit test for basic JSON serialization
func TestJSONSerialization_Basic(t *testing.T) {
	original := CLIReleaseNotes{
		CLIName:         "claude",
		DisplayName:     "Claude CLI",
		LatestVersion:   "2.0.53",
		LocalVersion:    "2.0.30",
		UpdateAvailable: true,
		LastUpdated:     time.Date(2025, 11, 25, 10, 0, 0, 0, time.UTC),
		Releases: []ReleaseNote{
			{
				Version:     "2.0.53",
				ReleaseDate: time.Date(2025, 11, 25, 1, 12, 33, 0, time.UTC),
				Changelog:   "- Bug fixes\n- Performance improvements",
				URL:         "https://github.com/anthropics/claude-code/releases/tag/v2.0.53",
			},
		},
	}

	// Serialize
	jsonBytes, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	// Deserialize
	var deserialized CLIReleaseNotes
	err = json.Unmarshal(jsonBytes, &deserialized)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	// Verify fields
	if !reflect.DeepEqual(original, deserialized) {
		t.Errorf("Round-trip failed:\nOriginal: %+v\nDeserialized: %+v", original, deserialized)
	}
}
