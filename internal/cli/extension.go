package cli

import "time"

// ExtensionCLI 定义扩展CLI接口，继承CLIRunner并添加扩展方法
type ExtensionCLI interface {
	CLIRunner

	// GetVersion 返回扩展版本
	GetVersion() string

	// GetCapabilities 返回扩展能力列表
	GetCapabilities() []string

	// ValidateConfig 验证配置是否有效
	ValidateConfig() error

	// Initialize 初始化扩展（生命周期方法）
	Initialize(config map[string]interface{}) error

	// Shutdown 清理资源（生命周期方法）
	Shutdown() error
}

// ExtensionInfo 扩展元数据信息
type ExtensionInfo struct {
	Name         string            `json:"name"`
	Version      string            `json:"version"`
	Description  string            `json:"description"`
	Capabilities []string          `json:"capabilities"`
	Config       map[string]string `json:"config,omitempty"`
	Enabled      bool              `json:"enabled"`
	LastUsed     time.Time         `json:"last_used,omitempty"`
	ErrorCount   int               `json:"error_count,omitempty"`
}

// CLICreator 扩展创建函数类型
type CLICreator func() (CLIRunner, error)

// Metadata 扩展元数据（用于注册）
type Metadata struct {
	Name         string   `json:"name"`
	Version      string   `json:"version"`
	Description  string   `json:"description"`
	Author       string   `json:"author"`
	Tags         []string `json:"tags"`
	Capabilities []string `json:"capabilities"`
}

// CLIStats CLI性能统计
type CLIStats struct {
	TotalCalls  int           `json:"total_calls"`
	AvgDuration time.Duration `json:"avg_duration"`
	LastUsed    time.Time     `json:"last_used"`
	ErrorCount  int           `json:"error_count"`
	SuccessRate float64       `json:"success_rate"`
}

// CacheConfig 缓存配置
type CacheConfig struct {
	Enabled    bool   `json:"enabled"`
	TTLMinutes int    `json:"ttl_minutes"`
	MaxSizeMB  int    `json:"max_size_mb"`
	Strategy   string `json:"strategy"` // "lru", "fifo", "lfu"
}

// ExtensionConfig 扩展配置
type ExtensionConfig struct {
	Enabled  bool              `json:"enabled"`
	Config   map[string]string `json:"config"`
	Priority int               `json:"priority"`
}
