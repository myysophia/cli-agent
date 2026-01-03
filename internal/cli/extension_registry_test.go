package cli

import (
	"errors"
	"testing"
	"time"
)

// TestExtensionRegistry_Register 测试注册扩展
func TestExtensionRegistry_Register(t *testing.T) {
	registry := NewExtensionRegistry()

	metadata := Metadata{
		Name:        "test-extension",
		Version:     "1.0.0",
		Description: "Test extension",
		Author:      "Test Author",
		Tags:        []string{"test", "demo"},
		Capabilities: []string{"custom-feature"},
	}

	creator := func() (CLIRunner, error) {
		return &mockCLIRunner{name: "test-extension", output: "test"}, nil
	}

	err := registry.Register("test-extension", creator, metadata)
	if err != nil {
		t.Errorf("Register failed: %v", err)
	}

	// 验证已注册
	if !registry.IsRegistered("test-extension") {
		t.Error("Extension should be registered")
	}
}

// TestExtensionRegistry_RegisterDuplicate 测试重复注册
func TestExtensionRegistry_RegisterDuplicate(t *testing.T) {
	registry := NewExtensionRegistry()

	metadata := Metadata{Name: "dup", Version: "1.0.0"}
	creator := func() (CLIRunner, error) {
		return &mockCLIRunner{name: "dup", output: "test"}, nil
	}

	// 第一次注册
	err := registry.Register("dup", creator, metadata)
	if err != nil {
		t.Errorf("First register failed: %v", err)
	}

	// 第二次注册应该失败
	err = registry.Register("dup", creator, metadata)
	if err == nil {
		t.Error("Duplicate register should fail")
	}
}

// TestExtensionRegistry_Unregister 测试卸载扩展
func TestExtensionRegistry_Unregister(t *testing.T) {
	registry := NewExtensionRegistry()

	metadata := Metadata{Name: "temp", Version: "1.0.0"}
	creator := func() (CLIRunner, error) {
		return &mockCLIRunner{name: "temp", output: "test"}, nil
	}

	registry.Register("temp", creator, metadata)

	// 卸载
	err := registry.Unregister("temp")
	if err != nil {
		t.Errorf("Unregister failed: %v", err)
	}

	if registry.IsRegistered("temp") {
		t.Error("Extension should be unregistered")
	}
}

// TestExtensionRegistry_UnregisterNonExistent 测试卸载不存在的扩展
func TestExtensionRegistry_UnregisterNonExistent(t *testing.T) {
	registry := NewExtensionRegistry()

	err := registry.Unregister("nonexistent")
	if err == nil {
		t.Error("Unregister non-existent should fail")
	}
}

// TestExtensionRegistry_Get 测试获取扩展（懒加载）
func TestExtensionRegistry_Get(t *testing.T) {
	registry := NewExtensionRegistry()

	callCount := 0
	metadata := Metadata{Name: "lazy", Version: "1.0.0"}
	creator := func() (CLIRunner, error) {
		callCount++
		return &mockCLIRunner{name: "lazy", output: "test"}, nil
	}

	registry.Register("lazy", creator, metadata)

	// 第一次获取 - 应该调用creator
	ext1, err := registry.Get("lazy")
	if err != nil {
		t.Errorf("Get failed: %v", err)
	}
	if ext1 == nil {
		t.Error("Extension should not be nil")
	}
	if callCount != 1 {
		t.Errorf("Expected 1 creator call, got %d", callCount)
	}

	// 第二次获取 - 应该返回缓存的实例
	ext2, err := registry.Get("lazy")
	if err != nil {
		t.Errorf("Get failed: %v", err)
	}
	if callCount != 1 {
		t.Errorf("Expected creator to be called only once, got %d", callCount)
	}

	// 验证是同一个实例
	if ext1 != ext2 {
		t.Error("Should return same instance")
	}
}

// TestExtensionRegistry_Get_NonExistent 测试获取不存在的扩展
func TestExtensionRegistry_Get_NonExistent(t *testing.T) {
	registry := NewExtensionRegistry()

	_, err := registry.Get("nonexistent")
	if err == nil {
		t.Error("Get non-existent should fail")
	}
}

// TestExtensionRegistry_Get_CreatorError 测试创建器错误
func TestExtensionRegistry_Get_CreatorError(t *testing.T) {
	registry := NewExtensionRegistry()

	metadata := Metadata{Name: "error", Version: "1.0.0"}
	creator := func() (CLIRunner, error) {
		return nil, errors.New("creation failed")
	}

	registry.Register("error", creator, metadata)

	_, err := registry.Get("error")
	if err == nil {
		t.Error("Get should fail when creator fails")
	}
}

// TestExtensionRegistry_GetInfo 测试获取扩展信息
func TestExtensionRegistry_GetInfo(t *testing.T) {
	registry := NewExtensionRegistry()

	metadata := Metadata{
		Name:         "info-test",
		Version:      "2.5.1",
		Description:  "Testing info retrieval",
		Author:       "Info Author",
		Tags:         []string{"info", "test"},
		Capabilities: []string{"feature1", "feature2"},
	}

	creator := func() (CLIRunner, error) {
		return &mockCLIRunner{name: "info-test", output: "test"}, nil
	}

	registry.Register("info-test", creator, metadata)

	// 获取信息（未加载）
	info, err := registry.GetInfo("info-test")
	if err != nil {
		t.Errorf("GetInfo failed: %v", err)
	}

	if info.Name != "info-test" {
		t.Errorf("Expected name 'info-test', got '%s'", info.Name)
	}
	if info.Version != "2.5.1" {
		t.Errorf("Expected version '2.5.1', got '%s'", info.Version)
	}
	if info.Description != "Testing info retrieval" {
		t.Errorf("Expected description 'Testing info retrieval', got '%s'", info.Description)
	}
	if len(info.Capabilities) != 2 {
		t.Errorf("Expected 2 capabilities, got %d", len(info.Capabilities))
	}
	if info.Enabled {
		t.Error("Should not be enabled before loading")
	}

	// 加载后获取信息
	registry.Get("info-test")
	info, err = registry.GetInfo("info-test")
	if err != nil {
		t.Errorf("GetInfo after load failed: %v", err)
	}
	if !info.Enabled {
		t.Error("Should be enabled after loading")
	}
}

// TestExtensionRegistry_ListAll 测试列出所有扩展
func TestExtensionRegistry_ListAll(t *testing.T) {
	registry := NewExtensionRegistry()

	// 注册多个扩展
	for i := 0; i < 3; i++ {
		name := "ext-" + string(rune('0'+i))
		metadata := Metadata{Name: name, Version: "1.0.0"}
		creator := func() (CLIRunner, error) {
			return &mockCLIRunner{name: name, output: "test"}, nil
		}
		registry.Register(name, creator, metadata)
	}

	all := registry.ListAll()
	if len(all) != 3 {
		t.Errorf("Expected 3 extensions, got %d", len(all))
	}
}

// TestExtensionRegistry_ListLoaded 测试列出已加载的扩展
func TestExtensionRegistry_ListLoaded(t *testing.T) {
	registry := NewExtensionRegistry()

	// 注册两个扩展
	metadata1 := Metadata{Name: "loaded", Version: "1.0.0"}
	creator1 := func() (CLIRunner, error) {
		return &mockCLIRunner{name: "loaded", output: "test"}, nil
	}
	registry.Register("loaded", creator1, metadata1)

	metadata2 := Metadata{Name: "not-loaded", Version: "1.0.0"}
	creator2 := func() (CLIRunner, error) {
		return &mockCLIRunner{name: "not-loaded", output: "test"}, nil
	}
	registry.Register("not-loaded", creator2, metadata2)

	// 只加载第一个
	registry.Get("loaded")

	loaded := registry.ListLoaded()
	if len(loaded) != 1 {
		t.Errorf("Expected 1 loaded extension, got %d", len(loaded))
	}
	if loaded[0].Name != "loaded" {
		t.Errorf("Expected 'loaded', got '%s'", loaded[0].Name)
	}
}

// TestExtensionRegistry_Unload 测试卸载（释放内存）
func TestExtensionRegistry_Unload(t *testing.T) {
	registry := NewExtensionRegistry()

	metadata := Metadata{Name: "unload-test", Version: "1.0.0"}
	creator := func() (CLIRunner, error) {
		return &mockCLIRunner{name: "unload-test", output: "test"}, nil
	}

	registry.Register("unload-test", creator, metadata)
	registry.Get("unload-test") // 加载

	// 卸载
	err := registry.Unload("unload-test")
	if err != nil {
		t.Errorf("Unload failed: %v", err)
	}

	// 仍然注册，但未加载
	if !registry.IsRegistered("unload-test") {
		t.Error("Should still be registered")
	}

	loaded := registry.ListLoaded()
	if len(loaded) != 0 {
		t.Error("Should have no loaded extensions")
	}
}

// TestExtensionRegistry_Clear 测试清空所有
func TestExtensionRegistry_Clear(t *testing.T) {
	registry := NewExtensionRegistry()

	// 注册多个
	for i := 0; i < 3; i++ {
		name := "clear-" + string(rune('0'+i))
		metadata := Metadata{Name: name, Version: "1.0.0"}
		creator := func() (CLIRunner, error) {
			return &mockCLIRunner{name: name, output: "test"}, nil
		}
		registry.Register(name, creator, metadata)
		registry.Get(name) // 加载
	}

	registry.Clear()

	all := registry.ListAll()
	if len(all) != 0 {
		t.Errorf("Expected 0 extensions after clear, got %d", len(all))
	}
}

// TestExtensionRegistry_GetStats 测试统计信息
func TestExtensionRegistry_GetStats(t *testing.T) {
	registry := NewExtensionRegistry()

	metadata := Metadata{Name: "stats-test", Version: "1.0.0"}
	creator := func() (CLIRunner, error) {
		return &mockCLIRunner{name: "stats-test", output: "test"}, nil
	}

	registry.Register("stats-test", creator, metadata)

	// 获取统计（未加载）
	stats, err := registry.GetStats("stats-test")
	if err != nil {
		t.Errorf("GetStats failed: %v", err)
	}

	if stats.TotalCalls != 0 {
		t.Errorf("Expected 0 calls, got %d", stats.TotalCalls)
	}

	// 加载后获取统计
	registry.Get("stats-test")
	time.Sleep(10 * time.Millisecond) // 确保时间差
	registry.Get("stats-test")        // 再次获取以更新LastUsed

	stats, err = registry.GetStats("stats-test")
	if err != nil {
		t.Errorf("GetStats after load failed: %v", err)
	}

	if stats.LastUsed.IsZero() {
		t.Error("LastUsed should be set after loading")
	}
}

// TestExtensionRegistry_ConcurrentOperations 测试并发操作
func TestExtensionRegistry_ConcurrentOperations(t *testing.T) {
	registry := NewExtensionRegistry()

	done := make(chan bool, 50)

	// 并发注册
	for i := 0; i < 10; i++ {
		go func(id int) {
			name := "concurrent-" + string(rune('0'+id))
			metadata := Metadata{Name: name, Version: "1.0.0"}
			creator := func() (CLIRunner, error) {
				return &mockCLIRunner{name: name, output: "test"}, nil
			}
			registry.Register(name, creator, metadata)
			done <- true
		}(i)
	}

	// 并发获取
	for i := 0; i < 10; i++ {
		go func(id int) {
			name := "concurrent-" + string(rune('0'+id))
			registry.Get(name)
			done <- true
		}(i)
	}

	// 并发查询
	for i := 0; i < 10; i++ {
		go func() {
			registry.ListAll()
			registry.ListLoaded()
			done <- true
		}()
	}

	// 并发信息获取
	for i := 0; i < 10; i++ {
		go func(id int) {
			name := "concurrent-" + string(rune('0'+id))
			registry.GetInfo(name)
			done <- true
		}(i)
	}

	// 并发统计
	for i := 0; i < 10; i++ {
		go func(id int) {
			name := "concurrent-" + string(rune('0'+id))
			registry.GetStats(name)
			done <- true
		}(i)
	}

	// 等待所有完成
	for i := 0; i < 50; i++ {
		<-done
	}
}

// TestExtensionRegistry_ErrorCount 测试错误计数
func TestExtensionRegistry_ErrorCount(t *testing.T) {
	registry := NewExtensionRegistry()

	metadata := Metadata{Name: "error-count", Version: "1.0.0"}
	creator := func() (CLIRunner, error) {
		return nil, errors.New("creation error")
	}

	registry.Register("error-count", creator, metadata)

	// 多次尝试获取（都会失败）
	for i := 0; i < 3; i++ {
		registry.Get("error-count")
	}

	info, err := registry.GetInfo("error-count")
	if err != nil {
		t.Errorf("GetInfo failed: %v", err)
	}

	if info.ErrorCount != 3 {
		t.Errorf("Expected 3 errors, got %d", info.ErrorCount)
	}
}

// TestExtensionRegistry_LastUsed 测试最后使用时间
func TestExtensionRegistry_LastUsed(t *testing.T) {
	registry := NewExtensionRegistry()

	metadata := Metadata{Name: "last-used", Version: "1.0.0"}
	creator := func() (CLIRunner, error) {
		return &mockCLIRunner{name: "last-used", output: "test"}, nil
	}

	registry.Register("last-used", creator, metadata)

	// 获取前
	info, _ := registry.GetInfo("last-used")
	if !info.LastUsed.IsZero() {
		t.Error("LastUsed should be zero before first use")
	}

	// 使用
	time.Sleep(10 * time.Millisecond)
	before := time.Now()
	registry.Get("last-used")
	after := time.Now()

	// 验证时间更新
	info, _ = registry.GetInfo("last-used")
	if info.LastUsed.Before(before) || info.LastUsed.After(after) {
		t.Error("LastUsed not updated correctly")
	}
}

// TestExtensionRegistry_MultipleVersions 测试多版本管理
func TestExtensionRegistry_MultipleVersions(t *testing.T) {
	registry := NewExtensionRegistry()

	// 注册同一名称的不同版本（应该失败）
	metadata1 := Metadata{Name: "versioned", Version: "1.0.0"}
	creator1 := func() (CLIRunner, error) {
		return &mockCLIRunner{name: "versioned", output: "v1"}, nil
	}

	err := registry.Register("versioned", creator1, metadata1)
	if err != nil {
		t.Errorf("First register failed: %v", err)
	}

	// 尝试注册同名不同版本
	metadata2 := Metadata{Name: "versioned", Version: "2.0.0"}
	creator2 := func() (CLIRunner, error) {
		return &mockCLIRunner{name: "versioned", output: "v2"}, nil
	}

	err = registry.Register("versioned", creator2, metadata2)
	if err == nil {
		t.Error("Should not allow duplicate names")
	}

	// 验证版本信息
	info, err := registry.GetInfo("versioned")
	if err != nil {
		t.Errorf("GetInfo failed: %v", err)
	}
	if info.Version != "1.0.0" {
		t.Errorf("Expected version 1.0.0, got %s", info.Version)
	}
}
