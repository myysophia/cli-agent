package cli

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"
)

// IflowCLI 实现 iFlow CLI - 支持扩展、中间件、缓存和监控
type IflowCLI struct {
	middlewareChain *MiddlewareChain
	cache           *ResponseCache
	metrics         *MetricsCollector
	config          *IflowConfig
	mu              sync.RWMutex
}

// IflowConfig iFlow CLI 配置
type IflowConfig struct {
	Extensions    map[string]ExtensionConfig `json:"extensions,omitempty"`
	Cache         *CacheConfig               `json:"cache,omitempty"`
	Middleware    []MiddlewareConfig         `json:"middleware,omitempty"`
	MaxRetries    int                        `json:"max_retries,omitempty"`
	Timeout       int                        `json:"timeout,omitempty"`
	EnableMetrics bool                       `json:"enable_metrics,omitempty"`
}

// MiddlewareConfig 中间件配置
type MiddlewareConfig struct {
	Name    string            `json:"name"`
	Enabled bool              `json:"enabled"`
	Config  map[string]string `json:"config,omitempty"`
}

// IflowOutput iFlow CLI 输出格式
type IflowOutput struct {
	Type         string            `json:"type,omitempty"`
	Response     string            `json:"response"`
	User         string            `json:"user,omitempty"`
	SessionID    string            `json:"session_id,omitempty"`
	TotalCostUSD float64           `json:"total_cost_usd,omitempty"`
	DurationMS   int64             `json:"duration_ms,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

// ResponseCache 响应缓存
type ResponseCache struct {
	entries map[string]*cacheEntry
	config  *CacheConfig
	mu      sync.RWMutex
}

type cacheEntry struct {
	value      string
	expiration time.Time
}

// MetricsCollector 指标收集器
type MetricsCollector struct {
	requests map[string]*requestMetrics
	mu       sync.RWMutex
}

type requestMetrics struct {
	totalCalls    int
	totalDuration time.Duration
	errorCount    int
	lastRequest   time.Time
	cacheHits     int
	cacheMisses   int
}

// MiddlewareChain 中间件链
type MiddlewareChain struct {
	middlewares []Middleware
}

// Middleware 中间件接口
type Middleware interface {
	Name() string
	Before(opts *RunOptions) (*RunOptions, error)
	After(result string, err error) (string, error)
}

// NewIflowCLI 创建 iFlow CLI 实例
func NewIflowCLI() (*IflowCLI, error) {
	cli := &IflowCLI{
		middlewareChain: NewMiddlewareChain(),
		cache:           NewResponseCache(&CacheConfig{Enabled: true, TTLMinutes: 60, MaxSizeMB: 100, Strategy: "lru"}),
		metrics:         NewMetricsCollector(),
		config:          &IflowConfig{MaxRetries: 3, Timeout: 300, EnableMetrics: true},
	}

	// 添加默认中间件
	cli.middlewareChain.Add(&LoggingMiddleware{})
	cli.middlewareChain.Add(&MetricsMiddleware{collector: cli.metrics})
	cli.middlewareChain.Add(&RetryMiddleware{maxRetries: 3})

	log.Printf("🚀 [IflowCLI] Initialized with middleware chain (%d middlewares)", len(cli.middlewareChain.middlewares))
	return cli, nil
}

func (i *IflowCLI) Name() string {
	return "iflow"
}

func (i *IflowCLI) GetVersion() string {
	return "1.0.0"
}

func (i *IflowCLI) GetCapabilities() []string {
	return []string{
		"session-management",
		"extensions",
		"middleware",
		"caching",
		"metrics",
		"retry-mechanism",
		"config-validation",
	}
}

func (i *IflowCLI) ValidateConfig() error {
	if i.config == nil {
		return fmt.Errorf("config is nil")
	}

	// 验证缓存配置
	if i.config.Cache != nil {
		if i.config.Cache.TTLMinutes < 0 {
			return fmt.Errorf("cache TTL cannot be negative")
		}
		if i.config.Cache.MaxSizeMB < 0 {
			return fmt.Errorf("cache max size cannot be negative")
		}
	}

	// 验证重试配置
	if i.config.MaxRetries < 0 {
		return fmt.Errorf("max retries cannot be negative")
	}
	if i.config.MaxRetries > 10 {
		return fmt.Errorf("max retries too high (max: 10)")
	}

	// 验证超时配置
	if i.config.Timeout < 0 {
		return fmt.Errorf("timeout cannot be negative")
	}

	return nil
}

func (i *IflowCLI) Initialize(config map[string]interface{}) error {
	i.mu.Lock()
	defer i.mu.Unlock()

	// 解析配置
	if configData, ok := config["iflow_config"]; ok {
		if configJSON, err := json.Marshal(configData); err == nil {
			json.Unmarshal(configJSON, &i.config)
		}
	}

	// 配置缓存
	if cacheConfig, ok := config["cache"].(map[string]interface{}); ok {
		i.cache = NewResponseCache(&CacheConfig{
			Enabled:    getBool(cacheConfig, "enabled", true),
			TTLMinutes: getInt(cacheConfig, "ttl_minutes", 60),
			MaxSizeMB:  getInt(cacheConfig, "max_size_mb", 100),
			Strategy:   getString(cacheConfig, "strategy", "lru"),
		})
	}

	// 配置中间件
	if middlewareList, ok := config["middleware"].([]interface{}); ok {
		for _, m := range middlewareList {
			if mConfig, ok := m.(map[string]interface{}); ok {
				name := getString(mConfig, "name", "")
				if name != "" && getBool(mConfig, "enabled", true) {
					i.addMiddlewareByName(name, mConfig)
				}
			}
		}
	}

	log.Printf("⚙️  [IflowCLI] Initialized with custom config")
	return nil
}

func (i *IflowCLI) Shutdown() error {
	i.mu.Lock()
	defer i.mu.Unlock()

	// 清理缓存
	if i.cache != nil {
		i.cache.Clear()
	}

	// 保存指标（如果需要持久化）
	if i.metrics != nil {
		log.Printf("📊 [IflowCLI] Final metrics: %+v", i.metrics.GetSummary())
	}

	log.Printf("🛑 [IflowCLI] Shutdown complete")
	return nil
}

func (i *IflowCLI) Run(opts *RunOptions) (string, error) {
	startTime := time.Now()

	// 1. 应用中间件 - Before
	processedOpts, err := i.middlewareChain.ApplyBefore(opts)
	if err != nil {
		return "", fmt.Errorf("middleware before error: %v", err)
	}

	// 2. 检查缓存
	cacheKey := i.generateCacheKey(processedOpts)
	if i.cache != nil && i.cache.config.Enabled {
		if cached, found := i.cache.Get(cacheKey); found {
			log.Printf("💾 [IflowCLI] Cache hit, response preview: %s", previewResponse(cached))
			if i.metrics != nil {
				i.metrics.RecordCacheHit(i.Name())
			}
			// 缓存中存储的是最终结果，直接返回
			return cached, nil
		}
	}

	// 3. 执行CLI命令（带重试）
	var output string
	var execErr error
	maxRetries := i.config.MaxRetries
	if maxRetries == 0 {
		maxRetries = 1
	}

	for attempt := 1; attempt <= maxRetries; attempt++ {
		output, execErr = i.executeCLI(processedOpts)
		if execErr == nil {
			break
		}

		if attempt < maxRetries {
			log.Printf("⚠️  [IflowCLI] Attempt %d failed, retrying... (%v)", attempt, execErr)
			time.Sleep(time.Duration(attempt) * 100 * time.Millisecond) // 指数退避
		}
	}

	duration := time.Since(startTime)

	// 4. 记录指标
	if i.metrics != nil {
		i.metrics.RecordRequest(i.Name(), duration, execErr != nil)
	}

	// 5. 应用中间件 - After
	finalResult, finalErr := i.middlewareChain.ApplyAfter(output, execErr)

	// 6. 添加元数据
	if finalErr == nil {
		finalResult = i.addMetadata(finalResult, duration, processedOpts)
	}

	// 7. 缓存最终响应（带元数据）
	if finalErr == nil && i.cache != nil && i.cache.config.Enabled {
		i.cache.Set(cacheKey, finalResult)
		log.Printf("💾 [IflowCLI] Cached response preview: %s", previewResponse(finalResult))
	}

	return finalResult, finalErr
}

// executeCLI 执行底层CLI命令
func (i *IflowCLI) executeCLI(opts *RunOptions) (string, error) {
	// 确定底层CLI类型（默认使用iflow-exec）
	cliType := i.resolveDelegateCLI(opts)

	// 获取底层CLI实例
	cli, err := NewCLI(cliType)
	if err != nil {
		return "", fmt.Errorf("failed to get CLI '%s': %v", cliType, err)
	}

	// 执行命令
	log.Printf("⚡ [IflowCLI] Delegating to %s CLI", cliType)
	output, err := cli.Run(opts)
	if err != nil {
		return "", err
	}

	return output, nil
}

func (i *IflowCLI) resolveDelegateCLI(opts *RunOptions) string {
	if opts == nil {
		return "iflow-exec"
	}

	model := opts.Model
	if model == "" || model == "iflow" {
		return "iflow-exec"
	}

	if isKnownCLIName(model) {
		if model == "iflow" {
			return "iflow-exec"
		}
		return model
	}

	return "iflow-exec"
}

func isKnownCLIName(name string) bool {
	if name == "" {
		return false
	}
	if IsRegistered(name) {
		return true
	}
	switch name {
	case "claude-code", "cursor-agent", "qwen-code":
		return true
	default:
		return false
	}
}

// generateCacheKey 生成缓存键
func (i *IflowCLI) generateCacheKey(opts *RunOptions) string {
	parts := []string{
		"iflow",
		opts.Prompt,
		opts.SystemPrompt,
		opts.Model,
		opts.PermissionMode,
	}

	// 添加允许的工具
	if len(opts.AllowedTools) > 0 {
		parts = append(parts, strings.Join(opts.AllowedTools, ","))
	}

	// 添加Skills
	if len(opts.Skills) > 0 {
		parts = append(parts, strings.Join(opts.Skills, ","))
	}

	// 添加SessionID（如果存在）
	if opts.SessionID != "" {
		parts = append(parts, opts.SessionID)
	}

	return fmt.Sprintf("%x", []byte(strings.Join(parts, "|")))
}

// addMetadata 添加元数据到响应
func (i *IflowCLI) addMetadata(result string, duration time.Duration, opts *RunOptions) string {
	var output IflowOutput
	if err := json.Unmarshal([]byte(result), &output); err != nil {
		// 如果不是JSON，直接返回
		return result
	}

	// 添加元数据
	if output.Metadata == nil {
		output.Metadata = make(map[string]string)
	}
	output.Metadata["cli"] = "iflow"
	output.Metadata["duration_ms"] = fmt.Sprintf("%d", duration.Milliseconds())
	output.Metadata["model"] = opts.Model
	output.Metadata["timestamp"] = time.Now().Format(time.RFC3339)

	// 如果有缓存配置，添加缓存信息
	if i.cache != nil && i.cache.config.Enabled {
		output.Metadata["cache_enabled"] = "true"
		output.Metadata["cache_strategy"] = i.cache.config.Strategy
	}

	jsonBytes, err := json.Marshal(output)
	if err != nil {
		return result
	}

	return string(jsonBytes)
}

func previewResponse(result string) string {
	if result == "" {
		return ""
	}

	var output IflowOutput
	if err := json.Unmarshal([]byte(result), &output); err == nil && output.Response != "" {
		return truncate(output.Response, 100)
	}

	return truncate(result, 100)
}

// addMiddlewareByName 根据名称添加中间件
func (i *IflowCLI) addMiddlewareByName(name string, config map[string]interface{}) {
	switch name {
	case "logging":
		i.middlewareChain.Add(&LoggingMiddleware{})
	case "metrics":
		i.middlewareChain.Add(&MetricsMiddleware{collector: i.metrics})
	case "retry":
		retries := getInt(config, "max_retries", 3)
		i.middlewareChain.Add(&RetryMiddleware{maxRetries: retries})
	case "cache":
		// 缓存已在初始化时配置
		log.Printf("ℹ️  [IflowCLI] Cache middleware already configured")
	default:
		log.Printf("⚠️  [IflowCLI] Unknown middleware: %s", name)
	}
}

// NewResponseCache 创建响应缓存
func NewResponseCache(config *CacheConfig) *ResponseCache {
	return &ResponseCache{
		entries: make(map[string]*cacheEntry),
		config:  config,
	}
}

func (c *ResponseCache) Get(key string) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.entries[key]
	if !exists {
		return "", false
	}

	if time.Now().After(entry.expiration) {
		return "", false
	}

	return entry.value, true
}

func (c *ResponseCache) Set(key string, value string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 清理过期条目
	c.cleanup()

	// 检查大小限制
	if len(c.entries) >= 1000 { // 简化：最大条目数
		// LRU策略：删除最旧的
		var oldestKey string
		var oldestTime time.Time
		for k, v := range c.entries {
			if oldestTime.IsZero() || v.expiration.Before(oldestTime) {
				oldestKey = k
				oldestTime = v.expiration
			}
		}
		delete(c.entries, oldestKey)
	}

	ttl := time.Duration(c.config.TTLMinutes) * time.Minute
	c.entries[key] = &cacheEntry{
		value:      value,
		expiration: time.Now().Add(ttl),
	}
}

func (c *ResponseCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries = make(map[string]*cacheEntry)
}

func (c *ResponseCache) cleanup() {
	now := time.Now()
	for key, entry := range c.entries {
		if now.After(entry.expiration) {
			delete(c.entries, key)
		}
	}
}

// NewMetricsCollector 创建指标收集器
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		requests: make(map[string]*requestMetrics),
	}
}

func (m *MetricsCollector) RecordRequest(cliName string, duration time.Duration, isError bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	metrics, exists := m.requests[cliName]
	if !exists {
		metrics = &requestMetrics{}
		m.requests[cliName] = metrics
	}

	metrics.totalCalls++
	metrics.totalDuration += duration
	metrics.lastRequest = time.Now()
	if isError {
		metrics.errorCount++
	}
}

func (m *MetricsCollector) RecordCacheHit(cliName string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	metrics, exists := m.requests[cliName]
	if !exists {
		metrics = &requestMetrics{}
		m.requests[cliName] = metrics
	}

	metrics.cacheHits++
}

func (m *MetricsCollector) RecordCacheMiss(cliName string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	metrics, exists := m.requests[cliName]
	if !exists {
		metrics = &requestMetrics{}
		m.requests[cliName] = metrics
	}

	metrics.cacheMisses++
}

func (m *MetricsCollector) GetSummary() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]interface{})
	for name, metrics := range m.requests {
		var avgDuration time.Duration
		if metrics.totalCalls > 0 {
			avgDuration = metrics.totalDuration / time.Duration(metrics.totalCalls)
		}

		var successRate float64
		if metrics.totalCalls > 0 {
			successRate = float64(metrics.totalCalls-metrics.errorCount) / float64(metrics.totalCalls)
		}

		result[name] = map[string]interface{}{
			"total_calls":  metrics.totalCalls,
			"avg_duration": avgDuration.String(),
			"error_count":  metrics.errorCount,
			"success_rate": successRate,
			"cache_hits":   metrics.cacheHits,
			"cache_misses": metrics.cacheMisses,
			"last_request": metrics.lastRequest.Format(time.RFC3339),
		}
	}
	return result
}

// NewMiddlewareChain 创建中间件链
func NewMiddlewareChain() *MiddlewareChain {
	return &MiddlewareChain{
		middlewares: make([]Middleware, 0),
	}
}

func (c *MiddlewareChain) Add(middleware Middleware) {
	c.middlewares = append(c.middlewares, middleware)
	log.Printf("🔗 [Middleware] Added: %s", middleware.Name())
}

func (c *MiddlewareChain) ApplyBefore(opts *RunOptions) (*RunOptions, error) {
	var err error
	processedOpts := opts

	for _, middleware := range c.middlewares {
		processedOpts, err = middleware.Before(processedOpts)
		if err != nil {
			return nil, fmt.Errorf("middleware '%s' before error: %v", middleware.Name(), err)
		}
	}

	return processedOpts, nil
}

func (c *MiddlewareChain) ApplyAfter(result string, err error) (string, error) {
	var processedResult = result
	var processedErr = err

	for i := len(c.middlewares) - 1; i >= 0; i-- {
		middleware := c.middlewares[i]
		processedResult, processedErr = middleware.After(processedResult, processedErr)
	}

	return processedResult, processedErr
}

// LoggingMiddleware 日志中间件
type LoggingMiddleware struct{}

func (m *LoggingMiddleware) Name() string {
	return "LoggingMiddleware"
}

func (m *LoggingMiddleware) Before(opts *RunOptions) (*RunOptions, error) {
	log.Printf("📝 [Middleware:Logging] Before: prompt=%s, model=%s", truncate(opts.Prompt, 50), opts.Model)
	return opts, nil
}

func (m *LoggingMiddleware) After(result string, err error) (string, error) {
	if err != nil {
		log.Printf("❌ [Middleware:Logging] After error: %v", err)
	} else {
		log.Printf("✅ [Middleware:Logging] After: result_length=%d", len(result))
	}
	return result, err
}

// MetricsMiddleware 指标中间件
type MetricsMiddleware struct {
	collector *MetricsCollector
}

func (m *MetricsMiddleware) Name() string {
	return "MetricsMiddleware"
}

func (m *MetricsMiddleware) Before(opts *RunOptions) (*RunOptions, error) {
	return opts, nil
}

func (m *MetricsMiddleware) After(result string, err error) (string, error) {
	// 指标已在主流程中记录，这里仅记录中间件处理
	return result, err
}

// RetryMiddleware 重试中间件（实际重试逻辑在主流程中）
type RetryMiddleware struct {
	maxRetries int
}

func (m *RetryMiddleware) Name() string {
	return "RetryMiddleware"
}

func (m *RetryMiddleware) Before(opts *RunOptions) (*RunOptions, error) {
	return opts, nil
}

func (m *RetryMiddleware) After(result string, err error) (string, error) {
	// 重试逻辑已在主流程中处理，这里仅记录配置
	return result, err
}

// 辅助函数
func getBool(m map[string]interface{}, key string, defaultValue bool) bool {
	if v, ok := m[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return defaultValue
}

func getInt(m map[string]interface{}, key string, defaultValue int) int {
	if v, ok := m[key]; ok {
		switch val := v.(type) {
		case int:
			return val
		case float64:
			return int(val)
		}
	}
	return defaultValue
}

func getString(m map[string]interface{}, key string, defaultValue string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return defaultValue
}
