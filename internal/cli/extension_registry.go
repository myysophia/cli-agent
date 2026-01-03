package cli

import (
	"fmt"
	"log"
	"sync"
	"time"
)

// ExtensionRegistry 扩展注册器，管理动态CLI扩展
type ExtensionRegistry struct {
	extensions map[string]*ExtensionEntry
	mu         sync.RWMutex
}

// ExtensionEntry 扩展条目
type ExtensionEntry struct {
	Creator    CLICreator
	Metadata   Metadata
	Instance   CLIRunner
	CreatedAt  time.Time
	LastUsed   time.Time
	ErrorCount int
}

// NewExtensionRegistry 创建扩展注册器
func NewExtensionRegistry() *ExtensionRegistry {
	return &ExtensionRegistry{
		extensions: make(map[string]*ExtensionEntry),
	}
}

// Register 注册一个CLI扩展
func (r *ExtensionRegistry) Register(name string, creator CLICreator, metadata Metadata) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.extensions[name]; exists {
		return fmt.Errorf("extension '%s' already registered", name)
	}

	r.extensions[name] = &ExtensionEntry{
		Creator:   creator,
		Metadata:  metadata,
		CreatedAt: time.Now(),
	}

	log.Printf("✅ [ExtensionRegistry] Registered extension: %s (v%s)", name, metadata.Version)
	return nil
}

// Unregister 卸载一个扩展
func (r *ExtensionRegistry) Unregister(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.extensions[name]; !exists {
		return fmt.Errorf("extension '%s' not found", name)
	}

	delete(r.extensions, name)
	log.Printf("🗑️  [ExtensionRegistry] Unregistered extension: %s", name)
	return nil
}

// Get 获取扩展实例（懒加载）
func (r *ExtensionRegistry) Get(name string) (CLIRunner, error) {
	r.mu.RLock()
	entry, exists := r.extensions[name]
	r.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("extension '%s' not registered", name)
	}

	// 如果实例已存在，直接返回
	if entry.Instance != nil {
		entry.LastUsed = time.Now()
		return entry.Instance, nil
	}

	// 懒加载实例
	r.mu.Lock()
	defer r.mu.Unlock()

	// 双重检查
	if entry.Instance != nil {
		return entry.Instance, nil
	}

	instance, err := entry.Creator()
	if err != nil {
		entry.ErrorCount++
		return nil, fmt.Errorf("failed to create extension '%s': %v", name, err)
	}

	entry.Instance = instance
	entry.LastUsed = time.Now()

	log.Printf("🚀 [ExtensionRegistry] Loaded extension: %s", name)
	return instance, nil
}

// IsRegistered 检查扩展是否已注册
func (r *ExtensionRegistry) IsRegistered(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, exists := r.extensions[name]
	return exists
}

// GetInfo 获取扩展信息
func (r *ExtensionRegistry) GetInfo(name string) (ExtensionInfo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	entry, exists := r.extensions[name]
	if !exists {
		return ExtensionInfo{}, fmt.Errorf("extension '%s' not found", name)
	}

	return ExtensionInfo{
		Name:         name,
		Version:      entry.Metadata.Version,
		Description:  entry.Metadata.Description,
		Capabilities: entry.Metadata.Capabilities,
		Enabled:      entry.Instance != nil,
		LastUsed:     entry.LastUsed,
		ErrorCount:   entry.ErrorCount,
	}, nil
}

// ListAll 列出所有已注册的扩展
func (r *ExtensionRegistry) ListAll() []ExtensionInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	infos := make([]ExtensionInfo, 0, len(r.extensions))
	for name, entry := range r.extensions {
		infos = append(infos, ExtensionInfo{
			Name:         name,
			Version:      entry.Metadata.Version,
			Description:  entry.Metadata.Description,
			Capabilities: entry.Metadata.Capabilities,
			Enabled:      entry.Instance != nil,
			LastUsed:     entry.LastUsed,
			ErrorCount:   entry.ErrorCount,
		})
	}
	return infos
}

// ListLoaded 列出已加载的扩展
func (r *ExtensionRegistry) ListLoaded() []ExtensionInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	infos := make([]ExtensionInfo, 0)
	for name, entry := range r.extensions {
		if entry.Instance != nil {
			infos = append(infos, ExtensionInfo{
				Name:         name,
				Version:      entry.Metadata.Version,
				Description:  entry.Metadata.Description,
				Capabilities: entry.Metadata.Capabilities,
				Enabled:      true,
				LastUsed:     entry.LastUsed,
				ErrorCount:   entry.ErrorCount,
			})
		}
	}
	return infos
}

// Unload 卸载但不删除扩展（释放内存）
func (r *ExtensionRegistry) Unload(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	entry, exists := r.extensions[name]
	if !exists {
		return fmt.Errorf("extension '%s' not found", name)
	}

	if entry.Instance != nil {
		// 如果实现了ExtensionCLI接口，调用Shutdown
		if extCLI, ok := entry.Instance.(ExtensionCLI); ok {
			if err := extCLI.Shutdown(); err != nil {
				log.Printf("⚠️  [ExtensionRegistry] Shutdown error for %s: %v", name, err)
			}
		}
		entry.Instance = nil
		log.Printf("💤 [ExtensionRegistry] Unloaded extension: %s", name)
	}

	return nil
}

// Clear 清空所有扩展
func (r *ExtensionRegistry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()

	for name, entry := range r.extensions {
		if entry.Instance != nil {
			if extCLI, ok := entry.Instance.(ExtensionCLI); ok {
				extCLI.Shutdown()
			}
		}
		delete(r.extensions, name)
	}

	log.Printf("🗑️  [ExtensionRegistry] Cleared all extensions")
}

// GetStats 获取扩展统计信息
func (r *ExtensionRegistry) GetStats(name string) (CLIStats, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	entry, exists := r.extensions[name]
	if !exists {
		return CLIStats{}, fmt.Errorf("extension '%s' not found", name)
	}

	stats := CLIStats{
		TotalCalls:  0, // 需要在CLI实现中跟踪
		LastUsed:    entry.LastUsed,
		ErrorCount:  entry.ErrorCount,
		SuccessRate: 0.0,
	}

	if entry.ErrorCount > 0 && entry.CreatedAt.Before(entry.LastUsed) {
		totalAttempts := int(time.Since(entry.CreatedAt).Minutes()) + entry.ErrorCount
		if totalAttempts > 0 {
			stats.SuccessRate = float64(totalAttempts-entry.ErrorCount) / float64(totalAttempts)
		}
	}

	return stats, nil
}
