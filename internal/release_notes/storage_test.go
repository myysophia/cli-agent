package release_notes

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// genReleaseNoteForStorage generates arbitrary ReleaseNote values for storage tests
func genReleaseNoteForStorage() gopter.Gen {
	return gopter.CombineGens(
		gen.AlphaString(),
		gen.TimeRange(time.Now().Add(-365*24*time.Hour), 365*24*time.Hour),
		gen.AlphaString(),
		gen.AlphaString(),
	).Map(func(values []interface{}) ReleaseNote {
		return ReleaseNote{
			Version:     values[0].(string),
			ReleaseDate: values[1].(time.Time),
			Changelog:   values[2].(string),
			URL:         values[3].(string),
		}
	})
}

// genCLIReleaseNotesForStorage generates arbitrary CLIReleaseNotes values for storage tests
func genCLIReleaseNotesForStorage() gopter.Gen {
	return gopter.CombineGens(
		gen.AlphaString(),
		gen.AlphaString(),
		gen.AlphaString(),
		gen.AlphaString(),
		gen.Bool(),
		gen.TimeRange(time.Now().Add(-365*24*time.Hour), 365*24*time.Hour),
		gen.SliceOf(genReleaseNoteForStorage()),
	).Map(func(values []interface{}) *CLIReleaseNotes {
		return &CLIReleaseNotes{
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

// genNonEmptyAlphaString generates non-empty alpha strings for valid CLI names
func genNonEmptyAlphaString() gopter.Gen {
	return gen.AlphaString().SuchThat(func(s string) bool {
		return len(s) > 0
	})
}

// genAllReleaseNotesForStorage generates arbitrary AllReleaseNotes values for storage tests
// It ensures CLIName matches the map key for valid round-trip behavior
func genAllReleaseNotesForStorage() gopter.Gen {
	return gopter.CombineGens(
		gen.MapOf(genNonEmptyAlphaString(), genCLIReleaseNotesForStorage()),
		gen.TimeRange(time.Now().Add(-365*24*time.Hour), 365*24*time.Hour),
	).Map(func(values []interface{}) *AllReleaseNotes {
		clis := values[0].(map[string]*CLIReleaseNotes)
		// Ensure CLIName matches the map key for valid data
		for key, cli := range clis {
			if cli != nil {
				cli.CLIName = key
			}
		}
		return &AllReleaseNotes{
			CLIs:        clis,
			LastUpdated: values[1].(time.Time),
		}
	})
}

// **Feature: cli-release-notes, Property 13: Persistent storage round-trip**
// **Validates: Requirements 8.4**
// For any cached release notes, saving to file and loading back SHALL produce equivalent data.
func TestProperty_PersistentStorageRoundTrip(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	properties.Property("persistent storage round-trip preserves AllReleaseNotes", prop.ForAll(
		func(original *AllReleaseNotes) bool {
			// Create a temporary directory for this test
			tempDir, err := os.MkdirTemp("", "storage_test_*")
			if err != nil {
				t.Logf("Failed to create temp dir: %v", err)
				return false
			}
			defer os.RemoveAll(tempDir)

			// Create storage with temp path
			storagePath := filepath.Join(tempDir, "release_notes.json")
			storage := NewStorage(PersistenceConfig{
				StoragePath:   storagePath,
				WriteOnUpdate: true,
				BackupEnabled: false, // Disable backups for simpler round-trip test
				BackupCount:   0,
			})

			// Save to storage
			if err := storage.SaveToStorage(original); err != nil {
				t.Logf("Failed to save: %v", err)
				return false
			}

			// Load from storage
			loaded, err := storage.LoadFromStorage()
			if err != nil {
				t.Logf("Failed to load: %v", err)
				return false
			}

			// Compare the data
			return compareAllReleaseNotes(original, loaded)
		},
		genAllReleaseNotesForStorage(),
	))

	properties.TestingRun(t)
}

// compareAllReleaseNotes compares two AllReleaseNotes accounting for time precision loss
func compareAllReleaseNotes(a, b *AllReleaseNotes) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	// Compare LastUpdated with second precision (JSON loses nanoseconds)
	if !timesEqualToSecondStorage(a.LastUpdated, b.LastUpdated) {
		return false
	}

	// Compare CLIs map size
	if len(a.CLIs) != len(b.CLIs) {
		return false
	}

	// Compare each CLI entry
	for key, aVal := range a.CLIs {
		bVal, exists := b.CLIs[key]
		if !exists {
			return false
		}
		if aVal == nil && bVal == nil {
			continue
		}
		if aVal == nil || bVal == nil {
			return false
		}
		if !compareCLIReleaseNotesStorage(*aVal, *bVal) {
			return false
		}
	}

	return true
}

// compareCLIReleaseNotesStorage compares two CLIReleaseNotes accounting for time precision loss
func compareCLIReleaseNotesStorage(a, b CLIReleaseNotes) bool {
	if a.CLIName != b.CLIName ||
		a.DisplayName != b.DisplayName ||
		a.LatestVersion != b.LatestVersion ||
		a.LocalVersion != b.LocalVersion ||
		a.UpdateAvailable != b.UpdateAvailable {
		return false
	}

	// Compare times with second precision (JSON loses nanoseconds)
	if !timesEqualToSecondStorage(a.LastUpdated, b.LastUpdated) {
		return false
	}

	// Compare releases
	if len(a.Releases) != len(b.Releases) {
		return false
	}

	for i := range a.Releases {
		if !compareReleaseNoteStorage(a.Releases[i], b.Releases[i]) {
			return false
		}
	}

	return true
}

// compareReleaseNoteStorage compares two ReleaseNote values
func compareReleaseNoteStorage(a, b ReleaseNote) bool {
	return a.Version == b.Version &&
		timesEqualToSecondStorage(a.ReleaseDate, b.ReleaseDate) &&
		a.Changelog == b.Changelog &&
		a.URL == b.URL
}

// timesEqualToSecondStorage compares two times with second precision
func timesEqualToSecondStorage(a, b time.Time) bool {
	return a.Unix() == b.Unix()
}

// Unit tests for storage operations

func TestStorage_SaveAndLoad_Basic(t *testing.T) {
	// Create a temporary directory
	tempDir, err := os.MkdirTemp("", "storage_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	storagePath := filepath.Join(tempDir, "release_notes.json")
	storage := NewStorage(PersistenceConfig{
		StoragePath:   storagePath,
		WriteOnUpdate: true,
		BackupEnabled: false,
	})

	original := &AllReleaseNotes{
		CLIs: map[string]*CLIReleaseNotes{
			"claude": {
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
						URL:         "https://example.com/release",
					},
				},
			},
		},
		LastUpdated: time.Date(2025, 11, 25, 10, 0, 0, 0, time.UTC),
	}

	// Save
	if err := storage.SaveToStorage(original); err != nil {
		t.Fatalf("Failed to save: %v", err)
	}

	// Verify file exists
	if !storage.Exists() {
		t.Error("Storage file should exist after save")
	}

	// Load
	loaded, err := storage.LoadFromStorage()
	if err != nil {
		t.Fatalf("Failed to load: %v", err)
	}

	// Verify data
	if len(loaded.CLIs) != 1 {
		t.Errorf("Expected 1 CLI, got %d", len(loaded.CLIs))
	}

	claude, exists := loaded.CLIs["claude"]
	if !exists {
		t.Fatal("Expected 'claude' CLI to exist")
	}

	if claude.LatestVersion != "2.0.53" {
		t.Errorf("Expected version '2.0.53', got '%s'", claude.LatestVersion)
	}

	if claude.UpdateAvailable != true {
		t.Error("Expected UpdateAvailable to be true")
	}
}

func TestStorage_SaveNilData(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "storage_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	storage := NewStorage(PersistenceConfig{
		StoragePath: filepath.Join(tempDir, "release_notes.json"),
	})

	err = storage.SaveToStorage(nil)
	if err == nil {
		t.Error("Expected error when saving nil data")
	}
}

func TestStorage_LoadNonExistent(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "storage_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	storage := NewStorage(PersistenceConfig{
		StoragePath:   filepath.Join(tempDir, "nonexistent.json"),
		BackupEnabled: false,
	})

	// Should return empty cache when file doesn't exist
	loaded, err := storage.LoadFromStorage()
	if err != nil {
		t.Fatalf("LoadFromStorage should not return error for non-existent file: %v", err)
	}

	if !IsEmptyCache(loaded) {
		t.Error("Expected empty cache for non-existent file")
	}
}

func TestStorage_BackupRotation(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "storage_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	storage := NewStorage(PersistenceConfig{
		StoragePath:   filepath.Join(tempDir, "release_notes.json"),
		WriteOnUpdate: true,
		BackupEnabled: true,
		BackupCount:   3,
	})

	// Save multiple times to trigger backup rotation
	for i := 0; i < 5; i++ {
		data := &AllReleaseNotes{
			CLIs: map[string]*CLIReleaseNotes{
				"test": {
					CLIName:       "test",
					LatestVersion: string(rune('0' + i)),
				},
			},
			LastUpdated: time.Now(),
		}
		if err := storage.SaveToStorage(data); err != nil {
			t.Fatalf("Failed to save iteration %d: %v", i, err)
		}
	}

	// Should have at most BackupCount backups
	backupCount := storage.BackupCount()
	if backupCount > 3 {
		t.Errorf("Expected at most 3 backups, got %d", backupCount)
	}
}

func TestStorage_Delete(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "storage_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	storage := NewStorage(PersistenceConfig{
		StoragePath:   filepath.Join(tempDir, "release_notes.json"),
		BackupEnabled: true,
		BackupCount:   3,
	})

	// Save some data
	data := &AllReleaseNotes{
		CLIs:        map[string]*CLIReleaseNotes{},
		LastUpdated: time.Now(),
	}
	if err := storage.SaveToStorage(data); err != nil {
		t.Fatalf("Failed to save: %v", err)
	}

	if !storage.Exists() {
		t.Error("Storage should exist after save")
	}

	// Delete
	if err := storage.Delete(); err != nil {
		t.Fatalf("Failed to delete: %v", err)
	}

	if storage.Exists() {
		t.Error("Storage should not exist after delete")
	}
}


// Fault Recovery Tests - Task 7.2

func TestStorage_FaultRecovery_CorruptedMainFile_LoadsFromBackup(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "storage_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	storagePath := filepath.Join(tempDir, "release_notes.json")
	storage := NewStorage(PersistenceConfig{
		StoragePath:   storagePath,
		WriteOnUpdate: true,
		BackupEnabled: true,
		BackupCount:   3,
	})

	// Save valid data first
	validData := &AllReleaseNotes{
		CLIs: map[string]*CLIReleaseNotes{
			"claude": {
				CLIName:       "claude",
				DisplayName:   "Claude CLI",
				LatestVersion: "2.0.53",
			},
		},
		LastUpdated: time.Date(2025, 11, 25, 10, 0, 0, 0, time.UTC),
	}
	if err := storage.SaveToStorage(validData); err != nil {
		t.Fatalf("Failed to save valid data: %v", err)
	}

	// Save again to create a backup
	validData.CLIs["claude"].LatestVersion = "2.0.54"
	if err := storage.SaveToStorage(validData); err != nil {
		t.Fatalf("Failed to save second version: %v", err)
	}

	// Corrupt the main file
	if err := os.WriteFile(storagePath, []byte("invalid json {{{"), 0644); err != nil {
		t.Fatalf("Failed to corrupt main file: %v", err)
	}

	// Load should recover from backup
	loaded, err := storage.LoadFromStorage()
	if err != nil {
		t.Fatalf("LoadFromStorage should not return error: %v", err)
	}

	// Should have loaded from backup (version 2.0.53)
	if loaded == nil || len(loaded.CLIs) == 0 {
		t.Fatal("Expected to load data from backup")
	}

	claude, exists := loaded.CLIs["claude"]
	if !exists {
		t.Fatal("Expected 'claude' CLI to exist")
	}

	// Backup should have the older version
	if claude.LatestVersion != "2.0.53" {
		t.Errorf("Expected version '2.0.53' from backup, got '%s'", claude.LatestVersion)
	}
}

func TestStorage_FaultRecovery_AllFilesCorrupted_ReturnsEmptyCache(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "storage_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	storagePath := filepath.Join(tempDir, "release_notes.json")
	storage := NewStorage(PersistenceConfig{
		StoragePath:   storagePath,
		WriteOnUpdate: true,
		BackupEnabled: true,
		BackupCount:   3,
	})

	// Create corrupted main file
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		t.Fatalf("Failed to create dir: %v", err)
	}
	if err := os.WriteFile(storagePath, []byte("corrupted main"), 0644); err != nil {
		t.Fatalf("Failed to create corrupted main file: %v", err)
	}

	// Create corrupted backup files
	for i := 1; i <= 3; i++ {
		backupPath := filepath.Join(tempDir, "release_notes.json.bak."+string(rune('0'+i)))
		if err := os.WriteFile(backupPath, []byte("corrupted backup"), 0644); err != nil {
			t.Fatalf("Failed to create corrupted backup %d: %v", i, err)
		}
	}

	// Load should return empty cache without error
	loaded, err := storage.LoadFromStorage()
	if err != nil {
		t.Fatalf("LoadFromStorage should not return error: %v", err)
	}

	// Should return empty cache
	if !IsEmptyCache(loaded) {
		t.Error("Expected empty cache when all files are corrupted")
	}
}

func TestStorage_FaultRecovery_WithStatus_MainFileSuccess(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "storage_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	storagePath := filepath.Join(tempDir, "release_notes.json")
	storage := NewStorage(PersistenceConfig{
		StoragePath:   storagePath,
		BackupEnabled: true,
		BackupCount:   3,
	})

	// Save valid data
	validData := &AllReleaseNotes{
		CLIs: map[string]*CLIReleaseNotes{
			"claude": {CLIName: "claude", LatestVersion: "2.0.53"},
		},
		LastUpdated: time.Now(),
	}
	if err := storage.SaveToStorage(validData); err != nil {
		t.Fatalf("Failed to save: %v", err)
	}

	// Load with status
	loaded, status := storage.LoadFromStorageWithStatus()

	if !status.Success {
		t.Error("Expected success=true")
	}
	if status.Source != "main" {
		t.Errorf("Expected source='main', got '%s'", status.Source)
	}
	if loaded == nil || len(loaded.CLIs) == 0 {
		t.Error("Expected loaded data")
	}
}

func TestStorage_FaultRecovery_WithStatus_BackupRecovery(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "storage_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	storagePath := filepath.Join(tempDir, "release_notes.json")
	storage := NewStorage(PersistenceConfig{
		StoragePath:   storagePath,
		BackupEnabled: true,
		BackupCount:   3,
	})

	// Save valid data twice to create backup
	validData := &AllReleaseNotes{
		CLIs: map[string]*CLIReleaseNotes{
			"claude": {CLIName: "claude", LatestVersion: "2.0.53"},
		},
		LastUpdated: time.Now(),
	}
	if err := storage.SaveToStorage(validData); err != nil {
		t.Fatalf("Failed to save first: %v", err)
	}
	validData.CLIs["claude"].LatestVersion = "2.0.54"
	if err := storage.SaveToStorage(validData); err != nil {
		t.Fatalf("Failed to save second: %v", err)
	}

	// Corrupt main file
	if err := os.WriteFile(storagePath, []byte("corrupted"), 0644); err != nil {
		t.Fatalf("Failed to corrupt: %v", err)
	}

	// Load with status
	loaded, status := storage.LoadFromStorageWithStatus()

	if !status.Success {
		t.Error("Expected success=true (recovered from backup)")
	}
	if status.Source != "backup" {
		t.Errorf("Expected source='backup', got '%s'", status.Source)
	}
	if status.BackupNumber != 1 {
		t.Errorf("Expected backup number 1, got %d", status.BackupNumber)
	}
	if loaded == nil || len(loaded.CLIs) == 0 {
		t.Error("Expected loaded data from backup")
	}
}

func TestStorage_FaultRecovery_WithStatus_AllFailed(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "storage_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	storagePath := filepath.Join(tempDir, "release_notes.json")
	storage := NewStorage(PersistenceConfig{
		StoragePath:   storagePath,
		BackupEnabled: true,
		BackupCount:   3,
	})

	// Create corrupted main file
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		t.Fatalf("Failed to create dir: %v", err)
	}
	if err := os.WriteFile(storagePath, []byte("corrupted"), 0644); err != nil {
		t.Fatalf("Failed to create corrupted file: %v", err)
	}

	// Load with status
	loaded, status := storage.LoadFromStorageWithStatus()

	if status.Success {
		t.Error("Expected success=false when all files corrupted")
	}
	if status.Source != "empty" {
		t.Errorf("Expected source='empty', got '%s'", status.Source)
	}
	if !IsEmptyCache(loaded) {
		t.Error("Expected empty cache")
	}
}

func TestStorage_IsEmptyCache(t *testing.T) {
	tests := []struct {
		name     string
		data     *AllReleaseNotes
		expected bool
	}{
		{
			name:     "nil data",
			data:     nil,
			expected: true,
		},
		{
			name: "empty CLIs and zero time",
			data: &AllReleaseNotes{
				CLIs:        make(map[string]*CLIReleaseNotes),
				LastUpdated: time.Time{},
			},
			expected: true,
		},
		{
			name: "empty CLIs but non-zero time",
			data: &AllReleaseNotes{
				CLIs:        make(map[string]*CLIReleaseNotes),
				LastUpdated: time.Now(),
			},
			expected: false,
		},
		{
			name: "non-empty CLIs",
			data: &AllReleaseNotes{
				CLIs: map[string]*CLIReleaseNotes{
					"claude": {CLIName: "claude"},
				},
				LastUpdated: time.Time{},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsEmptyCache(tt.data)
			if result != tt.expected {
				t.Errorf("IsEmptyCache() = %v, expected %v", result, tt.expected)
			}
		})
	}
}
