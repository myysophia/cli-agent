package cli

import (
	"fmt"
	"log"
	"sync"
	"time"
)

// CLIType 定义支持的 CLI 类型
type CLIType string

const (
	CLIClaude    CLIType = "claude"
	CLICodex     CLIType = "codex"
	CLICursor    CLIType = "cursor"
	CLIGemini    CLIType = "gemini"
	CLIQwen      CLIType = "qwen"
	CLIIFlow     CLIType = "iflow"      // 新增 iflow CLI 类型
	CLIIFlowExec CLIType = "iflow-exec" // iflow 可执行命令
)

// Factory 工厂接口，支持扩展注册
type Factory interface {
	NewCLI(cliType string) (CLIRunner, error)
	RegisterCLI(name string, creator CLICreator, metadata Metadata) error
	UnregisterCLI(name string) error
	IsRegistered(name string) bool
	GetMetadata(name string) (Metadata, error)
	ListAvailable() []string
	ListWithMetadata() map[string]Metadata
	ValidateCLIConfig(name string, config map[string]interface{}) error
	GetStats(name string) (CLIStats, error)
}

// DefaultFactory 默认工厂实现
type DefaultFactory struct {
	registry *ExtensionRegistry
	stats    map[string]*cliStatsInternal
	mu       sync.RWMutex
}

type cliStatsInternal struct {
	totalCalls    int
	totalDuration time.Duration
	lastUsed      time.Time
	errorCount    int
}

// NewDefaultFactory 创建默认工厂
func NewDefaultFactory() *DefaultFactory {
	return &DefaultFactory{
		registry: NewExtensionRegistry(),
		stats:    make(map[string]*cliStatsInternal),
	}
}

// trackExecution 记录执行统计
func (f *DefaultFactory) trackExecution(name string, duration time.Duration, err error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	stats, exists := f.stats[name]
	if !exists {
		stats = &cliStatsInternal{}
		f.stats[name] = stats
	}

	stats.totalCalls++
	stats.totalDuration += duration
	stats.lastUsed = time.Now()
	if err != nil {
		stats.errorCount++
	}
}

// GetStats 获取CLI统计信息
func (f *DefaultFactory) GetStats(name string) (CLIStats, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	stats, exists := f.stats[name]
	if !exists {
		return CLIStats{}, fmt.Errorf("no stats available for '%s'", name)
	}

	var avgDuration time.Duration
	if stats.totalCalls > 0 {
		avgDuration = stats.totalDuration / time.Duration(stats.totalCalls)
	}

	var successRate float64
	if stats.totalCalls > 0 {
		successRate = float64(stats.totalCalls-stats.errorCount) / float64(stats.totalCalls)
	}

	return CLIStats{
		TotalCalls:  stats.totalCalls,
		AvgDuration: avgDuration,
		LastUsed:    stats.lastUsed,
		ErrorCount:  stats.errorCount,
		SuccessRate: successRate,
	}, nil
}

// NewCLI 根据类型创建对应的 CLI 实例（增强版）
func NewCLI(cliType string) (CLIRunner, error) {
	return defaultFactory.NewCLI(cliType)
}

// defaultFactory 全局默认工厂实例
var defaultFactory = NewDefaultFactory()

// RegisterCLI 注册自定义CLI扩展
func RegisterCLI(name string, creator CLICreator, metadata Metadata) error {
	return defaultFactory.RegisterCLI(name, creator, metadata)
}

// UnregisterCLI 卸载CLI扩展
func UnregisterCLI(name string) error {
	return defaultFactory.UnregisterCLI(name)
}

// IsRegistered 检查CLI是否已注册
func IsRegistered(name string) bool {
	return defaultFactory.IsRegistered(name)
}

// GetMetadata 获取CLI元数据
func GetMetadata(name string) (Metadata, error) {
	return defaultFactory.GetMetadata(name)
}

// ListAvailable 返回所有支持的 CLI 类型
func ListAvailable() []string {
	return defaultFactory.ListAvailable()
}

// ListWithMetadata 返回带元数据的CLI列表
func ListWithMetadata() map[string]Metadata {
	return defaultFactory.ListWithMetadata()
}

// ValidateCLIConfig 验证CLI配置
func ValidateCLIConfig(name string, config map[string]interface{}) error {
	return defaultFactory.ValidateCLIConfig(name, config)
}

// GetCLIStats 获取CLI统计信息
func GetCLIStats(name string) (CLIStats, error) {
	return defaultFactory.GetStats(name)
}

// DefaultFactory 实现 Factory 接口

func (f *DefaultFactory) NewCLI(cliType string) (CLIRunner, error) {
	startTime := time.Now()

	// 首先检查是否是扩展注册的CLI
	if f.registry.IsRegistered(cliType) {
		instance, err := f.registry.Get(cliType)
		if err != nil {
			f.trackExecution(cliType, time.Since(startTime), err)
			return nil, err
		}
		f.trackExecution(cliType, time.Since(startTime), nil)
		return instance, nil
	}

	// 然后检查内置CLI
	var cli CLIRunner
	var err error

	switch CLIType(cliType) {
	case CLIClaude, "claude-code":
		cli = NewClaudeCLI()
	case CLICodex:
		cli = NewCodexCLI()
	case CLICursor, "cursor-agent":
		cli = NewCursorCLI()
	case CLIGemini:
		cli = NewGeminiCLI()
	case CLIQwen, "qwen-code":
		cli = NewQwenCLI()
	case CLIIFlow:
		cli, err = NewIflowCLI()
	case CLIIFlowExec:
		cli = NewIflowExecCLI()
	default:
		err = fmt.Errorf("unsupported CLI type: %s", cliType)
	}

	if err != nil {
		f.trackExecution(cliType, time.Since(startTime), err)
		return nil, err
	}

	if cli != nil {
		f.trackExecution(cliType, time.Since(startTime), nil)
	}

	return cli, nil
}

func (f *DefaultFactory) RegisterCLI(name string, creator CLICreator, metadata Metadata) error {
	return f.registry.Register(name, creator, metadata)
}

func (f *DefaultFactory) UnregisterCLI(name string) error {
	return f.registry.Unregister(name)
}

func (f *DefaultFactory) IsRegistered(name string) bool {
	// 检查扩展注册器
	if f.registry.IsRegistered(name) {
		return true
	}

	// 检查内置CLI
	switch CLIType(name) {
	case CLIClaude, CLICodex, CLICursor, CLIGemini, CLIQwen, CLIIFlow, CLIIFlowExec:
		return true
	}

	return false
}

func (f *DefaultFactory) GetMetadata(name string) (Metadata, error) {
	// 首先尝试从扩展注册器获取
	info, err := f.registry.GetInfo(name)
	if err == nil {
		return Metadata{
			Name:         info.Name,
			Version:      info.Version,
			Description:  info.Description,
			Author:       "Extension",
			Tags:         []string{"extension"},
			Capabilities: info.Capabilities,
		}, nil
	}

	// 检查是否是内置CLI，返回硬编码的元数据
	builtinMetadata := map[string]Metadata{
		"claude": {
			Name:         "claude",
			Version:      "1.0.0",
			Description:  "Claude Code CLI implementation",
			Author:       "Anthropic",
			Tags:         []string{"ai", "claude", "anthropic"},
			Capabilities: []string{"session-management", "tools", "system-prompt", "skills"},
		},
		"codex": {
			Name:         "codex",
			Version:      "1.0.0",
			Description:  "OpenAI Codex CLI implementation",
			Author:       "OpenAI",
			Tags:         []string{"ai", "codex", "openai"},
			Capabilities: []string{"session-management"},
		},
		"cursor": {
			Name:         "cursor",
			Version:      "1.0.0",
			Description:  "Cursor Agent CLI implementation",
			Author:       "Cursor",
			Tags:         []string{"ai", "cursor", "agent"},
			Capabilities: []string{"session-management", "agent-mode"},
		},
		"gemini": {
			Name:         "gemini",
			Version:      "1.0.0",
			Description:  "Google Gemini CLI implementation",
			Author:       "Google",
			Tags:         []string{"ai", "gemini", "google"},
			Capabilities: []string{"session-management"},
		},
		"qwen": {
			Name:         "qwen",
			Version:      "1.0.0",
			Description:  "Alibaba Qwen CLI implementation",
			Author:       "Alibaba",
			Tags:         []string{"ai", "qwen", "alibaba"},
			Capabilities: []string{"session-management"},
		},
		"iflow": {
			Name:         "iflow",
			Version:      "1.0.0",
			Description:  "iFlow CLI with extension support and advanced features",
			Author:       "iFlow Team",
			Tags:         []string{"ai", "iflow", "extension", "advanced"},
			Capabilities: []string{"session-management", "extensions", "middleware", "caching", "metrics", "plugins"},
		},
		"iflow-exec": {
			Name:         "iflow-exec",
			Version:      "1.0.0",
			Description:  "iFlow CLI executable wrapper",
			Author:       "iFlow Team",
			Tags:         []string{"ai", "iflow"},
			Capabilities: []string{"session-management"},
		},
	}

	if meta, ok := builtinMetadata[name]; ok {
		return meta, nil
	}

	return Metadata{}, fmt.Errorf("CLI '%s' not found", name)
}

func (f *DefaultFactory) ListAvailable() []string {
	// 内置CLI
	builtins := []string{
		string(CLIClaude),
		string(CLICodex),
		string(CLICursor),
		string(CLIGemini),
		string(CLIQwen),
		string(CLIIFlow),
		string(CLIIFlowExec),
	}

	// 添加注册的扩展
	extensions := f.registry.ListAll()
	for _, ext := range extensions {
		builtins = append(builtins, ext.Name)
	}

	return builtins
}

func (f *DefaultFactory) ListWithMetadata() map[string]Metadata {
	result := make(map[string]Metadata)

	// 内置CLI元数据
	builtinMetadata := map[string]Metadata{
		"claude": {
			Name:         "claude",
			Version:      "1.0.0",
			Description:  "Claude Code CLI implementation",
			Author:       "Anthropic",
			Tags:         []string{"ai", "claude", "anthropic"},
			Capabilities: []string{"session-management", "tools", "system-prompt", "skills"},
		},
		"codex": {
			Name:         "codex",
			Version:      "1.0.0",
			Description:  "OpenAI Codex CLI implementation",
			Author:       "OpenAI",
			Tags:         []string{"ai", "codex", "openai"},
			Capabilities: []string{"session-management"},
		},
		"cursor": {
			Name:         "cursor",
			Version:      "1.0.0",
			Description:  "Cursor Agent CLI implementation",
			Author:       "Cursor",
			Tags:         []string{"ai", "cursor", "agent"},
			Capabilities: []string{"session-management", "agent-mode"},
		},
		"gemini": {
			Name:         "gemini",
			Version:      "1.0.0",
			Description:  "Google Gemini CLI implementation",
			Author:       "Google",
			Tags:         []string{"ai", "gemini", "google"},
			Capabilities: []string{"session-management"},
		},
		"qwen": {
			Name:         "qwen",
			Version:      "1.0.0",
			Description:  "Alibaba Qwen CLI implementation",
			Author:       "Alibaba",
			Tags:         []string{"ai", "qwen", "alibaba"},
			Capabilities: []string{"session-management"},
		},
		"iflow": {
			Name:         "iflow",
			Version:      "1.0.0",
			Description:  "iFlow CLI with extension support and advanced features",
			Author:       "iFlow Team",
			Tags:         []string{"ai", "iflow", "extension", "advanced"},
			Capabilities: []string{"session-management", "extensions", "middleware", "caching", "metrics", "plugins"},
		},
		"iflow-exec": {
			Name:         "iflow-exec",
			Version:      "1.0.0",
			Description:  "iFlow CLI executable wrapper",
			Author:       "iFlow Team",
			Tags:         []string{"ai", "iflow"},
			Capabilities: []string{"session-management"},
		},
	}

	for name, meta := range builtinMetadata {
		result[name] = meta
	}

	// 添加扩展元数据
	extensions := f.registry.ListAll()
	for _, ext := range extensions {
		result[ext.Name] = Metadata{
			Name:         ext.Name,
			Version:      ext.Version,
			Description:  ext.Description,
			Author:       "Extension",
			Tags:         []string{"extension"},
			Capabilities: ext.Capabilities,
		}
	}

	return result
}

func (f *DefaultFactory) ValidateCLIConfig(name string, config map[string]interface{}) error {
	// 检查CLI是否存在
	if !f.IsRegistered(name) {
		// 检查是否是内置CLI
		found := false
		for _, builtin := range []string{"claude", "codex", "cursor", "gemini", "qwen", "iflow", "iflow-exec"} {
			if name == builtin {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("CLI '%s' is not registered", name)
		}
	}

	// 如果是扩展CLI，验证其配置
	if f.registry.IsRegistered(name) {
		instance, err := f.registry.Get(name)
		if err != nil {
			return err
		}

		if extCLI, ok := instance.(ExtensionCLI); ok {
			return extCLI.ValidateConfig()
		}
	}

	// 基础配置验证
	if config != nil {
		// 检查必需字段（根据CLI类型）
		if env, ok := config["env"].(map[string]interface{}); ok {
			// 验证环境变量
			for key, value := range env {
				if key == "" {
					return fmt.Errorf("empty environment variable key")
				}
				if value == nil {
					return fmt.Errorf("nil value for environment variable '%s'", key)
				}
			}
		}
	}

	return nil
}

// 兼容性函数 - 保持与旧代码的兼容

// SupportedCLIs 返回所有支持的 CLI 类型（向后兼容）
func SupportedCLIs() []string {
	return ListAvailable()
}

// NewCLIWithOptions 创建CLI并记录统计（增强版）
func NewCLIWithOptions(cliType string, opts *RunOptions) (CLIRunner, error) {
	startTime := time.Now()
	cli, err := NewCLI(cliType)
	if err != nil {
		defaultFactory.trackExecution(cliType, time.Since(startTime), err)
		return nil, err
	}

	// 如果是扩展CLI，初始化配置
	if extCLI, ok := cli.(ExtensionCLI); ok {
		if opts != nil && opts.Env != nil {
			config := make(map[string]interface{})
			for k, v := range opts.Env {
				config[k] = v
			}
			if err := extCLI.Initialize(config); err != nil {
				defaultFactory.trackExecution(cliType, time.Since(startTime), err)
				return nil, fmt.Errorf("failed to initialize extension '%s': %v", cliType, err)
			}
		}
	}

	defaultFactory.trackExecution(cliType, time.Since(startTime), nil)
	return cli, nil
}

// GetFactory 获取全局工厂实例
func GetFactory() Factory {
	return defaultFactory
}

// SetFactory 设置全局工厂实例（用于测试）
func SetFactory(factory Factory) {
	if df, ok := factory.(*DefaultFactory); ok {
		defaultFactory = df
	}
}

// 初始化时注册内置CLI的元数据（用于一致性）
func init() {
	// 预注册内置CLI信息，确保ListWithMetadata返回完整信息
	log.Printf("📦 [Factory] Initialized with %d built-in CLI types", len([]CLIType{CLIClaude, CLICodex, CLICursor, CLIGemini, CLIQwen, CLIIFlow, CLIIFlowExec}))
}
