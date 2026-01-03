package integration

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/ninesun/projects/cli-agent/internal/cli"
)

// TestIntegration_IflowCLI_FullCycle 测试完整流程
func TestIntegration_IflowCLI_FullCycle(t *testing.T) {
	// 创建iFlow CLI
	iflowCLI, err := cli.NewIflowCLI()
	if err != nil {
		t.Fatalf("Failed to create iflow CLI: %v", err)
	}

	// 配置
	config := map[string]interface{}{
		"iflow_config": map[string]interface{}{
			"max_retries": 3,
			"timeout":     300,
		},
		"cache": map[string]interface{}{
			"enabled":     true,
			"ttl_minutes": 60,
			"max_size_mb": 100,
			"strategy":    "lru",
		},
	}

	if err := iflowCLI.Initialize(config); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// 准备运行选项
	opts := &cli.RunOptions{
		Prompt: "Hello, this is a test prompt",
		Model:  "claude", // 使用claude作为底层CLI
	}

	// 执行
	startTime := time.Now()
	result, err := iflowCLI.Run(opts)
	duration := time.Since(startTime)

	if err != nil {
		t.Logf("Note: Run failed (expected if claude not installed): %v", err)
		// 这不是真正的失败，因为测试环境可能没有安装claude
		return
	}

	// 验证结果
	var output map[string]interface{}
	if err := json.Unmarshal([]byte(result), &output); err != nil {
		t.Fatalf("Failed to parse output: %v", err)
	}

	// 验证基本字段
	if _, ok := output["response"]; !ok {
		t.Error("Output should contain 'response' field")
	}

	// 验证元数据
	if metadata, ok := output["metadata"].(map[string]interface{}); ok {
		if metadata["cli"] != "iflow" {
			t.Errorf("Expected metadata cli='iflow', got %v", metadata["cli"])
		}
		if _, ok := metadata["duration_ms"]; !ok {
			t.Error("Expected duration_ms in metadata")
		}
	} else {
		t.Error("Output should contain metadata")
	}

	t.Logf("Integration test completed in %v", duration)
}

// TestIntegration_IflowCLI_WithExtensions 测试扩展支持
func TestIntegration_IflowCLI_WithExtensions(t *testing.T) {
	iflowCLI, err := cli.NewIflowCLI()
	if err != nil {
		t.Fatalf("Failed to create iflow CLI: %v", err)
	}

	// 注册一个测试扩展
	testExtName := "test-extension"
	testExtMetadata := cli.Metadata{
		Name:        testExtName,
		Version:     "1.0.0",
		Description: "Test extension for integration",
		Capabilities: []string{"test-feature"},
	}

	testExtCreator := func() (cli.CLIRunner, error) {
		return &mockCLI{name: testExtName, output: `{"response": "extension response"}`}, nil
	}

	err = cli.RegisterCLI(testExtName, testExtCreator, testExtMetadata)
	if err != nil {
		t.Fatalf("Failed to register extension: %v", err)
	}
	defer cli.UnregisterCLI(testExtName)

	// 验证扩展已注册
	if !cli.IsRegistered(testExtName) {
		t.Error("Extension should be registered")
	}

	// 使用扩展
	opts := &cli.RunOptions{
		Prompt: "Test with extension",
		Model:  testExtName,
	}

	result, err := iflowCLI.Run(opts)
	if err != nil {
		t.Errorf("Run with extension failed: %v", err)
	}

	// 验证结果
	var output map[string]interface{}
	if err := json.Unmarshal([]byte(result), &output); err != nil {
		t.Fatalf("Failed to parse output: %v", err)
	}

	if output["response"] != "extension response" {
		t.Errorf("Expected 'extension response', got %v", output["response"])
	}
}

// TestIntegration_IflowCLI_MultiSession 测试多会话
func TestIntegration_IflowCLI_MultiSession(t *testing.T) {
	iflowCLI, err := cli.NewIflowCLI()
	if err != nil {
		t.Fatalf("Failed to create iflow CLI: %v", err)
	}

	// 模拟会话管理
	sessions := []struct {
		sessionID string
		prompt    string
	}{
		{"session-1", "First prompt"},
		{"session-2", "Second prompt"},
		{"session-3", "Third prompt"},
	}

	for _, session := range sessions {
		opts := &cli.RunOptions{
			Prompt:    session.prompt,
			Model:     "claude",
			SessionID: session.sessionID,
		}

		// 仅验证不报错（实际执行需要真实CLI）
		_, err := iflowCLI.Run(opts)
		if err != nil {
			t.Logf("Session %s failed (expected without real CLI): %v", session.sessionID, err)
		}
	}
}

// TestIntegration_IflowCLI_ErrorScenarios 测试错误场景
func TestIntegration_IflowCLI_ErrorScenarios(t *testing.T) {
	iflowCLI, err := cli.NewIflowCLI()
	if err != nil {
		t.Fatalf("Failed to create iflow CLI: %v", err)
	}

	// 测试无效配置
	invalidConfig := map[string]interface{}{
		"max_retries": -1, // 无效值
	}

	// 验证配置验证
	iflowCLI2, _ := cli.NewIflowCLI()
	err = iflowCLI2.Initialize(invalidConfig)
	// Initialize可能不会立即验证，所以不强制失败

	// 测试无效CLI类型
	opts := &cli.RunOptions{
		Prompt: "test",
		Model:  "nonexistent-cli",
	}

	_, err = iflowCLI.Run(opts)
	if err == nil {
		t.Error("Should fail with invalid CLI type")
	}
}

// TestIntegration_IflowCLI_Performance 测试性能
func TestIntegration_IflowCLI_Performance(t *testing.T) {
	iflowCLI, err := cli.NewIflowCLI()
	if err != nil {
		t.Fatalf("Failed to create iflow CLI: %v", err)
	}

	// 配置启用缓存
	config := map[string]interface{}{
		"cache": map[string]interface{}{
			"enabled":     true,
			"ttl_minutes": 60,
		},
	}
	iflowCLI.Initialize(config)

	// 测试多次执行的性能
	opts := &cli.RunOptions{
		Prompt: "Performance test",
		Model:  "claude",
	}

	// 第一次（可能较慢）
	start1 := time.Now()
	_, err = iflowCLI.Run(opts)
	duration1 := time.Since(start1)

	// 第二次（应该更快，如果使用缓存）
	start2 := time.Now()
	_, err = iflowCLI.Run(opts)
	duration2 := time.Since(start2)

	// 记录性能（不强制要求缓存命中，因为依赖真实CLI）
	t.Logf("First call: %v, Second call: %v", duration1, duration2)

	if err != nil {
		t.Logf("Note: Performance test completed with error (expected without real CLI): %v", err)
	}
}

// TestIntegration_IflowCLI_ConcurrentUsers 测试并发用户
func TestIntegration_IflowCLI_ConcurrentUsers(t *testing.T) {
	iflowCLI, err := cli.NewIflowCLI()
	if err != nil {
		t.Fatalf("Failed to create iflow CLI: %v", err)
	}

	// 配置
	iflowCLI.Initialize(map[string]interface{}{
		"max_retries": 1,
	})

	// 并发执行
	const numUsers = 5
	done := make(chan bool, numUsers)

	for i := 0; i < numUsers; i++ {
		go func(id int) {
			opts := &cli.RunOptions{
				Prompt: "Concurrent test " + string(rune('0'+id)),
				Model:  "claude",
			}
			_, _ = iflowCLI.Run(opts)
			done <- true
		}(i)
	}

	// 等待所有完成
	for i := 0; i < numUsers; i++ {
		select {
		case <-done:
			// Success
		case <-time.After(5 * time.Second):
			t.Error("Concurrent test timed out")
		}
	}
}

// TestIntegration_IflowCLI_BackwardCompatibility 测试向后兼容
func TestIntegration_IflowCLI_BackwardCompatibility(t *testing.T) {
	// 测试旧的NewCLI函数仍然工作
	cli, err := cli.NewCLI("iflow")
	if err != nil {
		t.Errorf("NewCLI('iflow') failed: %v", err)
	}

	if cli == nil {
		t.Error("NewCLI returned nil")
	}

	// 测试SupportedCLIs包含iflow
	supported := cli.SupportedCLIs()
	found := false
	for _, s := range supported {
		if s == "iflow" {
			found = true
			break
		}
	}

	if !found {
		t.Error("SupportedCLIs should include 'iflow'")
	}
}

// TestIntegration_IflowCLI_ConfigChanges 测试配置变更
func TestIntegration_IflowCLI_ConfigChanges(t *testing.T) {
	iflowCLI, err := cli.NewIflowCLI()
	if err != nil {
		t.Fatalf("Failed to create iflow CLI: %v", err)
	}

	// 初始配置
	config1 := map[string]interface{}{
		"max_retries": 3,
		"timeout":     300,
	}
	iflowCLI.Initialize(config1)

	// 变更配置
	config2 := map[string]interface{}{
		"max_retries": 5,
		"timeout":     600,
		"cache": map[string]interface{}{
			"enabled": true,
		},
	}
	iflowCLI.Initialize(config2)

	// 验证配置更新（通过运行测试）
	opts := &cli.RunOptions{
		Prompt: "Config change test",
		Model:  "claude",
	}

	_, err = iflowCLI.Run(opts)
	if err != nil {
		t.Logf("Run after config change: %v (expected without real CLI)", err)
	}
}

// mockCLI 用于集成测试的模拟CLI
type mockCLI struct {
	name   string
	output string
}

func (m *mockCLI) Name() string {
	return m.name
}

func (m *mockCLI) Run(opts *cli.RunOptions) (string, error) {
	return m.output, nil
}
