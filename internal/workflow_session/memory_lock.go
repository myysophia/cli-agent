package workflow_session

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type memoryLockEntry struct {
	token     string
	expiresAt time.Time
}

// MemoryLocker 提供进程内锁，用于 Redis 不可用时兜底。
type MemoryLocker struct {
	mu    sync.Mutex
	locks map[string]memoryLockEntry
}

func NewMemoryLocker() *MemoryLocker {
	return &MemoryLocker{
		locks: make(map[string]memoryLockEntry),
	}
}

func (l *MemoryLocker) TryLock(_ context.Context, workflowRunID string, ttl time.Duration) (LockHandle, bool, error) {
	if workflowRunID == "" {
		return nil, false, fmt.Errorf("workflow run id is required")
	}
	if ttl <= 0 {
		return nil, false, fmt.Errorf("ttl must be positive")
	}
	token, err := newLockToken()
	if err != nil {
		return nil, false, err
	}
	now := time.Now()

	l.mu.Lock()
	defer l.mu.Unlock()

	if entry, ok := l.locks[workflowRunID]; ok {
		if entry.expiresAt.IsZero() || now.Before(entry.expiresAt) {
			return nil, false, nil
		}
		delete(l.locks, workflowRunID)
	}

	l.locks[workflowRunID] = memoryLockEntry{
		token:     token,
		expiresAt: now.Add(ttl),
	}

	return &memoryLockHandle{
		locker: l,
		key:    workflowRunID,
		token:  token,
	}, true, nil
}

type memoryLockHandle struct {
	locker *MemoryLocker
	key    string
	token  string
}

func (h *memoryLockHandle) Unlock(_ context.Context) error {
	if h == nil || h.locker == nil {
		return nil
	}
	h.locker.mu.Lock()
	defer h.locker.mu.Unlock()

	entry, ok := h.locker.locks[h.key]
	if !ok {
		return nil
	}
	if entry.token != h.token {
		return fmt.Errorf("lock not owned or already released")
	}
	delete(h.locker.locks, h.key)
	return nil
}
