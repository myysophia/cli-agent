package integration

import (
	"testing"

	"github.com/ninesun/projects/cli-agent/internal/cli"
)

// TestIntegration_Extension_Lifecycle 测试扩展生命周期
func TestIntegration_Extension_Lifecycle(t *testing.T) {
	// 注册
	extName := "lifecycle-test"
	metadata := cli.Metadata{
		Name:        extName,
		Version:     "1.0.0",
		Description: "Lifecycle test extension",
		Capabilities: []string{"lifecycle"},
	}

	initialized := false
	shutdownCalled := false

	creator := func() (cli.CLIRunner, error) {
		return &lifecycleCLI{
			name: extName,
			onInit: func() error {
				initialized = true
				return nil
			},
			onShutdown: func() error {
				shutdownCalled = true
				return nil
			},
		}, nil
	}

	err := cli.RegisterCLI(extName, creator, metadata)
	if err != nil {
		t.Fatalf("Failed to register: %v", err)
	}
	defer cli.UnregisterCLI(extName)

	// 获取并初始化
	factory := cli.GetFactory()
	cliInstance, err := factory.NewCLI(extName)
	if err != nil {
		t.Fatalf("Failed to create: %v", err)
	}

	if extCLI, ok := cliInstance.(cli.ExtensionCLI); ok {
		config := map[string]interface{}{"test": "value"}
		if err := extCLI.Initialize(config); err != nil {
			t.Errorf("Initialize failed: %v", err)
		}

		if !initialized {
			t.Error("Initialize callback not called")
		}

		// 关闭
		if err := extCLI.Shutdown(); err != nil {
			t.Errorf("Shutdown failed: %v", err)
		}

		if !shutdownCalled {
			t.Error("Shutdown callback not called")
		}
	} else {
		t.Error("CLI is not ExtensionCLI")
	}
}

// TestIntegration_Extension_Dependencies 测试扩展依赖
func TestIntegration_Extension_Dependencies(t *testing.T) {
	// 注册基础扩展
	baseExt := "base-ext"
	baseMetadata := cli.Metadata{
		Name:        baseExt,
		Version:     "1.0.0",
		Description: "Base extension",
		Capabilities: []string{"base"},
	}

	baseCreator := func() (cli.CLIRunner, error) {
		return &mockCLI{name: baseExt, output: "base"}, nil
	}

	err := cli.RegisterCLI(baseExt, baseCreator, baseMetadata)
	if err != nil {
		t.Fatalf("Failed to register base: %v", err)
	}
	defer cli.UnregisterCLI(baseExt)

	// 注册依赖扩展
	dependentExt := "dependent-ext"
	dependentMetadata := cli.Metadata{
		Name:        dependentExt,
		Version:     "1.0.0",
		Description: "Dependent extension",
		Capabilities: []string{"dependent", "requires-base"},
	}

	dependentCreator := func() (cli.CLIRunner, error) {
		// 检查基础扩展是否存在
		if !cli.IsRegistered(baseExt) {
			return nil, t.Errorf("Base extension not found")
		}
		return &mockCLI{name: dependentExt, output: "dependent"}, nil
	}

	err = cli.RegisterCLI(dependentExt, dependentCreator, dependentMetadata)
	if err != nil {
		t.Fatalf("Failed to register dependent: %v", err)
	}
	defer cli.UnregisterCLI(dependentExt)

	// 验证两者都可创建
	factory := cli.GetFactory()
	_, err = factory.NewCLI(baseExt)
	if err != nil {
		t.Errorf("Failed to create base: %v", err)
	}

	_, err = factory.NewCLI(dependentExt)
	if err != nil {
		t.Errorf("Failed to create dependent: %v", err)
	}
}

// TestIntegration_Extension_Conflicts 测试扩展冲突
func TestIntegration_Extension_Conflicts(t *testing.T) {
	// 注册第一个扩展
	ext1 := "conflict-test"
	metadata1 := cli.Metadata{
		Name:        ext1,
		Version:     "1.0.0",
		Description: "First version",
		Capabilities: []string{"feature"},
	}

	creator1 := func() (cli.CLIRunner, error) {
		return &mockCLI{name: ext1, output: "v1"}, nil
	}

	err := cli.RegisterCLI(ext1, creator1, metadata1)
	if err != nil {
		t.Fatalf("Failed to register first: %v", err)
	}
	defer cli.UnregisterCLI(ext1)

	// 尝试注册同名扩展（应该失败）
	metadata2 := cli.Metadata{
		Name:        ext1,
		Version:     "2.0.0",
		Description: "Second version",
		Capabilities: []string{"feature"},
	}

	creator2 := func() (cli.CLIRunner, error) {
		return &mockCLI{name: ext1, output: "v2"}, nil
	}

	err = cli.RegisterCLI(ext1, creator2, metadata2)
	if err == nil {
		t.Error("Should not allow duplicate extension names")
	}

	// 验证仍然是第一个版本
	factory := cli.GetFactory()
	cliInstance, err := factory.NewCLI(ext1)
	if err != nil {
		t.Fatalf("Failed to create: %v", err)
	}

	result, err := cliInstance.Run(&cli.RunOptions{Prompt: "test"})
	if err != nil {
		t.Errorf("Run failed: %v", err)
	}

	if result != "v1" {
		t.Errorf("Expected v1, got %s", result)
	}
}

// TestIntegration_Extension_Performance 测试扩展性能
func TestIntegration_Extension_Performance(t *testing.T) {
	extName := "perf-test"
	metadata := cli.Metadata{
		Name:        extName,
		Version:     "1.0.0",
		Description: "Performance test",
		Capabilities: []string{"perf"},
	}

	callCount := 0
	creator := func() (cli.CLIRunner, error) {
		callCount++
		return &mockCLI{name: extName, output: "perf"}, nil
	}

	err := cli.RegisterCLI(extName, creator, metadata)
	if err != nil {
		t.Fatalf("Failed to register: %v", err)
	}
	defer cli.UnregisterCLI(extName)

	factory := cli.GetFactory()

	// 多次创建（应该使用缓存实例）
	for i := 0; i < 5; i++ {
		_, err := factory.NewCLI(extName)
		if err != nil {
			t.Errorf("Create failed on iteration %d: %v", i, err)
		}
	}

	// 应该只调用一次创建函数（懒加载+缓存）
	if callCount != 1 {
		t.Errorf("Expected 1 creator call, got %d", callCount)
	}
}

// TestIntegration_Extension_Security 测试扩展安全
func TestIntegration_Extension_Security(t *testing.T) {
	extName := "security-test"
	metadata := cli.Metadata{
		Name:        extName,
		Version:     "1.0.0",
		Description: "Security test",
		Capabilities: []string{"security"},
	}

	// 创建会验证配置的扩展
	creator := func() (cli.CLIRunner, error) {
		return &securityCLI{name: extName}, nil
	}

	err := cli.RegisterCLI(extName, creator, metadata)
	if err != nil {
		t.Fatalf("Failed to register: %v", err)
	}
	defer cli.UnregisterCLI(extName)

	factory := cli.GetFactory()
	cliInstance, err := factory.NewCLI(extName)
	if err != nil {
		t.Fatalf("Failed to create: %v", err)
	}

	// 测试验证配置
	if extCLI, ok := cliInstance.(cli.ExtensionCLI); ok {
		err := extCLI.ValidateConfig()
		if err != nil {
			t.Errorf("ValidateConfig failed: %v", err)
		}
	}
}

// TestIntegration_Extension_Upgrade 测试扩展升级
func TestIntegration_Extension_Upgrade(t *testing.T) {
	extName := "upgrade-test"

	// 注册旧版本
	oldMetadata := cli.Metadata{
		Name:        extName,
		Version:     "1.0.0",
		Description: "Old version",
		Capabilities: []string{"old-feature"},
	}

	oldCreator := func() (cli.CLIRunner, error) {
		return &mockCLI{name: extName, output: "v1"}, nil
	}

	err := cli.RegisterCLI(extName, oldCreator, oldMetadata)
	if err != nil {
		t.Fatalf("Failed to register old: %v", err)
	}
	defer cli.UnregisterCLI(extName)

	// 获取信息
	factory := cli.GetFactory()
	metadata, err := factory.GetMetadata(extName)
	if err != nil {
		t.Fatalf("Failed to get metadata: %v", err)
	}

	if metadata.Version != "1.0.0" {
		t.Errorf("Expected version 1.0.0, got %s", metadata.Version)
	}

	// 卸载
	err = cli.UnregisterCLI(extName)
	if err != nil {
		t.Fatalf("Failed to unregister: %v", err)
	}

	// 注册新版本
	newMetadata := cli.Metadata{
		Name:        extName,
		Version:     "2.0.0",
		Description: "New version",
		Capabilities: []string{"new-feature"},
	}

	newCreator := func() (cli.CLIRunner, error) {
		return &mockCLI{name: extName, output: "v2"}, nil
	}

	err = cli.RegisterCLI(extName, newCreator, newMetadata)
	if err != nil {
		t.Fatalf("Failed to register new: %v", err)
	}

	// 验证新版本
	metadata, err = factory.GetMetadata(extName)
	if err != nil {
		t.Fatalf("Failed to get new metadata: %v", err)
	}

	if metadata.Version != "2.0.0" {
		t.Errorf("Expected version 2.0.0, got %s", metadata.Version)
	}
}

// TestIntegration_Extension_MultiLoad 测试多扩展加载
func TestIntegration_Extension_MultiLoad(t *testing.T) {
	// 注册多个扩展
	extensions := []struct {
		name     string
		version  string
		output   string
	}{
		{"ext-a", "1.0.0", "output-a"},
		{"ext-b", "1.0.0", "output-b"},
		{"ext-c", "1.0.0", "output-c"},
	}

	for _, ext := range extensions {
		metadata := cli.Metadata{
			Name:        ext.name,
			Version:     ext.version,
			Description: "Multi-load test",
			Capabilities: []string{"multi"},
		}

		creator := func(out string) func() (cli.CLIRunner, error) {
			return func() (cli.CLIRunner, error) {
				return &mockCLI{name: ext.name, output: out}, nil
			}
		}(ext.output)

		err := cli.RegisterCLI(ext.name, creator, metadata)
		if err != nil {
			t.Fatalf("Failed to register %s: %v", ext.name, err)
		}
		defer cli.UnregisterCLI(ext.name)
	}

	factory := cli.GetFactory()

	// 验证所有扩展都可创建
	for _, ext := range extensions {
		cliInstance, err := factory.NewCLI(ext.name)
		if err != nil {
			t.Errorf("Failed to create %s: %v", ext.name, err)
			continue
		}

		result, err := cliInstance.Run(&cli.RunOptions{Prompt: "test"})
		if err != nil {
			t.Errorf("Run %s failed: %v", ext.name, err)
			continue
		}

		if result != ext.output {
			t.Errorf("Expected %s, got %s", ext.output, result)
		}
	}

	// 验证列表
	available := factory.ListAvailable()
	for _, ext := range extensions {
		found := false
		for _, a := range available {
			if a == ext.name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Extension %s not in available list", ext.name)
		}
	}
}

// Helper types for integration tests

type lifecycleCLI struct {
	name       string
	onInit     func() error
	onShutdown func() error
}

func (l *lifecycleCLI) Name() string {
	return l.name
}

func (l *lifecycleCLI) Run(opts *cli.RunOptions) (string, error) {
	return "lifecycle", nil
}

func (l *lifecycleCLI) GetVersion() string {
	return "1.0.0"
}

func (l *lifecycleCLI) GetCapabilities() []string {
	return []string{"lifecycle"}
}

func (l *lifecycleCLI) ValidateConfig() error {
	return nil
}

func (l *lifecycleCLI) Initialize(config map[string]interface{}) error {
	if l.onInit != nil {
		return l.onInit()
	}
	return nil
}

func (l *lifecycleCLI) Shutdown() error {
	if l.onShutdown != nil {
		return l.onShutdown()
	}
	return nil
}

type securityCLI struct {
	name string
}

func (s *securityCLI) Name() string {
	return s.name
}

func (s *securityCLI) Run(opts *cli.RunOptions) (string, error) {
	return "secure", nil
}

func (s *securityCLI) GetVersion() string {
	return "1.0.0"
}

func (s *securityCLI) GetCapabilities() []string {
	return []string{"security"}
}

func (s *securityCLI) ValidateConfig() error {
	// 模拟安全验证
	return nil
}

func (s *securityCLI) Initialize(config map[string]interface{}) error {
	return nil
}

func (s *securityCLI) Shutdown() error {
	return nil
}
