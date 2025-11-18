package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
)

// ProfileConfig 表示单个配置 profile
type ProfileConfig struct {
	Name string            `json:"name"`
	CLI  string            `json:"cli,omitempty"` // 可选：指定使用的 CLI 工具（"claude" 或 "codex"）
	Env  map[string]string `json:"env"`
}

// Config 表示整个配置文件
type Config struct {
	Profiles map[string]ProfileConfig `json:"profiles"`
	Default  string                   `json:"default"`
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

// initConfig 初始化全局配置
func initConfig() error {
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

	return nil
}
