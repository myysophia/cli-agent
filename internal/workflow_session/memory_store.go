package workflow_session

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type memoryEntry struct {
	sessionID string
	expiresAt time.Time
}

// MemoryMappingStore 提供进程内映射存储，用于 Redis 不可用时兜底。
type MemoryMappingStore struct {
	mu    sync.RWMutex
	items map[string]memoryEntry
}

func NewMemoryMappingStore() *MemoryMappingStore {
	return &MemoryMappingStore{
		items: make(map[string]memoryEntry),
	}
}

func (s *MemoryMappingStore) Get(_ context.Context, workflowRunID string) (string, bool, error) {
	if workflowRunID == "" {
		return "", false, fmt.Errorf("workflow run id is required")
	}
	s.mu.RLock()
	entry, ok := s.items[workflowRunID]
	s.mu.RUnlock()
	if !ok {
		return "", false, nil
	}
	if !entry.expiresAt.IsZero() && time.Now().After(entry.expiresAt) {
		s.mu.Lock()
		delete(s.items, workflowRunID)
		s.mu.Unlock()
		return "", false, nil
	}
	return entry.sessionID, true, nil
}

func (s *MemoryMappingStore) Set(_ context.Context, workflowRunID string, sessionID string, ttl time.Duration) error {
	if workflowRunID == "" {
		return fmt.Errorf("workflow run id is required")
	}
	if sessionID == "" {
		return fmt.Errorf("session id is required")
	}
	if ttl <= 0 {
		return fmt.Errorf("ttl must be positive")
	}
	s.mu.Lock()
	s.items[workflowRunID] = memoryEntry{
		sessionID: sessionID,
		expiresAt: time.Now().Add(ttl),
	}
	s.mu.Unlock()
	return nil
}

func (s *MemoryMappingStore) Touch(_ context.Context, workflowRunID string, ttl time.Duration) error {
	if workflowRunID == "" {
		return fmt.Errorf("workflow run id is required")
	}
	if ttl <= 0 {
		return fmt.Errorf("ttl must be positive")
	}
	s.mu.Lock()
	entry, ok := s.items[workflowRunID]
	if ok {
		entry.expiresAt = time.Now().Add(ttl)
		s.items[workflowRunID] = entry
	}
	s.mu.Unlock()
	return nil
}
