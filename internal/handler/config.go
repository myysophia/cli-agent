package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
)

// ProfileConfig 表示单个配置 profile
type ProfileConfig struct {
	Name   string            `json:"name"`
	CLI    string            `json:"cli,omitempty"`    // 可选：指定使用的 CLI 工具（"claude", "codex", "cursor"）
	Model  string            `json:"model,omitempty"`  // 可选：指定模型名称
	Skills []string          `json:"skills,omitempty"` // 可选：Claude Skills 列表（目录或文件路径）
	Env    map[string]string `json:"env"`
}

// ReleaseNotesConfig 表示 release notes 服务配置
type ReleaseNotesConfig struct {
	RefreshIntervalMinutes int    `json:"refresh_interval_minutes"` // 刷新间隔（分钟），默认 60
	CacheTTLMinutes        int    `json:"cache_ttl_minutes"`        // 缓存 TTL（分钟），默认 60
	StoragePath            string `json:"storage_path"`             // 存储路径，默认 "data/release_notes.json"
}

// Config 表示整个配置文件
type Config struct {
	Profiles     map[string]ProfileConfig `json:"profiles"`
	Default      string                   `json:"default"`
	ReleaseNotes *ReleaseNotesConfig      `json:"release_notes,omitempty"`
}

var globalConfig *Config

// loadConfig 加载配置文件
func loadConfig(configPath string) (*Config, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %v", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %v", err)
	}

	return &config, nil
}

// getProfile 获取指定的 profile，如果为空则使用默认
func (c *Config) getProfile(profileName string) (*ProfileConfig, error) {
	if profileName == "" {
		profileName = c.Default
	}

	profile, ok := c.Profiles[profileName]
	if !ok {
		return nil, fmt.Errorf("profile '%s' not found", profileName)
	}

	return &profile, nil
}

// InitConfig 初始化全局配置
func InitConfig() error {
	configPath := "configs.json"
	
	// 检查配置文件是否存在
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		log.Printf("⚠️  Config file not found: %s, using environment variables", configPath)
		return nil
	}

	config, err := loadConfig(configPath)
	if err != nil {
		return err
	}

	globalConfig = config
	log.Printf("✅ Loaded config with %d profiles, default: %s", len(config.Profiles), config.Default)
	
	// 列出所有可用的 profiles
	for name, profile := range config.Profiles {
		log.Printf("   - %s: %s", name, profile.Name)
	}
	
	// 打印 release notes 配置
	if config.ReleaseNotes != nil {
		log.Printf("📋 Release notes config: refresh=%dm, cache_ttl=%dm, storage=%s",
			config.ReleaseNotes.RefreshIntervalMinutes,
			config.ReleaseNotes.CacheTTLMinutes,
			config.ReleaseNotes.StoragePath)
	}

	return nil
}

// GetReleaseNotesConfig 返回 release notes 配置，如果未配置则返回默认值
func GetReleaseNotesConfig() ReleaseNotesConfig {
	if globalConfig != nil && globalConfig.ReleaseNotes != nil {
		cfg := *globalConfig.ReleaseNotes
		// 设置默认值
		if cfg.RefreshIntervalMinutes <= 0 {
			cfg.RefreshIntervalMinutes = 60
		}
		if cfg.CacheTTLMinutes <= 0 {
			cfg.CacheTTLMinutes = 60
		}
		if cfg.StoragePath == "" {
			cfg.StoragePath = "data/release_notes.json"
		}
		return cfg
	}
	// 返回默认配置
	return ReleaseNotesConfig{
		RefreshIntervalMinutes: 60,
		CacheTTLMinutes:        60,
		StoragePath:            "data/release_notes.json",
	}
}
