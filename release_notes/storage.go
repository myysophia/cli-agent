package release_notes

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Storage handles file persistence for release notes
type Storage struct {
	config PersistenceConfig
}

// NewStorage creates a new storage instance with the given configuration
func NewStorage(config PersistenceConfig) *Storage {
	if config.StoragePath == "" {
		config.StoragePath = "data/release_notes.json"
	}
	if config.BackupCount <= 0 {
		config.BackupCount = 3
	}
	return &Storage{
		config: config,
	}
}

// DefaultPersistenceConfig returns the default persistence configuration
func DefaultPersistenceConfig() PersistenceConfig {
	return PersistenceConfig{
		StoragePath:   "data/release_notes.json",
		WriteOnUpdate: true,
		BackupEnabled: true,
		BackupCount:   3,
	}
}

// SaveToStorage saves release notes to file using atomic write (temp file + rename)
func (s *Storage) SaveToStorage(data *AllReleaseNotes) error {
	if data == nil {
		return fmt.Errorf("cannot save nil data")
	}

	// Ensure directory exists
	dir := filepath.Dir(s.config.StoragePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Create backup before writing if enabled and file exists
	if s.config.BackupEnabled {
		if _, err := os.Stat(s.config.StoragePath); err == nil {
			if err := s.rotateBackups(); err != nil {
				// Log but don't fail on backup error
				fmt.Printf("warning: failed to create backup: %v\n", err)
			}
		}
	}

	// Marshal data to JSON with indentation for readability
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	// Write to temporary file first (atomic write)
	tempFile := s.config.StoragePath + ".tmp"
	if err := os.WriteFile(tempFile, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	// Rename temp file to target (atomic operation on most filesystems)
	if err := os.Rename(tempFile, s.config.StoragePath); err != nil {
		// Clean up temp file on failure
		os.Remove(tempFile)
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	return nil
}


// LoadFromStorage loads release notes from file with JSON validation.
// Implements fault recovery: tries main file first, then backup files.
// If all fail, logs errors and returns an empty cache to allow the system to continue.
func (s *Storage) LoadFromStorage() (*AllReleaseNotes, error) {
	// Try to load from main file first
	data, err := s.loadFromFile(s.config.StoragePath)
	if err == nil {
		log.Printf("✅ Loaded release notes from %s", s.config.StoragePath)
		return data, nil
	}

	// Log the main file error
	log.Printf("⚠️  Failed to load main storage file %s: %v", s.config.StoragePath, err)

	// If main file failed and backups are enabled, try backup files
	if s.config.BackupEnabled {
		log.Printf("🔄 Attempting to recover from backup files...")
		backupData, backupErr := s.loadFromBackups()
		if backupErr == nil {
			log.Printf("✅ Successfully recovered from backup file")
			return backupData, nil
		}
		log.Printf("⚠️  Failed to recover from backups: %v", backupErr)
	}

	// All recovery attempts failed - return empty cache and continue
	log.Printf("⚠️  All storage recovery attempts failed, starting with empty cache")
	return s.createEmptyCache(), nil
}

// LoadFromStorageWithFaultRecovery loads release notes with comprehensive fault recovery.
// This is an alias for LoadFromStorage that makes the fault recovery behavior explicit.
func (s *Storage) LoadFromStorageWithFaultRecovery() (*AllReleaseNotes, error) {
	return s.LoadFromStorage()
}

// createEmptyCache creates an empty AllReleaseNotes structure
func (s *Storage) createEmptyCache() *AllReleaseNotes {
	return &AllReleaseNotes{
		CLIs:        make(map[string]*CLIReleaseNotes),
		LastUpdated: time.Time{}, // Zero time indicates never updated
	}
}

// loadFromFile loads and validates JSON from a specific file
func (s *Storage) loadFromFile(path string) (*AllReleaseNotes, error) {
	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("file does not exist: %s", path)
	}

	// Read file content
	jsonData, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", path, err)
	}

	// Validate JSON by unmarshaling
	var data AllReleaseNotes
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("invalid JSON in file %s: %w", path, err)
	}

	// Validate required fields
	if err := s.validateData(&data); err != nil {
		return nil, fmt.Errorf("validation failed for %s: %w", path, err)
	}

	return &data, nil
}

// validateData validates the loaded data has required fields
func (s *Storage) validateData(data *AllReleaseNotes) error {
	if data.CLIs == nil {
		data.CLIs = make(map[string]*CLIReleaseNotes)
	}

	for name, cli := range data.CLIs {
		if cli == nil {
			return fmt.Errorf("nil CLI entry for %s", name)
		}
		if cli.CLIName == "" {
			cli.CLIName = name
		}
		if cli.Releases == nil {
			cli.Releases = []ReleaseNote{}
		}
	}

	return nil
}

// loadFromBackups attempts to load from backup files in order (newest first)
func (s *Storage) loadFromBackups() (*AllReleaseNotes, error) {
	backups := s.getBackupFiles()
	if len(backups) == 0 {
		return nil, fmt.Errorf("no backup files found")
	}

	log.Printf("📂 Found %d backup file(s) to try", len(backups))

	var lastErr error
	for i, backup := range backups {
		log.Printf("🔍 Trying backup file %d/%d: %s", i+1, len(backups), backup)
		data, err := s.loadFromFile(backup)
		if err == nil {
			log.Printf("✅ Successfully loaded from backup: %s", backup)
			return data, nil
		}
		log.Printf("⚠️  Backup file %s failed: %v", backup, err)
		lastErr = err
	}

	return nil, fmt.Errorf("all %d backup files failed: %w", len(backups), lastErr)
}

// rotateBackups creates a new backup and removes old ones beyond BackupCount
func (s *Storage) rotateBackups() error {
	// Get existing backups sorted by number (newest first)
	backups := s.getBackupFiles()

	// Remove oldest backups if we have too many
	for len(backups) >= s.config.BackupCount {
		oldest := backups[len(backups)-1]
		if err := os.Remove(oldest); err != nil {
			return fmt.Errorf("failed to remove old backup %s: %w", oldest, err)
		}
		backups = backups[:len(backups)-1]
	}

	// Shift existing backups (bak.1 -> bak.2, bak.2 -> bak.3, etc.)
	for i := len(backups) - 1; i >= 0; i-- {
		oldPath := backups[i]
		newNum := s.getBackupNumber(oldPath) + 1
		newPath := fmt.Sprintf("%s.bak.%d", s.config.StoragePath, newNum)
		if err := os.Rename(oldPath, newPath); err != nil {
			return fmt.Errorf("failed to rotate backup %s to %s: %w", oldPath, newPath, err)
		}
	}

	// Copy current file to bak.1
	if _, err := os.Stat(s.config.StoragePath); err == nil {
		backupPath := s.config.StoragePath + ".bak.1"
		if err := s.copyFile(s.config.StoragePath, backupPath); err != nil {
			return fmt.Errorf("failed to create backup: %w", err)
		}
	}

	return nil
}


// getBackupFiles returns a list of backup files sorted by number (newest first)
func (s *Storage) getBackupFiles() []string {
	dir := filepath.Dir(s.config.StoragePath)
	base := filepath.Base(s.config.StoragePath)
	pattern := base + ".bak.*"

	matches, err := filepath.Glob(filepath.Join(dir, pattern))
	if err != nil || len(matches) == 0 {
		return nil
	}

	// Sort by backup number (ascending, so bak.1 comes first)
	sort.Slice(matches, func(i, j int) bool {
		numI := s.getBackupNumber(matches[i])
		numJ := s.getBackupNumber(matches[j])
		return numI < numJ
	})

	return matches
}

// getBackupNumber extracts the backup number from a backup filename
func (s *Storage) getBackupNumber(path string) int {
	// Extract number from path like "file.json.bak.1"
	parts := strings.Split(path, ".bak.")
	if len(parts) < 2 {
		return 0
	}
	var num int
	fmt.Sscanf(parts[len(parts)-1], "%d", &num)
	return num
}

// copyFile copies a file from src to dst
func (s *Storage) copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}

// Exists checks if the storage file exists
func (s *Storage) Exists() bool {
	_, err := os.Stat(s.config.StoragePath)
	return err == nil
}

// Path returns the storage file path
func (s *Storage) Path() string {
	return s.config.StoragePath
}

// BackupCount returns the number of existing backup files
func (s *Storage) BackupCount() int {
	return len(s.getBackupFiles())
}

// Delete removes the storage file and all backups
func (s *Storage) Delete() error {
	// Remove main file
	if err := os.Remove(s.config.StoragePath); err != nil && !os.IsNotExist(err) {
		return err
	}

	// Remove all backups
	for _, backup := range s.getBackupFiles() {
		if err := os.Remove(backup); err != nil && !os.IsNotExist(err) {
			return err
		}
	}

	return nil
}

// IsEmptyCache checks if the given AllReleaseNotes represents an empty cache
// (typically returned when all recovery attempts failed)
func IsEmptyCache(data *AllReleaseNotes) bool {
	if data == nil {
		return true
	}
	return len(data.CLIs) == 0 && data.LastUpdated.IsZero()
}

// RecoveryStatus represents the result of a storage recovery attempt
type RecoveryStatus struct {
	Success      bool   // Whether data was successfully loaded
	Source       string // Source of the data: "main", "backup", "empty"
	BackupNumber int    // If recovered from backup, which backup number (1, 2, 3...)
	Error        error  // The last error encountered (if any)
}

// LoadFromStorageWithStatus loads release notes and returns detailed recovery status
func (s *Storage) LoadFromStorageWithStatus() (*AllReleaseNotes, RecoveryStatus) {
	status := RecoveryStatus{}

	// Try to load from main file first
	data, err := s.loadFromFile(s.config.StoragePath)
	if err == nil {
		log.Printf("✅ Loaded release notes from %s", s.config.StoragePath)
		status.Success = true
		status.Source = "main"
		return data, status
	}

	// Log the main file error
	log.Printf("⚠️  Failed to load main storage file %s: %v", s.config.StoragePath, err)
	status.Error = err

	// If main file failed and backups are enabled, try backup files
	if s.config.BackupEnabled {
		log.Printf("🔄 Attempting to recover from backup files...")
		backups := s.getBackupFiles()

		for i, backup := range backups {
			log.Printf("🔍 Trying backup file %d/%d: %s", i+1, len(backups), backup)
			backupData, backupErr := s.loadFromFile(backup)
			if backupErr == nil {
				log.Printf("✅ Successfully loaded from backup: %s", backup)
				status.Success = true
				status.Source = "backup"
				status.BackupNumber = s.getBackupNumber(backup)
				return backupData, status
			}
			log.Printf("⚠️  Backup file %s failed: %v", backup, backupErr)
			status.Error = backupErr
		}
	}

	// All recovery attempts failed - return empty cache and continue
	log.Printf("⚠️  All storage recovery attempts failed, starting with empty cache")
	status.Success = false
	status.Source = "empty"
	return s.createEmptyCache(), status
}
