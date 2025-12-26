package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"
)

type AdminProfileSummary struct {
	Key         string `json:"key"`
	Name        string `json:"name"`
	CLI         string `json:"cli,omitempty"`
	Model       string `json:"model,omitempty"`
	ToolsCount  int    `json:"tools_count"`
	SkillsCount int    `json:"skills_count"`
	EnvCount    int    `json:"env_count"`
	IsDefault   bool   `json:"is_default"`
}

type AdminProfilesResponse struct {
	Default  string                `json:"default"`
	Profiles []AdminProfileSummary `json:"profiles"`
}

type AdminProfileEnvItem struct {
	Key    string `json:"key"`
	Value  string `json:"value"`
	Masked bool   `json:"masked"`
}

type AdminProfilePayload struct {
	Key                string                `json:"key"`
	Name               string                `json:"name"`
	CLI                string                `json:"cli,omitempty"`
	Model              string                `json:"model,omitempty"`
	AllowedTools       []string              `json:"allowed_tools,omitempty"`
	Skills             []string              `json:"skills,omitempty"`
	SystemPrompt       string                `json:"system_prompt,omitempty"`
	SystemPromptMasked bool                  `json:"system_prompt_masked,omitempty"`
	Env                []AdminProfileEnvItem `json:"env,omitempty"`
	IsDefault          bool                  `json:"is_default"`
}

func handleAdminProfiles(w http.ResponseWriter, r *http.Request, relativePath string) {
	base := "/api/config/profiles"
	if relativePath == base {
		switch r.Method {
		case http.MethodGet:
			handleAdminProfilesList(w)
		case http.MethodPost:
			handleAdminProfilesCreate(w, r)
		default:
			writeMethodNotAllowed(w)
		}
		return
	}

	if strings.HasPrefix(relativePath, base+"/") {
		key := strings.TrimPrefix(relativePath, base+"/")
		if key == "" || strings.Contains(key, "/") {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
			return
		}
		switch r.Method {
		case http.MethodGet:
			handleAdminProfilesGet(w, key)
		case http.MethodPut:
			handleAdminProfilesUpdate(w, r, key)
		case http.MethodDelete:
			handleAdminProfilesDelete(w, key)
		default:
			writeMethodNotAllowed(w)
		}
		return
	}

	writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
}

func handleAdminProfilesList(w http.ResponseWriter) {
	cfg := getGlobalConfig()
	response := AdminProfilesResponse{
		Default:  "",
		Profiles: []AdminProfileSummary{},
	}
	if cfg == nil {
		writeJSON(w, http.StatusOK, response)
		return
	}

	response.Default = cfg.Default
	keys := make([]string, 0, len(cfg.Profiles))
	for key := range cfg.Profiles {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		profile := cfg.Profiles[key]
		envCount := 0
		if profile.Env != nil {
			envCount = len(profile.Env)
		}
		summary := AdminProfileSummary{
			Key:         key,
			Name:        profile.Name,
			CLI:         profile.CLI,
			Model:       profile.Model,
			ToolsCount:  len(profile.AllowedTools),
			SkillsCount: len(profile.Skills),
			EnvCount:    envCount,
			IsDefault:   key == cfg.Default,
		}
		response.Profiles = append(response.Profiles, summary)
	}

	writeJSON(w, http.StatusOK, response)
}

func handleAdminProfilesGet(w http.ResponseWriter, key string) {
	cfg := getGlobalConfig()
	if cfg == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "profile not found"})
		return
	}
	profile, ok := cfg.Profiles[key]
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "profile not found"})
		return
	}

	payload := buildAdminProfilePayload(key, profile, cfg.Default)
	writeJSON(w, http.StatusOK, payload)
}

func handleAdminProfilesCreate(w http.ResponseWriter, r *http.Request) {
	var payload AdminProfilePayload
	if err := decodeAdminProfilePayload(w, r, &payload); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if payload.Key == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "profile key is required"})
		return
	}

	updated, err := updateProfilesConfig(func(cfg *Config) error {
		if _, exists := cfg.Profiles[payload.Key]; exists {
			return fmt.Errorf("profile already exists")
		}
		cfg.Profiles[payload.Key] = buildProfileFromPayload(payload, ProfileConfig{})
		if payload.IsDefault {
			cfg.Default = payload.Key
		}
		return nil
	})
	if err != nil {
		status := http.StatusInternalServerError
		if err.Error() == "profile already exists" {
			status = http.StatusConflict
		}
		writeJSON(w, status, map[string]string{"error": err.Error()})
		return
	}

	profile := updated.Profiles[payload.Key]
	writeJSON(w, http.StatusOK, buildAdminProfilePayload(payload.Key, profile, updated.Default))
}

func handleAdminProfilesUpdate(w http.ResponseWriter, r *http.Request, key string) {
	var payload AdminProfilePayload
	if err := decodeAdminProfilePayload(w, r, &payload); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	updated, err := updateProfilesConfig(func(cfg *Config) error {
		existing, ok := cfg.Profiles[key]
		if !ok {
			return fmt.Errorf("profile not found")
		}
		payload.Key = key
		cfg.Profiles[key] = buildProfileFromPayload(payload, existing)
		if payload.IsDefault {
			cfg.Default = key
		} else if cfg.Default == key {
			cfg.Default = ""
		}
		return nil
	})
	if err != nil {
		status := http.StatusInternalServerError
		if err.Error() == "profile not found" {
			status = http.StatusNotFound
		}
		writeJSON(w, status, map[string]string{"error": err.Error()})
		return
	}

	profile := updated.Profiles[key]
	writeJSON(w, http.StatusOK, buildAdminProfilePayload(key, profile, updated.Default))
}

func handleAdminProfilesDelete(w http.ResponseWriter, key string) {
	updated, err := updateProfilesConfig(func(cfg *Config) error {
		if _, ok := cfg.Profiles[key]; !ok {
			return fmt.Errorf("profile not found")
		}
		delete(cfg.Profiles, key)
		if cfg.Default == key {
			cfg.Default = pickFirstProfileKey(cfg.Profiles)
		}
		return nil
	})
	if err != nil {
		status := http.StatusInternalServerError
		if err.Error() == "profile not found" {
			status = http.StatusNotFound
		}
		writeJSON(w, status, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted", "default": updated.Default})
}

func decodeAdminProfilePayload(w http.ResponseWriter, r *http.Request, payload *AdminProfilePayload) error {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 2<<20))
	if err := decoder.Decode(payload); err != nil {
		return fmt.Errorf("invalid profile payload")
	}
	return nil
}

func updateProfilesConfig(mutate func(cfg *Config) error) (*Config, error) {
	cfg := getGlobalConfig()
	if cfg == nil {
		cfg = &Config{
			Profiles: map[string]ProfileConfig{},
		}
	}
	clone, err := cloneConfig(cfg)
	if err != nil {
		return nil, err
	}
	if clone == nil {
		clone = &Config{
			Profiles: map[string]ProfileConfig{},
		}
	}
	if clone.Profiles == nil {
		clone.Profiles = map[string]ProfileConfig{}
	}
	if err := mutate(clone); err != nil {
		return nil, err
	}

	path := getConfigPath()
	if path == "" {
		return nil, fmt.Errorf("config path not resolved")
	}
	if err := writeConfigFile(path, clone); err != nil {
		return nil, err
	}
	setGlobalConfig(clone, path, time.Now())
	return clone, nil
}

func buildAdminProfilePayload(key string, profile ProfileConfig, defaultKey string) AdminProfilePayload {
	envItems := make([]AdminProfileEnvItem, 0)
	keys := make([]string, 0, len(profile.Env))
	for envKey := range profile.Env {
		keys = append(keys, envKey)
	}
	sort.Strings(keys)
	for _, envKey := range keys {
		value := profile.Env[envKey]
		masked := isSensitiveEnvKey(envKey)
		if masked {
			value = maskValue(value)
		}
		envItems = append(envItems, AdminProfileEnvItem{
			Key:    envKey,
			Value:  value,
			Masked: masked,
		})
	}

	return AdminProfilePayload{
		Key:                key,
		Name:               profile.Name,
		CLI:                profile.CLI,
		Model:              profile.Model,
		AllowedTools:       profile.AllowedTools,
		Skills:             profile.Skills,
		SystemPrompt:       profile.SystemPrompt,
		SystemPromptMasked: false,
		Env:                envItems,
		IsDefault:          key == defaultKey,
	}
}

func buildProfileFromPayload(payload AdminProfilePayload, existing ProfileConfig) ProfileConfig {
	updated := ProfileConfig{
		Name:         payload.Name,
		CLI:          payload.CLI,
		Model:        payload.Model,
		AllowedTools: payload.AllowedTools,
		Skills:       payload.Skills,
		Env:          map[string]string{},
	}

	updated.SystemPrompt = payload.SystemPrompt

	for _, item := range payload.Env {
		if item.Key == "" {
			continue
		}
		if item.Masked {
			if existingValue, ok := existing.Env[item.Key]; ok {
				updated.Env[item.Key] = existingValue
			}
			continue
		}
		updated.Env[item.Key] = item.Value
	}

	return updated
}

func pickFirstProfileKey(profiles map[string]ProfileConfig) string {
	if len(profiles) == 0 {
		return ""
	}
	keys := make([]string, 0, len(profiles))
	for key := range profiles {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys[0]
}

func maskValue(value string) string {
	if value == "" {
		return ""
	}
	length := len(value)
	if length <= 4 {
		return strings.Repeat("*", length)
	}
	head := value[:2]
	tail := value[length-2:]
	return head + strings.Repeat("*", length-4) + tail
}
