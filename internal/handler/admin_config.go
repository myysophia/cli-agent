package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type AdminConfigMeta struct {
	Exists     bool     `json:"exists"`
	Source     string   `json:"source"`
	LastLoaded string   `json:"last_loaded,omitempty"`
	Warnings   []string `json:"warnings,omitempty"`
}

type AdminConfigResponse struct {
	Config *Config         `json:"config,omitempty"`
	Meta   AdminConfigMeta `json:"meta"`
}

func handleAdminConfigGet(w http.ResponseWriter, _ *http.Request) {
	response, err := buildAdminConfigResponse(nil)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func handleAdminConfigUpdate(w http.ResponseWriter, r *http.Request) {
	var incoming Config
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 2<<20))
	if err := decoder.Decode(&incoming); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid config payload"})
		return
	}

	updated, warnings, err := applyConfigUpdate(&incoming)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	InitWorkflowSessionManager()
	response, err := buildAdminConfigResponseWithConfig(updated, warnings)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func handleAdminConfigReload(w http.ResponseWriter, _ *http.Request) {
	before := getGlobalConfig()
	updated, err := reloadConfigFromDisk()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	InitWorkflowSessionManager()
	warnings := buildConfigWarnings(before, updated)
	response, err := buildAdminConfigResponseWithConfig(updated, warnings)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func buildAdminConfigResponse(warnings []string) (AdminConfigResponse, error) {
	cfg := getGlobalConfig()
	return buildAdminConfigResponseWithConfig(cfg, warnings)
}

func buildAdminConfigResponseWithConfig(cfg *Config, warnings []string) (AdminConfigResponse, error) {
	path := getConfigPath()
	exists := path != "" && fileExists(path)
	source := "configs.json"
	if path != "" {
		source = filepath.Base(path)
	}

	redacted, err := redactConfig(cfg)
	if err != nil {
		return AdminConfigResponse{}, err
	}

	meta := AdminConfigMeta{
		Exists:   exists,
		Source:   source,
		Warnings: warnings,
	}
	if loadedAt := getConfigLoadedAt(); !loadedAt.IsZero() {
		meta.LastLoaded = loadedAt.Format(time.RFC3339)
	}

	return AdminConfigResponse{
		Config: redacted,
		Meta:   meta,
	}, nil
}

func applyConfigUpdate(incoming *Config) (*Config, []string, error) {
	path := getConfigPath()
	if path == "" {
		return nil, nil, fmt.Errorf("config path not resolved")
	}

	before := getGlobalConfig()
	merged := mergeRedactedConfig(before, incoming)

	if merged.Profiles == nil {
		merged.Profiles = map[string]ProfileConfig{}
	}

	if err := writeConfigFile(path, merged); err != nil {
		return nil, nil, err
	}

	setGlobalConfig(merged, path, time.Now())
	return merged, buildConfigWarnings(before, merged), nil
}

func reloadConfigFromDisk() (*Config, error) {
	path := getConfigPath()
	if path == "" {
		return nil, fmt.Errorf("config path not resolved")
	}
	if !fileExists(path) {
		return nil, fmt.Errorf("config file not found: %s", path)
	}
	config, err := loadConfig(path)
	if err != nil {
		return nil, err
	}
	setGlobalConfig(config, path, time.Now())
	return config, nil
}

func writeConfigFile(path string, cfg *Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}

func mergeRedactedConfig(existing *Config, incoming *Config) *Config {
	if existing == nil || incoming == nil {
		return incoming
	}

	merged := *incoming
	if merged.AdminUI != nil && existing.AdminUI != nil {
		if merged.AdminUI.Token == redactedValue {
			merged.AdminUI.Token = existing.AdminUI.Token
		}
	}
	if merged.WorkflowSession != nil && merged.WorkflowSession.Redis != nil && existing.WorkflowSession != nil && existing.WorkflowSession.Redis != nil {
		if merged.WorkflowSession.Redis.Password == redactedValue {
			merged.WorkflowSession.Redis.Password = existing.WorkflowSession.Redis.Password
		}
	}

	if merged.Profiles != nil {
		for name, profile := range merged.Profiles {
			previous, ok := existing.Profiles[name]
			if !ok {
				continue
			}
			if profile.SystemPrompt == redactedValue {
				profile.SystemPrompt = previous.SystemPrompt
			}
			if profile.Env != nil {
				for key, value := range profile.Env {
					if value == redactedValue {
						if oldValue, ok := previous.Env[key]; ok {
							profile.Env[key] = oldValue
						} else {
							delete(profile.Env, key)
						}
					}
				}
			}
			merged.Profiles[name] = profile
		}
	}

	return &merged
}

func buildConfigWarnings(before *Config, after *Config) []string {
	var warnings []string
	if before == nil || after == nil {
		return warnings
	}

	if before.Server != nil && after.Server != nil {
		if before.Server.Port != after.Server.Port || before.Server.Host != after.Server.Host {
			warnings = append(warnings, "server 配置变更需重启生效")
		}
	}
	if before.ReleaseNotes != nil && after.ReleaseNotes != nil {
		if before.ReleaseNotes.RefreshIntervalMinutes != after.ReleaseNotes.RefreshIntervalMinutes ||
			before.ReleaseNotes.CacheTTLMinutes != after.ReleaseNotes.CacheTTLMinutes ||
			before.ReleaseNotes.StoragePath != after.ReleaseNotes.StoragePath {
			warnings = append(warnings, "release_notes 配置变更需重启生效")
		}
	}
	if before.AdminUI != nil && after.AdminUI != nil {
		if before.AdminUI.BasePath != after.AdminUI.BasePath || before.AdminUI.StaticDir != after.AdminUI.StaticDir {
			warnings = append(warnings, "admin_ui.base_path 或 static_dir 变更需重启生效")
		}
	}
	return warnings
}
