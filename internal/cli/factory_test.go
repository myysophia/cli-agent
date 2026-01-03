package cli

import (
	"testing"
)

// TestFactory_NewCLI_Builtins 测试内置CLI创建
func TestFactory_NewCLI_Builtins(t *testing.T) {
	factory := NewDefaultFactory()

	builtins := []string{"claude", "codex", "cursor", "gemini", "qwen", "iflow", "iflow-exec"}

	for _, cliType := range builtins {
		t.Run(cliType, func(t *testing.T) {
			cli, err := factory.NewCLI(cliType)
			if err != nil {
				t.Errorf("NewCLI(%s) failed: %v", cliType, err)
			}
			if cli == nil {
				t.Errorf("NewCLI(%s) returned nil", cliType)
			}
		})
	}
}

// TestFactory_NewCLI_InvalidType 测试无效CLI类型
func TestFactory_NewCLI_InvalidType(t *testing.T) {
	factory := NewDefaultFactory()

	_, err := factory.NewCLI("invalid-cli")
	if err == nil {
		t.Error("Expected error for invalid CLI type")
	}
}

// TestFactory_RegisterCLI 测试注册自定义CLI
func TestFactory_RegisterCLI(t *testing.T) {
	factory := NewDefaultFactory()

	// 创建mock CLI创建函数
	mockCreator := func() (CLIRunner, error) {
		return &mockCLIRunner{name: "custom", output: "test"}, nil
	}

	metadata := Metadata{
		Name:         "custom-cli",
		Version:      "1.0.0",
		Description:  "Custom CLI for testing",
		Author:       "Test Author",
		Tags:         []string{"test"},
		Capabilities: []string{"custom-feature"},
	}

	err := factory.RegisterCLI("custom-cli", mockCreator, metadata)
	if err != nil {
		t.Errorf("RegisterCLI failed: %v", err)
	}

	// 验证可以创建
	cli, err := factory.NewCLI("custom-cli")
	if err != nil {
		t.Errorf("NewCLI for registered custom CLI failed: %v", err)
	}
	if cli == nil {
		t.Error("Custom CLI is nil")
	}
}

// TestFactory_UnregisterCLI 测试卸载CLI
func TestFactory_UnregisterCLI(t *testing.T) {
	factory := NewDefaultFactory()

	mockCreator := func() (CLIRunner, error) {
		return &mockCLIRunner{name: "custom", output: "test"}, nil
	}

	metadata := Metadata{Name: "temp-cli", Version: "1.0.0"}
	factory.RegisterCLI("temp-cli", mockCreator, metadata)

	// 验证已注册
	if !factory.IsRegistered("temp-cli") {
		t.Error("CLI should be registered")
	}

	// 卸载
	err := factory.UnregisterCLI("temp-cli")
	if err != nil {
		t.Errorf("UnregisterCLI failed: %v", err)
	}

	// 验证已卸载
	if factory.IsRegistered("temp-cli") {
		t.Error("CLI should be unregistered")
	}
}

// TestFactory_IsRegistered 测试注册检查
func TestFactory_IsRegistered(t *testing.T) {
	factory := NewDefaultFactory()

	// 内置CLI应该已注册
	if !factory.IsRegistered("claude") {
		t.Error("Built-in CLI 'claude' should be registered")
	}

	// 未注册的CLI
	if factory.IsRegistered("nonexistent") {
		t.Error("Non-existent CLI should not be registered")
	}
}

// TestFactory_GetMetadata 测试获取元数据
func TestFactory_GetMetadata(t *testing.T) {
	factory := NewDefaultFactory()

	// 测试内置CLI元数据
	meta, err := factory.GetMetadata("iflow")
	if err != nil {
		t.Errorf("GetMetadata failed: %v", err)
	}

	if meta.Name != "iflow" {
		t.Errorf("Expected name 'iflow', got '%s'", meta.Name)
	}

	if meta.Version != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got '%s'", meta.Version)
	}
}

// TestFactory_ListAvailable 测试列出可用CLI
func TestFactory_ListAvailable(t *testing.T) {
	factory := NewDefaultFactory()

	available := factory.ListAvailable()

	// 应该包含所有内置CLI
	expectedBuiltins := []string{"claude", "codex", "cursor", "gemini", "qwen", "iflow", "iflow-exec"}
	for _, expected := range expectedBuiltins {
		found := false
		for _, cli := range available {
			if cli == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected CLI '%s' not in available list", expected)
		}
	}
}

// TestFactory_ListWithMetadata 测试带元数据的列表
func TestFactory_ListWithMetadata(t *testing.T) {
	factory := NewDefaultFactory()

	metadata := factory.ListWithMetadata()

	// 验证iflow元数据
	iflowMeta, ok := metadata["iflow"]
	if !ok {
		t.Error("iflow metadata not found")
	}

	if iflowMeta.Name != "iflow" {
		t.Errorf("Expected name 'iflow', got '%s'", iflowMeta.Name)
	}

	// 验证包含扩展能力
	hasExtension := false
	for _, cap := range iflowMeta.Capabilities {
		if cap == "extensions" {
			hasExtension = true
			break
		}
	}
	if !hasExtension {
		t.Error("iflow should have 'extensions' capability")
	}
}

// TestFactory_ValidateCLIConfig 测试配置验证
func TestFactory_ValidateCLIConfig(t *testing.T) {
	factory := NewDefaultFactory()

	// 测试有效配置
	validConfig := map[string]interface{}{
		"env": map[string]interface{}{
			"API_KEY": "test-key",
		},
	}

	err := factory.ValidateCLIConfig("claude", validConfig)
	if err != nil {
		t.Errorf("Valid config should pass: %v", err)
	}

	// 测试无效CLI
	err = factory.ValidateCLIConfig("nonexistent", nil)
	if err == nil {
		t.Error("Should fail for non-existent CLI")
	}

	// 测试无效环境变量
	invalidConfig := map[string]interface{}{
		"env": map[string]interface{}{
			"": "value",
		},
	}

	err = factory.ValidateCLIConfig("claude", invalidConfig)
	if err == nil {
		t.Error("Should fail for empty env key")
	}
}

// TestFactory_GetStats 测试统计信息
func TestFactory_GetStats(t *testing.T) {
	factory := NewDefaultFactory()

	// 注册一个mock CLI用于测试统计
	mockCreator := func() (CLIRunner, error) {
		return &mockCLIRunner{name: "stats-test", output: "test"}, nil
	}
	metadata := Metadata{Name: "stats-test", Version: "1.0.0"}
	factory.RegisterCLI("stats-test", mockCreator, metadata)

	// 创建CLI两次（统计NewCLI调用）
	cli1, err := factory.NewCLI("stats-test")
	if err != nil {
		t.Fatalf("Failed to create CLI 1: %v", err)
	}

	cli2, err := factory.NewCLI("stats-test")
	if err != nil {
		t.Fatalf("Failed to create CLI 2: %v", err)
	}

	// 执行几次Run以验证CLI正常工作
	opts := &RunOptions{Prompt: "test", Model: "stats-test"}
	cli1.Run(opts)
	cli2.Run(opts)

	// 获取统计（应该记录2次NewCLI调用）
	stats, err := factory.GetStats("stats-test")
	if err != nil {
		t.Errorf("GetStats failed: %v", err)
	}

	if stats.TotalCalls != 2 {
		t.Errorf("Expected 2 calls, got %d", stats.TotalCalls)
	}

	if stats.SuccessRate != 1.0 {
		t.Errorf("Expected 100%% success rate, got %f", stats.SuccessRate)
	}
}

// TestFactory_ConcurrentAccess 测试并发访问
func TestFactory_ConcurrentAccess(t *testing.T) {
	factory := NewDefaultFactory()

	done := make(chan bool, 20)

	// 并发注册
	for i := 0; i < 5; i++ {
		go func(id int) {
			name := "custom-" + string(rune('0'+id))
			metadata := Metadata{Name: name, Version: "1.0.0"}
			creator := func() (CLIRunner, error) {
				return &mockCLIRunner{name: name, output: "test"}, nil
			}
			factory.RegisterCLI(name, creator, metadata)
			done <- true
		}(i)
	}

	// 并发查询
	for i := 0; i < 5; i++ {
		go func() {
			factory.ListAvailable()
			done <- true
		}()
	}

	// 并发创建
	for i := 0; i < 5; i++ {
		go func() {
			factory.NewCLI("claude")
			done <- true
		}()
	}

	// 并发统计
	for i := 0; i < 5; i++ {
		go func() {
			factory.GetStats("claude")
			done <- true
		}()
	}

	// 等待所有完成
	for i := 0; i < 20; i++ {
		<-done
	}
}

// TestFactory_MetadataConsistency 测试元数据一致性
func TestFactory_MetadataConsistency(t *testing.T) {
	factory := NewDefaultFactory()

	// 验证ListAvailable和ListWithMetadata的一致性
	available := factory.ListAvailable()
	withMetadata := factory.ListWithMetadata()

	if len(available) < len(withMetadata) {
		t.Errorf("ListAvailable should have at least as many items as ListWithMetadata")
	}

	// 验证每个可用CLI都有元数据
	for _, cli := range available {
		if _, ok := withMetadata[cli]; !ok {
			t.Errorf("CLI '%s' in available list but not in metadata", cli)
		}
	}
}

// TestDefaultFactory_GlobalInstance 测试全局工厂实例
func TestDefaultFactory_GlobalInstance(t *testing.T) {
	// 测试全局函数
	factory := GetFactory()
	if factory == nil {
		t.Error("GetFactory() returned nil")
	}

	// 测试SetFactory
	testFactory := NewDefaultFactory()
	SetFactory(testFactory)

	// 验证已设置
	if GetFactory() != testFactory {
		t.Error("SetFactory() did not set the factory")
	}
}

// TestExtensionRegistry 测试扩展注册器
func TestExtensionRegistry(t *testing.T) {
	registry := NewExtensionRegistry()

	// 注册扩展
	metadata := Metadata{
		Name:         "test-ext",
		Version:      "1.0.0",
		Description:  "Test extension",
		Capabilities: []string{"test"},
	}

	creator := func() (CLIRunner, error) {
		return &mockCLIRunner{name: "test-ext", output: "test"}, nil
	}

	err := registry.Register("test-ext", creator, metadata)
	if err != nil {
		t.Errorf("Register failed: %v", err)
	}

	// 验证已注册
	if !registry.IsRegistered("test-ext") {
		t.Error("Extension should be registered")
	}

	// 获取扩展
	ext, err := registry.Get("test-ext")
	if err != nil {
		t.Errorf("Get failed: %v", err)
	}
	if ext == nil {
		t.Error("Extension should not be nil")
	}

	// 获取信息
	info, err := registry.GetInfo("test-ext")
	if err != nil {
		t.Errorf("GetInfo failed: %v", err)
	}
	if info.Name != "test-ext" {
		t.Errorf("Expected name 'test-ext', got '%s'", info.Name)
	}

	// 列出所有
	all := registry.ListAll()
	if len(all) == 0 {
		t.Error("ListAll should return extensions")
	}

	// 卸载
	err = registry.Unregister("test-ext")
	if err != nil {
		t.Errorf("Unregister failed: %v", err)
	}

	if registry.IsRegistered("test-ext") {
		t.Error("Extension should be unregistered")
	}
}

// TestExtensionRegistry_Concurrent 测试扩展注册器并发
func TestExtensionRegistry_Concurrent(t *testing.T) {
	registry := NewExtensionRegistry()

	done := make(chan bool, 20)

	// 并发注册
	for i := 0; i < 10; i++ {
		go func(id int) {
			name := "ext-" + string(rune('0'+id))
			metadata := Metadata{Name: name, Version: "1.0.0"}
			creator := func() (CLIRunner, error) {
				return &mockCLIRunner{name: name, output: "test"}, nil
			}
			registry.Register(name, creator, metadata)
			done <- true
		}(i)
	}

	// 并发读取
	for i := 0; i < 10; i++ {
		go func() {
			registry.ListAll()
			registry.IsRegistered("ext-0")
			done <- true
		}()
	}

	for i := 0; i < 20; i++ {
		<-done
	}
}

// TestNewCLIWithOptions 测试NewCLIWithOptions
func TestNewCLIWithOptions(t *testing.T) {
	factory := NewDefaultFactory()

	// 注册一个mock CLI
	mockCreator := func() (CLIRunner, error) {
		return &mockCLIRunner{name: "test-options", output: "test"}, nil
	}
	metadata := Metadata{Name: "test-options", Version: "1.0.0"}
	factory.RegisterCLI("test-options", mockCreator, metadata)

	// 设置全局工厂
	originalFactory := defaultFactory
	defer func() { defaultFactory = originalFactory }()
	defaultFactory = factory

	opts := &RunOptions{
		Prompt: "test",
		Model:  "test-options",
		Env: map[string]string{
			"TEST_VAR": "test-value",
		},
	}

	cli, err := NewCLIWithOptions("test-options", opts)
	if err != nil {
		t.Errorf("NewCLIWithOptions failed: %v", err)
	}

	if cli == nil {
		t.Error("CLI should not be nil")
	}
}

// TestSupportedCLIs 测试SupportedCLIs向后兼容
func TestSupportedCLIs(t *testing.T) {
	supported := SupportedCLIs()

	if len(supported) == 0 {
		t.Error("SupportedCLIs should return CLI types")
	}

	// 应该包含iflow
	found := false
	for _, cli := range supported {
		if cli == "iflow" {
			found = true
			break
		}
	}

	if !found {
		t.Error("SupportedCLIs should include 'iflow'")
	}
}
