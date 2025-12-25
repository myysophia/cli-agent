package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
)

// ProfileConfig 表示单个配置 profile
type ProfileConfig struct {
	Name         string            `json:"name"`
	CLI          string            `json:"cli,omitempty"`           // 可选：指定使用的 CLI 工具（"claude", "codex", "cursor"）
	Model        string            `json:"model,omitempty"`         // 可选：指定模型名称
	Skills       []string          `json:"skills,omitempty"`        // 可选：Claude Skills 列表（目录或文件路径）
	SystemPrompt string            `json:"system_prompt,omitempty"` // 可选：系统提示词
	Env          map[string]string `json:"env"`
}

// ServerConfig 表示服务器配置
type ServerConfig struct {
	Port int    `json:"port"` // 端口号，默认 8080
	Host string `json:"host"` // 监听地址，默认 0.0.0.0
}

// ReleaseNotesConfig 表示 release notes 服务配置
type ReleaseNotesConfig struct {
	RefreshIntervalMinutes int    `json:"refresh_interval_minutes"` // 刷新间隔（分钟），默认 60
	CacheTTLMinutes        int    `json:"cache_ttl_minutes"`        // 缓存 TTL（分钟），默认 60
	StoragePath            string `json:"storage_path"`             // 存储路径，默认 "data/release_notes.json"
}

// WorkflowSessionRedisConfig 表示 workflow 会话映射 Redis 配置
type WorkflowSessionRedisConfig struct {
	Addr           string `json:"addr"`             // Redis 地址，默认 "127.0.0.1:6379"
	Username       string `json:"username"`         // Redis 用户名（可选）
	Password       string `json:"password"`         // Redis 密码（可选）
	DB             int    `json:"db"`               // Redis DB，默认 0
	DialTimeoutMS  int    `json:"dial_timeout_ms"`  // 连接超时（毫秒），默认 5000
	ReadTimeoutMS  int    `json:"read_timeout_ms"`  // 读超时（毫秒），默认 3000
	WriteTimeoutMS int    `json:"write_timeout_ms"` // 写超时（毫秒），默认 3000
	PoolSize       int    `json:"pool_size"`        // 连接池大小，默认 10
}

// WorkflowSessionConfig 表示 workflow 会话管理配置
type WorkflowSessionConfig struct {
	MappingTTLMinutes   int                         `json:"mapping_ttl_minutes"`    // 映射 TTL（分钟），默认 1440
	LockTTLMS           int                         `json:"lock_ttl_ms"`            // 锁 TTL（毫秒），默认 120000
	LockWaitTimeoutMS   int                         `json:"lock_wait_timeout_ms"`   // 最大等待（毫秒），默认 120000
	LockRetryIntervalMS int                         `json:"lock_retry_interval_ms"` // 重试间隔（毫秒），默认 200
	Redis               *WorkflowSessionRedisConfig `json:"redis,omitempty"`
}

// Config 表示整个配置文件
type Config struct {
	Server          *ServerConfig            `json:"server,omitempty"`
	Profiles        map[string]ProfileConfig `json:"profiles"`
	Default         string                   `json:"default"`
	ReleaseNotes    *ReleaseNotesConfig      `json:"release_notes,omitempty"`
	WorkflowSession *WorkflowSessionConfig   `json:"workflow_session,omitempty"`
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
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "configs.json"
	}

	// 检查配置文件是否存在
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		if os.Getenv("CONFIG_PATH") != "" {
			return fmt.Errorf("config file not found: %s", configPath)
		}
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

	// 打印 workflow session 配置
	if config.WorkflowSession != nil {
		log.Printf("📋 Workflow session config: mapping_ttl=%dm, lock_ttl=%dms, wait_timeout=%dms, retry_interval=%dms",
			config.WorkflowSession.MappingTTLMinutes,
			config.WorkflowSession.LockTTLMS,
			config.WorkflowSession.LockWaitTimeoutMS,
			config.WorkflowSession.LockRetryIntervalMS)
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

// GetServerConfig 返回服务器配置，如果未配置则返回默认值
func GetServerConfig() ServerConfig {
	if globalConfig != nil && globalConfig.Server != nil {
		cfg := *globalConfig.Server
		// 设置默认值
		if cfg.Port <= 0 {
			cfg.Port = 8080
		}
		if cfg.Host == "" {
			cfg.Host = "0.0.0.0"
		}
		return cfg
	}
	// 返回默认配置
	return ServerConfig{
		Port: 8080,
		Host: "0.0.0.0",
	}
}

// GetWorkflowSessionConfig 返回 workflow 会话管理配置，如果未配置则返回默认值
func GetWorkflowSessionConfig() WorkflowSessionConfig {
	defaultRedis := WorkflowSessionRedisConfig{
		Addr:           "127.0.0.1:6379",
		Username:       "",
		Password:       "",
		DB:             0,
		DialTimeoutMS:  5000,
		ReadTimeoutMS:  3000,
		WriteTimeoutMS: 3000,
		PoolSize:       10,
	}
	defaultConfig := WorkflowSessionConfig{
		MappingTTLMinutes:   1440,
		LockTTLMS:           120000,
		LockWaitTimeoutMS:   120000,
		LockRetryIntervalMS: 200,
		Redis:               &defaultRedis,
	}

	if globalConfig != nil && globalConfig.WorkflowSession != nil {
		cfg := *globalConfig.WorkflowSession
		if cfg.MappingTTLMinutes <= 0 {
			cfg.MappingTTLMinutes = defaultConfig.MappingTTLMinutes
		}
		if cfg.LockTTLMS <= 0 {
			cfg.LockTTLMS = defaultConfig.LockTTLMS
		}
		if cfg.LockWaitTimeoutMS <= 0 {
			cfg.LockWaitTimeoutMS = defaultConfig.LockWaitTimeoutMS
		}
		if cfg.LockRetryIntervalMS <= 0 {
			cfg.LockRetryIntervalMS = defaultConfig.LockRetryIntervalMS
		}
		if cfg.Redis == nil {
			cfg.Redis = &defaultRedis
			return cfg
		}
		if cfg.Redis.Addr == "" {
			cfg.Redis.Addr = defaultRedis.Addr
		}
		if cfg.Redis.DialTimeoutMS <= 0 {
			cfg.Redis.DialTimeoutMS = defaultRedis.DialTimeoutMS
		}
		if cfg.Redis.ReadTimeoutMS <= 0 {
			cfg.Redis.ReadTimeoutMS = defaultRedis.ReadTimeoutMS
		}
		if cfg.Redis.WriteTimeoutMS <= 0 {
			cfg.Redis.WriteTimeoutMS = defaultRedis.WriteTimeoutMS
		}
		if cfg.Redis.PoolSize <= 0 {
			cfg.Redis.PoolSize = defaultRedis.PoolSize
		}
		return cfg
	}

	return defaultConfig
}
