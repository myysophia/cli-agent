package cli

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

// TestIflowCLI_Name 测试名称返回
func TestIflowCLI_Name(t *testing.T) {
	cli, err := NewIflowCLI()
	if err != nil {
		t.Fatalf("Failed to create iflow CLI: %v", err)
	}

	if cli.Name() != "iflow" {
		t.Errorf("Expected name 'iflow', got '%s'", cli.Name())
	}
}

// TestIflowCLI_GetVersion 测试版本信息
func TestIflowCLI_GetVersion(t *testing.T) {
	cli, err := NewIflowCLI()
	if err != nil {
		t.Fatalf("Failed to create iflow CLI: %v", err)
	}

	version := cli.GetVersion()
	if version != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got '%s'", version)
	}
}

// TestIflowCLI_GetCapabilities 测试能力列表
func TestIflowCLI_GetCapabilities(t *testing.T) {
	cli, err := NewIflowCLI()
	if err != nil {
		t.Fatalf("Failed to create iflow CLI: %v", err)
	}

	caps := cli.GetCapabilities()
	expectedCaps := []string{"session-management", "extensions", "middleware", "caching", "metrics", "retry-mechanism", "config-validation"}

	if len(caps) != len(expectedCaps) {
		t.Errorf("Expected %d capabilities, got %d", len(expectedCaps), len(caps))
	}

	for _, expected := range expectedCaps {
		found := false
		for _, cap := range caps {
			if cap == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected capability '%s' not found", expected)
		}
	}
}

// TestIflowCLI_ValidateConfig 测试配置验证
func TestIflowCLI_ValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  *IflowConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: &IflowConfig{
				MaxRetries: 3,
				Timeout:    300,
				Cache: &CacheConfig{
					Enabled:    true,
					TTLMinutes: 60,
					MaxSizeMB:  100,
					Strategy:   "lru",
				},
			},
			wantErr: false,
		},
		{
			name: "negative retries",
			config: &IflowConfig{
				MaxRetries: -1,
				Timeout:    300,
			},
			wantErr: true,
		},
		{
			name: "too many retries",
			config: &IflowConfig{
				MaxRetries: 15,
				Timeout:    300,
			},
			wantErr: true,
		},
		{
			name: "negative timeout",
			config: &IflowConfig{
				MaxRetries: 3,
				Timeout:    -1,
			},
			wantErr: true,
		},
		{
			name: "negative cache TTL",
			config: &IflowConfig{
				Cache: &CacheConfig{
					TTLMinutes: -1,
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cli := &IflowCLI{config: tt.config}
			err := cli.ValidateConfig()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestIflowCLI_Initialize 测试初始化
func TestIflowCLI_Initialize(t *testing.T) {
	cli, err := NewIflowCLI()
	if err != nil {
		t.Fatalf("Failed to create iflow CLI: %v", err)
	}

	config := map[string]interface{}{
		"iflow_config": map[string]interface{}{
			"max_retries": 5,
			"timeout":     600,
		},
		"cache": map[string]interface{}{
			"enabled":     true,
			"ttl_minutes": 120,
			"max_size_mb": 200,
			"strategy":    "fifo",
		},
		"middleware": []interface{}{
			map[string]interface{}{
				"name":    "logging",
				"enabled": true,
			},
		},
	}

	err = cli.Initialize(config)
	if err != nil {
		t.Errorf("Initialize() failed: %v", err)
	}

	if cli.config.MaxRetries != 5 {
		t.Errorf("Expected max_retries=5, got %d", cli.config.MaxRetries)
	}

	if cli.cache.config.TTLMinutes != 120 {
		t.Errorf("Expected cache TTL=120, got %d", cli.cache.config.TTLMinutes)
	}
}

// TestIflowCLI_Shutdown 测试关闭
func TestIflowCLI_Shutdown(t *testing.T) {
	cli, err := NewIflowCLI()
	if err != nil {
		t.Fatalf("Failed to create iflow CLI: %v", err)
	}

	// 添加一些缓存数据
	cli.cache.Set("test-key", "test-value")

	err = cli.Shutdown()
	if err != nil {
		t.Errorf("Shutdown() failed: %v", err)
	}

	// 验证缓存已清空
	if _, found := cli.cache.Get("test-key"); found {
		t.Error("Cache should be cleared after shutdown")
	}
}

// TestIflowCLI_Run_Basic 测试基本运行（使用mock）
func TestIflowCLI_Run_Basic(t *testing.T) {
	// 创建mock CLI
	mockCLI := &mockCLIRunner{
		name:   "mock",
		output: `{"session_id": "test-123", "user": "test prompt", "response": "mock response"}`,
	}

	// 使用工厂注册临时CLI
	factory := NewDefaultFactory()
	metadata := Metadata{Name: "test-cli", Version: "1.0.0"}
	creator := func() (CLIRunner, error) {
		return mockCLI, nil
	}
	factory.RegisterCLI("test-cli", creator, metadata)
	defer factory.UnregisterCLI("test-cli")

	// 设置全局工厂
	originalFactory := defaultFactory
	defer func() { defaultFactory = originalFactory }()
	defaultFactory = factory

	cli, err := NewIflowCLI()
	if err != nil {
		t.Fatalf("Failed to create iflow CLI: %v", err)
	}

	opts := &RunOptions{
		Prompt: "test prompt",
		Model:  "test-cli",
	}

	result, err := cli.Run(opts)
	if err != nil {
		t.Errorf("Run() failed: %v", err)
	}

	// 验证结果包含响应
	var output IflowOutput
	if err := json.Unmarshal([]byte(result), &output); err != nil {
		t.Errorf("Failed to parse output: %v", err)
	}

	if output.Response != "mock response" {
		t.Errorf("Expected response 'mock response', got '%s'", output.Response)
	}

	// 验证元数据被添加
	if output.Metadata == nil {
		t.Error("Expected metadata to be added")
	} else {
		if output.Metadata["cli"] != "iflow" {
			t.Errorf("Expected metadata cli='iflow', got '%s'", output.Metadata["cli"])
		}
	}
}

// TestIflowCLI_Run_WithCache 测试缓存功能
func TestIflowCLI_Run_WithCache(t *testing.T) {
	factory := NewDefaultFactory()

	callCount := 0
	mockCLI := &mockCLIRunner{
		name:   "mock",
		output: `{"session_id": "test-123", "user": "test prompt", "response": "cached response"}`,
	}

	// 注册带调用计数的mock CLI
	metadata := Metadata{Name: "cache-test", Version: "1.0.0"}
	creator := func() (CLIRunner, error) {
		callCount++
		return mockCLI, nil
	}
	factory.RegisterCLI("cache-test", creator, metadata)

	// 设置全局工厂
	originalFactory := defaultFactory
	defer func() { defaultFactory = originalFactory }()
	defaultFactory = factory

	cli, err := NewIflowCLI()
	if err != nil {
		t.Fatalf("Failed to create iflow CLI: %v", err)
	}

	opts := &RunOptions{
		Prompt: "test prompt",
		Model:  "cache-test",
	}

	// 第一次调用 - 应该命中CLI
	result1, err := cli.Run(opts)
	if err != nil {
		t.Errorf("First Run() failed: %v", err)
	}
	if callCount != 1 {
		t.Errorf("Expected 1 CLI call, got %d", callCount)
	}

	// 第二次调用 - 应该命中缓存
	result2, err := cli.Run(opts)
	if err != nil {
		t.Errorf("Second Run() failed: %v", err)
	}
	if callCount != 1 {
		t.Errorf("Expected CLI to be called only once (cached), got %d", callCount)
	}

	// 验证两次结果都包含相同的响应内容
	var output1, output2 IflowOutput
	if err := json.Unmarshal([]byte(result1), &output1); err != nil {
		t.Errorf("Failed to parse result1: %v", err)
	}
	if err := json.Unmarshal([]byte(result2), &output2); err != nil {
		t.Errorf("Failed to parse result2: %v", err)
	}

	if output1.Response != output2.Response {
		t.Errorf("Cached response should match original: got '%s' vs '%s'", output1.Response, output2.Response)
	}

	// 验证都包含元数据
	if output1.Metadata == nil || output2.Metadata == nil {
		t.Error("Both results should have metadata")
	}
}

// retryMockCLI 用于重试测试的特殊mock
type retryMockCLI struct {
	name      string
	callCount int
	failUntil int // 失败直到第几次调用
}

func (r *retryMockCLI) Name() string {
	return r.name
}

func (r *retryMockCLI) Run(opts *RunOptions) (string, error) {
	r.callCount++
	if r.callCount < r.failUntil {
		return "", errors.New("temporary error")
	}
	return `{"session_id": "test-123", "user": "test prompt", "response": "success"}`, nil
}

// TestIflowCLI_Run_WithRetry 测试重试机制
func TestIflowCLI_Run_WithRetry(t *testing.T) {
	factory := NewDefaultFactory()

	// 创建特殊的mock用于重试测试
	mock := &retryMockCLI{
		name:      "mock",
		failUntil: 3, // 前2次失败，第3次成功
	}

	// 注册mock CLI
	metadata := Metadata{Name: "retry-test", Version: "1.0.0"}
	creator := func() (CLIRunner, error) {
		return mock, nil
	}
	factory.RegisterCLI("retry-test", creator, metadata)

	// 设置全局工厂
	originalFactory := defaultFactory
	defer func() { defaultFactory = originalFactory }()
	defaultFactory = factory

	cli, err := NewIflowCLI()
	if err != nil {
		t.Fatalf("Failed to create iflow CLI: %v", err)
	}

	// 设置重试次数
	cli.config.MaxRetries = 3

	opts := &RunOptions{
		Prompt: "test prompt",
		Model:  "retry-test",
	}

	result, err := cli.Run(opts)
	if err != nil {
		t.Errorf("Run() with retries failed: %v", err)
	}

	if mock.callCount != 3 {
		t.Errorf("Expected 3 calls (2 failures + 1 success), got %d", mock.callCount)
	}

	var output IflowOutput
	if err := json.Unmarshal([]byte(result), &output); err != nil {
		t.Errorf("Failed to parse output: %v", err)
	}

	if output.Response != "success" {
		t.Errorf("Expected final response 'success', got '%s'", output.Response)
	}
}

// TestIflowCLI_Run_ErrorHandling 测试错误处理
func TestIflowCLI_Run_ErrorHandling(t *testing.T) {
	factory := NewDefaultFactory()

	// 注册总是失败的mock CLI
	metadata := Metadata{Name: "error-test", Version: "1.0.0"}
	creator := func() (CLIRunner, error) {
		return &mockCLIRunner{
			name:   "mock",
			output: "",
			err:    errors.New("execution failed"),
		}, nil
	}
	factory.RegisterCLI("error-test", creator, metadata)

	// 设置全局工厂
	originalFactory := defaultFactory
	defer func() { defaultFactory = originalFactory }()
	defaultFactory = factory

	cli, err := NewIflowCLI()
	if err != nil {
		t.Fatalf("Failed to create iflow CLI: %v", err)
	}

	cli.config.MaxRetries = 1 // 不重试

	opts := &RunOptions{
		Prompt: "test prompt",
		Model:  "error-test",
	}

	_, err = cli.Run(opts)
	if err == nil {
		t.Error("Expected error, got nil")
	}

	if !strings.Contains(err.Error(), "execution failed") {
		t.Errorf("Expected error message to contain 'execution failed', got: %v", err)
	}
}

// TestIflowCLI_MiddlewareChain 测试中间件链
func TestIflowCLI_MiddlewareChain(t *testing.T) {
	chain := NewMiddlewareChain()

	// 添加测试中间件
	chain.Add(&testMiddleware1{})
	chain.Add(&testMiddleware2{})

	opts := &RunOptions{
		Prompt: "test",
		Model:  "claude",
	}

	// 测试Before
	processedOpts, err := chain.ApplyBefore(opts)
	if err != nil {
		t.Errorf("ApplyBefore failed: %v", err)
	}

	if processedOpts.Prompt != "test-modified1-modified2" {
		t.Errorf("Expected prompt 'test-modified1-modified2', got '%s'", processedOpts.Prompt)
	}

	// 测试After
	result, err := chain.ApplyAfter("original", nil)
	if err != nil {
		t.Errorf("ApplyAfter failed: %v", err)
	}

	if result != "original-modified2-modified1" {
		t.Errorf("Expected result 'original-modified2-modified1', got '%s'", result)
	}
}

// TestIflowCLI_CacheOperations 测试缓存操作
func TestIflowCLI_CacheOperations(t *testing.T) {
	config := &CacheConfig{
		Enabled:    true,
		TTLMinutes: 1,
		MaxSizeMB:  10,
		Strategy:   "lru",
	}

	cache := NewResponseCache(config)

	// Set and Get
	cache.Set("key1", "value1")
	value, found := cache.Get("key1")
	if !found {
		t.Error("Expected to find cached value")
	}
	if value != "value1" {
		t.Errorf("Expected 'value1', got '%s'", value)
	}

	// Non-existent key
	_, found = cache.Get("nonexistent")
	if found {
		t.Error("Should not find non-existent key")
	}

	// Clear
	cache.Clear()
	_, found = cache.Get("key1")
	if found {
		t.Error("Cache should be empty after clear")
	}
}

// TestIflowCLI_MetricsCollector 测试指标收集
func TestIflowCLI_MetricsCollector(t *testing.T) {
	collector := NewMetricsCollector()

	// Record requests
	collector.RecordRequest("iflow", 100*time.Millisecond, false)
	collector.RecordRequest("iflow", 150*time.Millisecond, false)
	collector.RecordRequest("iflow", 50*time.Millisecond, true)

	// Record cache hits
	collector.RecordCacheHit("iflow")
	collector.RecordCacheHit("iflow")
	collector.RecordCacheMiss("iflow")

	summary := collector.GetSummary()
	if summary == nil {
		t.Fatal("Expected summary, got nil")
	}

	if _, ok := summary["iflow"]; !ok {
		t.Error("Expected 'iflow' in summary")
	}

	iflowStats := summary["iflow"].(map[string]interface{})
	if iflowStats["total_calls"] != 3 {
		t.Errorf("Expected 3 total calls, got %v", iflowStats["total_calls"])
	}

	if iflowStats["error_count"] != 1 {
		t.Errorf("Expected 1 error, got %v", iflowStats["error_count"])
	}

	if iflowStats["cache_hits"] != 2 {
		t.Errorf("Expected 2 cache hits, got %v", iflowStats["cache_hits"])
	}
}

// TestIflowCLI_ConcurrentAccess 测试并发访问
func TestIflowCLI_ConcurrentAccess(t *testing.T) {
	cli, err := NewIflowCLI()
	if err != nil {
		t.Fatalf("Failed to create iflow CLI: %v", err)
	}

	// 测试并发初始化
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			config := map[string]interface{}{
				"max_retries": 5,
			}
			err := cli.Initialize(config)
			if err != nil {
				t.Errorf("Concurrent init failed: %v", err)
			}
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

// TestIflowCLI_LargeOutput 测试大输出处理
func TestIflowCLI_LargeOutput(t *testing.T) {
	factory := NewDefaultFactory()

	// 创建大响应
	largeResponse := strings.Repeat("This is a long response. ", 1000)
	mockCLI := &mockCLIRunner{
		name:   "mock",
		output: `{"session_id": "test-123", "user": "test", "response": "` + largeResponse + `"}`,
	}

	// 注册大输出mock CLI
	metadata := Metadata{Name: "large-test", Version: "1.0.0"}
	creator := func() (CLIRunner, error) {
		return mockCLI, nil
	}
	factory.RegisterCLI("large-test", creator, metadata)

	// 设置全局工厂
	originalFactory := defaultFactory
	defer func() { defaultFactory = originalFactory }()
	defaultFactory = factory

	cli, err := NewIflowCLI()
	if err != nil {
		t.Fatalf("Failed to create iflow CLI: %v", err)
	}

	opts := &RunOptions{
		Prompt: "test",
		Model:  "large-test",
	}

	result, err := cli.Run(opts)
	if err != nil {
		t.Errorf("Run() with large output failed: %v", err)
	}

	// 验证结果
	var output IflowOutput
	if err := json.Unmarshal([]byte(result), &output); err != nil {
		t.Errorf("Failed to parse large output: %v", err)
	}

	if len(output.Response) != len(largeResponse) {
		t.Errorf("Response length mismatch: expected %d, got %d", len(largeResponse), len(output.Response))
	}
}

// TestIflowCLI_EdgeCases 测试边界情况
func TestIflowCLI_EdgeCases(t *testing.T) {
	tests := []struct {
		name string
		opts *RunOptions
	}{
		{
			name: "empty prompt",
			opts: &RunOptions{Prompt: "", Model: "edge-test"},
		},
		{
			name: "empty model",
			opts: &RunOptions{Prompt: "test", Model: ""},
		},
		{
			name: "with session",
			opts: &RunOptions{Prompt: "test", Model: "edge-test", SessionID: "session-123"},
		},
		{
			name: "with tools",
			opts: &RunOptions{Prompt: "test", Model: "edge-test", AllowedTools: []string{"fetch", "edit"}},
		},
		{
			name: "with skills",
			opts: &RunOptions{Prompt: "test", Model: "edge-test", Skills: []string{"./skill1", "./skill2"}},
		},
	}

	factory := NewDefaultFactory()
	metadata := Metadata{Name: "edge-test", Version: "1.0.0"}
	mockCLI := &mockCLIRunner{
		name:   "mock",
		output: `{"session_id": "test", "user": "test", "response": "ok"}`,
	}
	creator := func() (CLIRunner, error) {
		return mockCLI, nil
	}
	factory.RegisterCLI("edge-test", creator, metadata)
	factory.RegisterCLI("iflow-exec", creator, Metadata{Name: "iflow-exec", Version: "1.0.0"})

	originalFactory := defaultFactory
	defer func() { defaultFactory = originalFactory }()
	defaultFactory = factory

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cli, err := NewIflowCLI()
			if err != nil {
				t.Fatalf("Failed to create iflow CLI: %v", err)
			}

			_, err = cli.Run(tt.opts)
			if err != nil {
				t.Errorf("Run() failed: %v", err)
			}
		})
	}
}

// Mock CLI Runner for testing
type mockCLIRunner struct {
	name   string
	output string
	err    error
}

func (m *mockCLIRunner) Name() string {
	return m.name
}

func (m *mockCLIRunner) Run(opts *RunOptions) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return m.output, nil
}

// Test Middleware implementations
type testMiddleware1 struct{}

func (m *testMiddleware1) Name() string {
	return "testMiddleware1"
}

func (m *testMiddleware1) Before(opts *RunOptions) (*RunOptions, error) {
	opts.Prompt = opts.Prompt + "-modified1"
	return opts, nil
}

func (m *testMiddleware1) After(result string, err error) (string, error) {
	return result + "-modified1", err
}

type testMiddleware2 struct{}

func (m *testMiddleware2) Name() string {
	return "testMiddleware2"
}

func (m *testMiddleware2) Before(opts *RunOptions) (*RunOptions, error) {
	opts.Prompt = opts.Prompt + "-modified2"
	return opts, nil
}

func (m *testMiddleware2) After(result string, err error) (string, error) {
	return result + "-modified2", err
}
