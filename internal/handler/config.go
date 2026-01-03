package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ProfileConfig 表示单个配置 profile
type ProfileConfig struct {
	Name         string            `json:"name"`
	CLI          string            `json:"cli,omitempty"`           // 可选：指定使用的 CLI 工具（"claude", "codex", "cursor"）
	Model        string            `json:"model,omitempty"`         // 可选：指定模型名称
	AllowedTools []string          `json:"allowed_tools,omitempty"` // 可选：允许的 MCP 工具列表（仅 Claude CLI）
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

// AdminUIConfig 表示后台管理 UI 配置
type AdminUIConfig struct {
	Enabled            bool   `json:"enabled"`               // 是否启用后台 UI
	Token              string `json:"token"`                 // 访问 Token
	BasePath           string `json:"base_path"`             // 路由前缀，默认 "/v1/admin"
	StaticDir          string `json:"static_dir,omitempty"`  // 本地静态目录（可选）
	CacheMaxAgeSeconds int    `json:"cache_max_age_seconds"` // 静态资源缓存秒数，默认 3600
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
	AdminUI         *AdminUIConfig           `json:"admin_ui,omitempty"`
}

const redactedValue = "__REDACTED__"

var (
	globalConfig         *Config
	globalConfigPath     string
	globalConfigLoadedAt time.Time
	globalConfigMu       sync.RWMutex
	dotenvOnce           sync.Once
)

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
	return InitConfigWithPath("")
}

// InitConfigWithPath 支持显式指定 configs.json 路径
func InitConfigWithPath(explicitPath string) error {
	configPath, strict := resolveConfigPath(explicitPath)
	return loadConfigToGlobal(configPath, strict)
}

func resolveConfigPath(explicitPath string) (string, bool) {
	if explicitPath != "" {
		return explicitPath, true
	}
	if envPath := os.Getenv("CONFIG_PATH"); envPath != "" {
		return envPath, true
	}
	if fileExists("configs.json") {
		return "configs.json", false
	}
	candidate := filepath.Join("configs", "configs.json")
	if fileExists(candidate) {
		return candidate, false
	}
	return "configs.json", false
}

func loadConfigToGlobal(configPath string, strict bool) error {
	if configPath == "" {
		configPath = "configs.json"
	}

	loadDotEnv()

	// 检查配置文件是否存在
	if !fileExists(configPath) {
		if strict {
			return fmt.Errorf("config file not found: %s", configPath)
		}
		log.Printf("⚠️  Config file not found: %s, using environment variables", configPath)
		setGlobalConfig(nil, configPath, time.Time{})
		return nil
	}

	config, err := loadConfig(configPath)
	if err != nil {
		return err
	}

	applyEnvOverrides(config)

	setGlobalConfig(config, configPath, time.Now())
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

	// 打印 admin UI 配置（不输出 token）
	if config.AdminUI != nil {
		log.Printf("📋 Admin UI config: enabled=%v, base_path=%s, static_dir=%s",
			config.AdminUI.Enabled,
			config.AdminUI.BasePath,
			config.AdminUI.StaticDir)
	}

	return nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func loadDotEnv() {
	dotenvOnce.Do(func() {
		path := ".env"
		if !fileExists(path) {
			return
		}
		data, err := os.ReadFile(path)
		if err != nil {
			log.Printf("⚠️  Failed to read .env: %v", err)
			return
		}

		lines := strings.Split(string(data), "\n")
		loaded := 0
		for _, line := range lines {
			key, value, ok := parseEnvLine(line)
			if !ok {
				continue
			}
			if _, exists := os.LookupEnv(key); exists {
				continue
			}
			if err := os.Setenv(key, value); err == nil {
				loaded++
			}
		}

		if loaded > 0 {
			log.Printf("✅ Loaded .env with %d variables", loaded)
		}
	})
}

func parseEnvLine(line string) (string, string, bool) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || strings.HasPrefix(trimmed, "#") {
		return "", "", false
	}
	if strings.HasPrefix(trimmed, "export ") {
		trimmed = strings.TrimSpace(strings.TrimPrefix(trimmed, "export "))
	}

	parts := strings.SplitN(trimmed, "=", 2)
	if len(parts) != 2 {
		return "", "", false
	}

	key := strings.TrimSpace(parts[0])
	value := strings.TrimSpace(parts[1])
	if key == "" {
		return "", "", false
	}
	if len(value) >= 2 {
		if (value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\'') {
			value = value[1 : len(value)-1]
		}
	}

	return key, value, true
}

func applyEnvOverrides(cfg *Config) {
	if cfg == nil {
		return
	}

	if cfg.AdminUI != nil {
		cfg.AdminUI.Token = resolveEnvPlaceholder(cfg.AdminUI.Token)
	}

	if cfg.WorkflowSession != nil && cfg.WorkflowSession.Redis != nil {
		cfg.WorkflowSession.Redis.Username = resolveEnvPlaceholder(cfg.WorkflowSession.Redis.Username)
		cfg.WorkflowSession.Redis.Password = resolveEnvPlaceholder(cfg.WorkflowSession.Redis.Password)
	}

	for name, profile := range cfg.Profiles {
		if profile.Env != nil {
			for key, value := range profile.Env {
				resolved := resolveEnvPlaceholder(value)
				if resolved == "" && value == "" {
					if envValue := os.Getenv(key); envValue != "" {
						resolved = envValue
					}
				}
				profile.Env[key] = resolved
			}
		}
		cfg.Profiles[name] = profile
	}
}

func resolveEnvPlaceholder(value string) string {
	trimmed := strings.TrimSpace(value)
	if strings.HasPrefix(trimmed, "${") && strings.HasSuffix(trimmed, "}") {
		key := strings.TrimSuffix(strings.TrimPrefix(trimmed, "${"), "}")
		return os.Getenv(key)
	}
	return value
}

// GetReleaseNotesConfig 返回 release notes 配置，如果未配置则返回默认值
func GetReleaseNotesConfig() ReleaseNotesConfig {
	cfgPtr := getGlobalConfig()
	if cfgPtr != nil && cfgPtr.ReleaseNotes != nil {
		cfg := *cfgPtr.ReleaseNotes
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
	cfgPtr := getGlobalConfig()
	if cfgPtr != nil && cfgPtr.Server != nil {
		cfg := *cfgPtr.Server
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

	cfgPtr := getGlobalConfig()
	if cfgPtr != nil && cfgPtr.WorkflowSession != nil {
		cfg := *cfgPtr.WorkflowSession
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

// GetAdminUIConfig 返回后台 UI 配置（含环境变量覆盖与默认值）
func GetAdminUIConfig() AdminUIConfig {
	defaultConfig := AdminUIConfig{
		Enabled:            false,
		Token:              "",
		BasePath:           "/v1/admin",
		StaticDir:          "",
		CacheMaxAgeSeconds: 3600,
	}

	cfg := defaultConfig
	cfgPtr := getGlobalConfig()
	if cfgPtr != nil && cfgPtr.AdminUI != nil {
		cfg = *cfgPtr.AdminUI
		if cfg.BasePath == "" {
			cfg.BasePath = defaultConfig.BasePath
		}
		if cfg.CacheMaxAgeSeconds <= 0 {
			cfg.CacheMaxAgeSeconds = defaultConfig.CacheMaxAgeSeconds
		}
	}

	if value := os.Getenv("ADMIN_UI_ENABLED"); value != "" {
		if parsed, err := strconv.ParseBool(value); err == nil {
			cfg.Enabled = parsed
		}
	}
	if value := os.Getenv("ADMIN_UI_TOKEN"); value != "" {
		cfg.Token = value
	}
	if value := os.Getenv("ADMIN_UI_BASE_PATH"); value != "" {
		cfg.BasePath = value
	}
	if value := os.Getenv("ADMIN_UI_STATIC_DIR"); value != "" {
		cfg.StaticDir = value
	}
	if value := os.Getenv("ADMIN_UI_CACHE_MAX_AGE"); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			cfg.CacheMaxAgeSeconds = parsed
		}
	}
	if cfg.Token != "" && cfg.BasePath == "" {
		cfg.BasePath = defaultConfig.BasePath
	}
	if cfg.Token != "" && !cfg.Enabled && os.Getenv("ADMIN_UI_ENABLED") == "" {
		cfg.Enabled = true
	}
	if cfg.Token == "" {
		cfg.Enabled = false
	}

	return cfg
}

// GetProfile 返回指定 profile 配置
func GetProfile(profileName string) (*ProfileConfig, error) {
	cfg := getGlobalConfig()
	if cfg == nil {
		return nil, fmt.Errorf("config not loaded")
	}
	return cfg.getProfile(profileName)
}

func getGlobalConfig() *Config {
	globalConfigMu.RLock()
	defer globalConfigMu.RUnlock()
	return globalConfig
}

func setGlobalConfig(cfg *Config, path string, loadedAt time.Time) {
	globalConfigMu.Lock()
	defer globalConfigMu.Unlock()
	globalConfig = cfg
	globalConfigPath = path
	globalConfigLoadedAt = loadedAt
}

func getConfigPath() string {
	globalConfigMu.RLock()
	defer globalConfigMu.RUnlock()
	return globalConfigPath
}

func getConfigLoadedAt() time.Time {
	globalConfigMu.RLock()
	defer globalConfigMu.RUnlock()
	return globalConfigLoadedAt
}

func cloneConfig(cfg *Config) (*Config, error) {
	if cfg == nil {
		return nil, nil
	}
	data, err := json.Marshal(cfg)
	if err != nil {
		return nil, err
	}
	var clone Config
	if err := json.Unmarshal(data, &clone); err != nil {
		return nil, err
	}
	return &clone, nil
}

func redactConfig(cfg *Config) (*Config, error) {
	clone, err := cloneConfig(cfg)
	if err != nil || clone == nil {
		return clone, err
	}
	if clone.AdminUI != nil && clone.AdminUI.Token != "" {
		clone.AdminUI.Token = redactedValue
	}
	if clone.WorkflowSession != nil && clone.WorkflowSession.Redis != nil && clone.WorkflowSession.Redis.Password != "" {
		clone.WorkflowSession.Redis.Password = redactedValue
	}
	for name, profile := range clone.Profiles {
		if profile.SystemPrompt != "" {
			profile.SystemPrompt = redactedValue
		}
		if profile.Env != nil {
			for key, value := range profile.Env {
				if value != "" && isSensitiveEnvKey(key) {
					profile.Env[key] = redactedValue
				}
			}
		}
		clone.Profiles[name] = profile
	}
	return clone, nil
}

func isSensitiveEnvKey(key string) bool {
	upper := strings.ToUpper(key)
	return strings.Contains(upper, "TOKEN") ||
		strings.Contains(upper, "SECRET") ||
		strings.Contains(upper, "PASSWORD") ||
		strings.Contains(upper, "API_KEY") ||
		strings.Contains(upper, "AUTH")
}
